package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
)

var ErrContainerUnavailable = errors.New("container runtime unavailable: docker daemon not reachable")

const (
	DefaultPythonImage  = "python:3.12-alpine"
	DefaultGoImage      = "golang:1.24-alpine"
	defaultMemoryMB     = 256
	defaultCPUCores     = 0.5
)

type ContainerConfig struct {
	PythonImage    string
	GoImage        string
	DefaultTimeout time.Duration
	MaxMemoryMB    int
	MaxCPUSeconds  float64
	NetworkBlocked bool
}

func DefaultContainerConfig() ContainerConfig {
	return ContainerConfig{
		PythonImage:    DefaultPythonImage,
		GoImage:        DefaultGoImage,
		DefaultTimeout: 30 * time.Second,
		MaxMemoryMB:    defaultMemoryMB,
		MaxCPUSeconds:  defaultCPUCores * float64(30*time.Second/time.Second),
		NetworkBlocked: true,
	}
}

func dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

type ContainerSandbox struct {
	logger      *slog.Logger
	regMgr      *registry.DuckDBRegistry
	metaRepo    *repository.MetadataRepository
	config      ContainerConfig
	hasRunsc    bool
	runscCached bool
}

func NewContainerSandbox(l *slog.Logger, r *registry.DuckDBRegistry, meta *repository.MetadataRepository, config ContainerConfig, _ SandboxManager) *ContainerSandbox {
	cs := &ContainerSandbox{
		logger:   l,
		regMgr:   r,
		metaRepo: meta,
		config:   config,
	}
	cs.detectRunsc()
	return cs
}

func (s *ContainerSandbox) detectRunsc() {
	// context.TODO: detectRunsc is called during NewContainerSandbox at init time,
	// before any request context is available. The 5s timeout is sufficient for
	// the docker info probe.
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Runtimes}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Debug("docker info failed, assuming no gVisor runsc", "error", err)
		s.runscCached = true
		s.hasRunsc = false
		return
	}

	if strings.Contains(string(output), "runsc") {
		s.hasRunsc = true
		s.logger.Info("gVisor runsc runtime detected, will use for container isolation")
	} else {
		s.hasRunsc = false
		s.logger.Warn("gVisor runsc not available, using plain Docker isolation (less secure)")
	}
	s.runscCached = true
}

type HealthCheckResult struct {
	Available    bool
	HasRunsc     bool
	DockerPingOK bool
	Error        string
}

func (s *ContainerSandbox) HealthCheck(ctx context.Context) HealthCheckResult {
	result := HealthCheckResult{}

	if !dockerAvailable() {
		result.Error = "docker binary not found in PATH"
		return result
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(pingCtx, "docker", "info", "--format", "{{.ServerVersion}}")
	_, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("docker daemon unreachable: %v", err)
		return result
	}

	result.DockerPingOK = true
	result.Available = true

	if s.runscCached {
		result.HasRunsc = s.hasRunsc
	} else {
		runtimeCtx, runtimeCancel := context.WithTimeout(ctx, 5*time.Second)
		defer runtimeCancel()

		runtimeCmd := exec.CommandContext(runtimeCtx, "docker", "info", "--format", "{{.Runtimes}}")
		runtimeOutput, runtimeErr := runtimeCmd.CombinedOutput()
		if runtimeErr == nil && strings.Contains(string(runtimeOutput), "runsc") {
			result.HasRunsc = true
		}
	}

	return result
}

func (s *ContainerSandbox) ExecuteTool(ctx context.Context, toolID string, input map[string]interface{}) (ExecutionResult, error) {
	if s.metaRepo == nil {
		return ExecutionResult{Error: "metadata repository not available", ExitCode: -1}, nil
	}

	code, err := s.metaRepo.GetToolCode(ctx, toolID)
	if err != nil {
		return ExecutionResult{Error: "tool not found: " + err.Error(), ExitCode: -1}, nil
	}

	if IsPythonCode(code) {
		if err := ValidatePythonCode(code); err != nil {
			return ExecutionResult{Error: "python validation failed: " + err.Error(), ExitCode: -1}, nil
		}
	} else {
		if err := ValidateGoCode(code); err != nil {
			return ExecutionResult{Error: "go validation failed: " + err.Error(), ExitCode: -1}, nil
		}
	}

	if !dockerAvailable() {
		s.logger.Error("docker unavailable — container isolation required for tool execution, refusing fallback", "tool_id", toolID)
		return ExecutionResult{Error: ErrContainerUnavailable.Error(), ExitCode: -1}, nil
	}

	return s.executeInContainer(ctx, code, input)
}

