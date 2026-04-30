package genesis

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SandboxResult provides a structured validation report for tool code.
//
// IMPORTANT: This sandbox provides heuristic isolation, NOT container-guaranteed
// isolation. It uses layered pattern matching and subprocess execution with
// timeout, but a determined adversary can bypass these checks via indirection,
// dynamic code generation, or exploiting the host Go runtime. For production
// workloads requiring strong isolation, use gVisor, Firecracker, or a full
// container runtime.
type SandboxResult struct {
	Passed          bool      `json:"passed"`
	Warnings        []string  `json:"warnings,omitempty"`
	RiskScore       float64   `json:"risk_score"`
	BlockedPatterns []string  `json:"blocked_patterns,omitempty"`
	Duration        time.Duration `json:"duration"`
}

// Sandbox validates tool code through layered heuristic checks.
type Sandbox struct {
	timeout    time.Duration
	execPath   string // path to "go" binary for subprocess validation
}

// NewSandbox creates a Sandbox with the given timeout for subprocess validation.
func NewSandbox(timeout time.Duration) *Sandbox {
	return &Sandbox{
		timeout:  timeout,
		execPath: "go",
	}
}

func (s *Sandbox) Validate(ctx context.Context, suggestion Suggestion) (*SandboxResult, error) {
	start := time.Now()

	if suggestion.Code == "" {
		return &SandboxResult{
			Passed:    true,
			Duration:  time.Since(start),
			RiskScore: 0,
		}, nil
	}

	result, err := s.validateCode(ctx, suggestion.Code)
	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(start)
	return result, nil
}

func (s *Sandbox) validateCode(ctx context.Context, code string) (*SandboxResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result := &SandboxResult{
		Passed:    true,
		RiskScore: 0,
	}

	blocked := s.checkDangerousPatterns(ctx, code)
	if len(blocked) > 0 {
		result.Passed = false
		result.BlockedPatterns = blocked
		result.RiskScore = clampRisk(float64(len(blocked)) * 0.2)
		return result, nil
	}

	warnings, obfuscationScore := s.detectObfuscation(code)
	result.Warnings = warnings
	result.RiskScore += obfuscationScore

	if obfuscationScore >= 0.8 {
		result.Passed = false
		result.BlockedPatterns = append(result.BlockedPatterns, "obfuscation_detected")
		return result, nil
	}

	subprocessPassed, subprocessWarnings, err := s.validateInSubprocess(ctx, code)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("subprocess unavailable: %v", err))
		result.RiskScore += 0.1
	} else if !subprocessPassed {
		result.Passed = false
		result.Warnings = append(result.Warnings, subprocessWarnings...)
		result.RiskScore = clampRisk(result.RiskScore + 0.3)
		return result, nil
	} else {
		result.Warnings = append(result.Warnings, subprocessWarnings...)
	}

	result.RiskScore = clampRisk(result.RiskScore)
	if result.Passed && result.RiskScore >= 0.6 {
		result.Passed = false
	}

	return result, nil
}

func (s *Sandbox) checkDangerousPatterns(ctx context.Context, code string) []string {
	dangerous := []string{
		"os/exec", "syscall", "unsafe", "reflect",
		"os.Remove", "os.RemoveAll", "os.Chmod",
		"net.Listen", "net.Dial",
	}

	var blocked []string
	for _, pattern := range dangerous {
		select {
		case <-ctx.Done():
			return blocked
		default:
		}
		if strings.Contains(code, pattern) {
			blocked = append(blocked, pattern)
		}
	}
	return blocked
}

