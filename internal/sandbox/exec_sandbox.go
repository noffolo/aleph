package sandbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/registry"
	"github.com/ff3300/aleph-v2/internal/repository"
)

type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    string
	Metrics  string
}

type SandboxManager interface {
	ExecuteTool(ctx context.Context, toolID string, input map[string]interface{}) (ExecutionResult, error)
	RunSkill(ctx context.Context, skillID string, input map[string]interface{}) (ExecutionResult, error)
}

type ExecSandbox struct {
	logger    *slog.Logger
	regMgr    *registry.DuckDBRegistry
	metaRepo  *repository.MetadataRepository
	pythonCmd string
	goCmd     string
}

func NewExecSandbox(l *slog.Logger, r *registry.DuckDBRegistry, meta *repository.MetadataRepository, py, goCmd string) *ExecSandbox {
	return &ExecSandbox{logger: l, regMgr: r, metaRepo: meta, pythonCmd: py, goCmd: goCmd}
}

func (s *ExecSandbox) ExecuteTool(ctx context.Context, toolID string, input map[string]interface{}) (ExecutionResult, error) {
	if s.metaRepo == nil {
		return ExecutionResult{Error: "metadata repository not available", ExitCode: -1}, nil
	}

	var code string
	code, err := s.metaRepo.GetToolCode(toolID)
	if err != nil {
		return ExecutionResult{Error: "tool not found: " + err.Error(), ExitCode: -1}, nil
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "aleph-sandbox-*")
	if err != nil {
		return ExecutionResult{Error: "failed to create temp dir: " + err.Error(), ExitCode: -1}, nil
	}
	defer os.RemoveAll(tmpDir)

	inputJSON, _ := json.Marshal(input)

	if strings.HasPrefix(code, "# python") || strings.HasPrefix(code, "#!/usr/bin/env python") {
		tmpFile := filepath.Join(tmpDir, "tool.py")
		if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
			return ExecutionResult{Error: "failed to write tool file: " + err.Error(), ExitCode: -1}, nil
		}
		cmd := exec.CommandContext(execCtx, s.pythonCmd, tmpFile)
		cmd.Dir = tmpDir
	cmd.Env = []string{
			"ALEPH_INPUT=" + string(inputJSON),
			"PATH=/usr/bin:/bin",
			"HOME=" + tmpDir,
		}
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return ExecutionResult{Error: err.Error(), Stderr: stderr.String(), ExitCode: -1}, nil
			}
		}
		return ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, nil
	}

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return ExecutionResult{Error: "failed to write tool file: " + err.Error(), ExitCode: -1}, nil
	}
	binPath := filepath.Join(tmpDir, "tool_bin")
	buildCmd := exec.CommandContext(execCtx, s.goCmd, "build", "-o", binPath, ".")
	buildCmd.Dir = tmpDir
	var buildStderr strings.Builder
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		return ExecutionResult{Error: "build failed: " + err.Error(), Stderr: buildStderr.String(), ExitCode: -1}, nil
	}
	runCmd := exec.CommandContext(execCtx, binPath)
	runCmd.Dir = tmpDir
	runCmd.Env = []string{
		"ALEPH_INPUT=" + string(inputJSON),
		"PATH=/usr/bin:/bin",
		"HOME=" + tmpDir,
	}
	var stdout, stderr strings.Builder
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr
	err = runCmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ExecutionResult{Error: err.Error(), Stderr: stderr.String(), ExitCode: -1}, nil
		}
	}
	return ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, nil
}

func (s *ExecSandbox) RunSkill(ctx context.Context, skillID string, input map[string]interface{}) (ExecutionResult, error) {
	if s.metaRepo == nil {
		return ExecutionResult{Error: "metadata repository not available", ExitCode: -1}, nil
	}

	var toolIDsJSON string
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
			return ExecutionResult{Error: "tool execution failed at step " + string(rune('0'+i)) + ": " + err.Error(), ExitCode: -1}, nil
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
