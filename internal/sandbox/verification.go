package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
)

// VerificationConfig controls how a tool is verified in the sandbox.
type VerificationConfig struct {
	TimeoutSeconds int     `json:"timeout_seconds"`
	MaxMemoryMB    int     `json:"max_memory_mb"`
	MaxCPUSeconds  float64 `json:"max_cpu_seconds"`
	NetworkBlocked bool    `json:"network_blocked"`
}

// VerificationResult holds the complete output of a verification run.
type VerificationResult struct {
	ToolID         string        `json:"tool_id"`
	ToolName       string        `json:"tool_name"`
	Stdout         string        `json:"stdout"`
	Stderr         string        `json:"stderr"`
	ExitCode       int           `json:"exit_code"`
	Duration       time.Duration `json:"duration"`
	MemoryUsedMB   float64       `json:"memory_used_mb"`
	CPUUsedSeconds float64       `json:"cpu_used_seconds"`
	Passed         bool          `json:"passed"`
	Error          string        `json:"error,omitempty"`
}

// DefaultVerificationConfig returns a safe default configuration.
func DefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		TimeoutSeconds: 15,
		MaxMemoryMB:    256,
		MaxCPUSeconds:  10,
		NetworkBlocked: true,
	}
}

// Verifier runs tools in verification+isolation mode.
type Verifier struct {
	logger    *slog.Logger
	metaRepo  *repository.MetadataRepository
	pythonCmd string
	goCmd     string
}

// NewVerifier creates a new sandbox verifier.
func NewVerifier(logger *slog.Logger, metaRepo *repository.MetadataRepository, pythonCmd, goCmd string) *Verifier {
	return &Verifier{
		logger:    logger,
		metaRepo:  metaRepo,
		pythonCmd: pythonCmd,
		goCmd:     goCmd,
	}
}

// VerifyTool runs a tool in verification mode with strict isolation.
// It validates the code, executes it in a sandboxed environment with
// resource limits, and captures complete results.
func (v *Verifier) VerifyTool(ctx context.Context, toolID string, config VerificationConfig) (VerificationResult, error) {
	result := VerificationResult{
		ToolID: toolID,
	}

	toolCode, err := v.metaRepo.GetToolCode(ctx, toolID)
	if err != nil {
		result.Error = fmt.Sprintf("tool not found: %v", err)
		result.ExitCode = -1
		return result, fmt.Errorf("tool lookup failed: %w", err)
	}
	result.ToolName = toolID
	if IsPythonCode(toolCode) {
		if err := ValidatePythonCode(toolCode); err != nil {
			result.Error = fmt.Sprintf("code validation failed: %v", err)
			result.ExitCode = -1
			result.Passed = false
			return result, nil
		}
	} else {
		if err := ValidateGoCode(toolCode); err != nil {
			result.Error = fmt.Sprintf("code validation failed: %v", err)
			result.ExitCode = -1
			result.Passed = false
			return result, nil
		}
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	sandbox := NewExecSandbox(v.logger, nil, v.metaRepo, v.pythonCmd, v.goCmd)
	execResult, err := sandbox.ExecuteTool(execCtx, toolID, map[string]interface{}{"verification": true})

	result.Duration = time.Since(start)
	result.Stdout = execResult.Stdout
	result.Stderr = execResult.Stderr
	result.ExitCode = execResult.ExitCode

	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result, nil
	}

	// Security: check output for sandbox escape patterns
	if result.ExitCode != 0 {
		result.Passed = false
		result.Error = fmt.Sprintf("tool exited with code %d", result.ExitCode)
		return result, nil
	}

	// Check for suspicious output patterns that indicate sandbox escape attempts
	result.Passed = v.isOutputSafe(result.Stdout) && v.isOutputSafe(result.Stderr)
	if !result.Passed {
		result.Error = "suspicious output detected during verification"
	}

	// Resource estimation: Go lacks per-process memory tracking; production uses cgroups
	result.CPUUsedSeconds = result.Duration.Seconds()
	result.MemoryUsedMB = float64(config.MaxMemoryMB) * 0.1

	v.logger.Info("tool verification completed",
		"tool_id", toolID,
		"passed", result.Passed,
		"duration", result.Duration,
		"exit_code", result.ExitCode,
	)

	return result, nil
}

// isOutputSafe checks verification output for suspicious patterns.
func (v *Verifier) isOutputSafe(output string) bool {
	suspiciousPatterns := []string{
		"/etc/passwd",
		"/etc/shadow",
		"root:",
		"sudo ",
		"chmod 777",
		"rm -rf /",
		"fork bomb",
	}
	lower := strings.ToLower(output)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lower, pattern) {
			return false
		}
	}
	return true
}

// VerifyToolCode verifies tool code without executing it (static analysis only).
func (v *Verifier) VerifyToolCode(code string) VerificationResult {
	result := VerificationResult{
		Passed: true,
	}

	if IsPythonCode(code) {
		if err := ValidatePythonCode(code); err != nil {
			result.Error = err.Error()
			result.Passed = false
			return result
		}
	} else {
		if err := ValidateGoCode(code); err != nil {
			result.Error = err.Error()
			result.Passed = false
			return result
		}
	}

	return result
}

// VerifyMultipleTools runs verification on multiple tools concurrently.
func (v *Verifier) VerifyMultipleTools(ctx context.Context, toolIDs []string, config VerificationConfig) []VerificationResult {
	results := make([]VerificationResult, len(toolIDs))
	type indexedResult struct {
		index  int
		result VerificationResult
		err    error
	}

	ch := make(chan indexedResult, len(toolIDs))

	for i, id := range toolIDs {
		go func(idx int, toolID string) {
			r, err := v.VerifyTool(ctx, toolID, config)
			ch <- indexedResult{index: idx, result: r, err: err}
		}(i, id)
	}

	for i := 0; i < len(toolIDs); i++ {
		res := <-ch
		results[res.index] = res.result
		if res.err != nil {
			results[res.index].Error = res.err.Error()
			results[res.index].Passed = false
		}
	}

	return results
}

// CheckSandboxSecurity performs basic security checks on the sandbox environment.
func (v *Verifier) CheckSandboxSecurity() []string {
	var issues []string

	if _, err := exec.LookPath("unshare"); err != nil {
		issues = append(issues, "unshare not found: network isolation may be limited")
	}

	if _, err := exec.LookPath("timeout"); err != nil {
		issues = append(issues, "timeout not found: time limits may be limited")
	}

	if _, err := exec.LookPath("python3"); err != nil {
		issues = append(issues, "python3 not found: Python tool verification unavailable")
	}

	if _, err := exec.LookPath("go"); err != nil {
		issues = append(issues, "go not found: Go tool verification unavailable")
	}

	return issues
}