func (s *ContainerSandbox) executeInContainer(ctx context.Context, code string, input map[string]interface{}) (ExecutionResult, error) {
	tmpDir, err := os.MkdirTemp("", "aleph-container-*")
	if err != nil {
		return ExecutionResult{Error: "failed to create temp dir: " + err.Error(), ExitCode: -1}, nil
	}
	defer os.RemoveAll(tmpDir)
	if err := os.Chmod(tmpDir, 0700); err != nil {
		return ExecutionResult{Error: "failed to set temp dir permissions: " + err.Error(), ExitCode: -1}, nil
	}

	isPython := IsPythonCode(code)
	image := s.config.GoImage
	containerCmd := "go build -o /tmp/tool /workspace/main.go && /tmp/tool"
	fileName := "main.go"

	if isPython {
		image = s.config.PythonImage
		containerCmd = "python /workspace/tool.py"
		fileName = "tool.py"
	}

	tmpFile := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(tmpFile, []byte(code), 0600); err != nil {
		return ExecutionResult{Error: "failed to write tool file: " + err.Error(), ExitCode: -1}, nil
	}

	inputJSON, _ := json.Marshal(input)

	timeout := s.config.DefaultTimeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"run", "--rm",
		"--network", "none",
		"--security-opt=no-new-privileges:true",
		"--cap-drop=ALL",
		"--read-only",
		"--tmpfs", "/tmp:size=100m",
	}

	if s.hasRunsc {
		args = append(args, "--runtime=runsc")
		s.logger.Debug("using gVisor runsc runtime for container", "image", image)
	}

	args = append(args,
		"--memory", fmt.Sprintf("%dm", s.config.MaxMemoryMB),
		"--memory-swap", fmt.Sprintf("%dm", s.config.MaxMemoryMB),
	)

	if s.config.MaxCPUSeconds > 0 {
		cpus := s.config.MaxCPUSeconds / float64(s.config.DefaultTimeout.Seconds())
		if cpus < 0.1 {
			cpus = 0.1
		}
		args = append(args, "--cpus", fmt.Sprintf("%.2f", cpus))
	}

	args = append(args,
		"-v", tmpDir+":/workspace:ro",
		"-w", "/workspace",
		"-e", "ALEPH_INPUT="+string(inputJSON),
		"-e", "HOME=/tmp",
		image,
		"sh", "-c", containerCmd,
	)

	cmd := exec.CommandContext(execCtx, "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ExecutionResult{
				Error:    "docker execution failed: " + err.Error(),
				Stderr:   stderr.String(),
				ExitCode: -1,
			}, nil
		}
	}

	runtimeLabel := "docker"
	if s.hasRunsc {
		runtimeLabel = "runsc"
	}

	return ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Metrics: fmt.Sprintf("container_time=%.2fs memory_limit=%dM cpu_limit=%.2f runtime=%s",
			duration.Seconds(), s.config.MaxMemoryMB, s.config.MaxCPUSeconds, runtimeLabel),
	}, nil
}

func (s *ContainerSandbox) RunSkill(ctx context.Context, skillID string, input map[string]interface{}) (ExecutionResult, error) {
	if s.metaRepo == nil {
		return ExecutionResult{Error: "metadata repository not available", ExitCode: -1}, nil
	}

	toolIDsJSON, err := s.metaRepo.GetSkillToolIDs(skillID)
	if err != nil {
		return ExecutionResult{Error: "skill not found: " + err.Error(), ExitCode: -1}, nil
	}

	var toolIDs []string
	if err := json.Unmarshal([]byte(toolIDsJSON), &toolIDs); err != nil {
		return ExecutionResult{Error: "invalid tool_ids JSON: " + err.Error(), ExitCode: -1}, nil
	}

	var result ExecutionResult
	currentInput := input
	for i, tid := range toolIDs {
		result, err = s.ExecuteTool(ctx, tid, currentInput)
		if err != nil {
			return ExecutionResult{Error: fmt.Sprintf("tool execution failed at step_%d: %s", i, err.Error()), ExitCode: -1}, nil
		}
		if result.ExitCode != 0 {
			return result, nil
		}
		var nextInput map[string]interface{}
		if err := json.Unmarshal([]byte(result.Stdout), &nextInput); err != nil {
			nextInput = map[string]interface{}{"stdout": result.Stdout}
		}
		currentInput = nextInput
	}
	return result, nil
}