// obfuscationPatterns matches common encoding/indirection tricks used to bypass
// string-based pattern matching.
var obfuscationPatterns = []struct {
	name    string
	pattern *regexp.Regexp
	score   float64
}{
	// Base64 decode attempts
	{"base64_decode", regexp.MustCompile(`encoding/base64`), 0.3},
	{"base64_decode_call", regexp.MustCompile(`base64\.(?:StdEncoding|URLEncoding)\.Decode`), 0.5},
	// Hex encoding/decoding
	{"hex_decode", regexp.MustCompile(`encoding/hex`), 0.2},
	{"hex_decode_call", regexp.MustCompile(`hex\.(?:DecodeString|Decode)`), 0.4},
	// Eval-like dynamic execution patterns
	{"exec_command_dynamic", regexp.MustCompile(`exec\.Command\(.*\+`), 0.5},
	{"exec_command_variable", regexp.MustCompile(`exec\.Command\([a-zA-Z_]\w*\)`), 0.4},
	// String concatenation to construct dangerous imports
	{"import_concat", regexp.MustCompile(`import\s*\(\s*[^)]*\+`), 0.6},
	// Reflect-based dynamic calls
	{"reflect_value_call", regexp.MustCompile(`reflect\.Value\.\w+Call`), 0.4},
	// Plugin dynamic loading
	{"plugin_open", regexp.MustCompile(`plugin\.Open`), 0.7},
	// os.Getenv used to construct commands
	{"getenv_command", regexp.MustCompile(`os\.Getenv\(.*\).*exec`), 0.5},
	// Runtime manipulation
	{"runtime_mmap", regexp.MustCompile(`syscall\.Mmap|Mmap`), 0.6},
	// Indirect process spawning via /proc or shell
	{"shell_redirect", regexp.MustCompile(`/proc/self|/dev/stdin`), 0.7},
	// CGo escape hatch
	{"cgo_escape", regexp.MustCompile(`(?:#include|import\s*"C")`), 0.6},
}

func (s *Sandbox) detectObfuscation(code string) ([]string, float64) {
	var warnings []string
	var totalScore float64

	for _, op := range obfuscationPatterns {
		if op.pattern.MatchString(code) {
			warnings = append(warnings, fmt.Sprintf("obfuscation pattern detected: %s (risk +%.1f)", op.name, op.score))
			totalScore += op.score
		}
	}

	// Heuristic: unusually high ratio of string literals to code may indicate packing
	stringLiteralCount := strings.Count(code, `"`) / 2
	lines := strings.Count(code, "\n") + 1
	if lines > 0 && stringLiteralCount > 3*lines {
		warnings = append(warnings, "unusually high string literal density (possible packed payload)")
		totalScore += 0.3
	}

	return warnings, totalScore
}

func (s *Sandbox) validateInSubprocess(ctx context.Context, code string) (bool, []string, error) {
	if _, err := exec.LookPath(s.execPath); err != nil {
		return true, []string{"subprocess skipped: go toolchain not found"}, nil
	}

	tmpDir, cleanup, err := createTempModule(code)
	if err != nil {
		return false, []string{fmt.Sprintf("temp module creation failed: %v", err)}, nil
	}
	defer cleanup()

	vetCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(vetCtx, s.execPath, "vet", "./...")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if vetCtx.Err() == context.DeadlineExceeded {
			return false, []string{"subprocess timed out"}, nil
		}
		return false, []string{fmt.Sprintf("go vet failed: %s", firstNLines(string(output), 3))}, nil
	}

	vetOutput := strings.TrimSpace(string(output))
	if vetOutput != "" {
		return true, []string{fmt.Sprintf("vet warnings: %s", firstNLines(vetOutput, 3))}, nil
	}

	return true, nil, nil
}

func createTempModule(code string) (dir string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "aleph-sandbox-*")
	if err != nil {
		return "", nil, fmt.Errorf("mkdir temp: %w", err)
	}

	cleanup = func() { os.RemoveAll(tmpDir) }

	goMod := "module sandbox\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write go.mod: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(code), 0o644); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write main.go: %w", err)
	}

	return tmpDir, cleanup, nil
}

func firstNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + "..."
}

func clampRisk(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	return v
}