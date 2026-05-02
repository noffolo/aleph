package genesis

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var blockedGoImports = map[string]string{
	"os":             "os: filesystem manipulation",
	"os/exec":        "os/exec: arbitrary command execution",
	"plugin":         "plugin: dynamic library loading",
	"runtime":        "runtime: low-level runtime access",
	"net":            "net: network access",
	"net/http":       "net/http: HTTP client/server",
	"net/url":        "net/url: SSRF vector",
	"net/smtp":       "net/smtp: email sending",
	"net/rpc":        "net/rpc: remote procedure calls",
	"syscall":        "syscall: low-level system access",
	"unsafe":         "unsafe: memory unsafety",
	"reflect":        "reflect: runtime type manipulation",
	"text/template":  "text/template: code injection via templates",
	"html/template":  "html/template: code injection via templates",
}

// blockedGoImportPrefixes defines import path prefixes that are always
// blocked. Any import starting with one of these prefixes is rejected.
// This covers wildcard packages like crypto/*, encoding/*, debug/*,
// internal/*, and net/* sub-packages not already in blockedGoImports.
var blockedGoImportPrefixes = map[string]string{
	"crypto/":   "crypto/*: cryptographic operations",
	"encoding/": "encoding/*: data encoding/decoding (potential obfuscation)",
	"debug/":    "debug/*: debugging and runtime introspection",
	"internal/": "internal/*: internal packages",
	"net/":      "net/*: network sub-package",
}

var blockedGoCalls = map[string]string{
	"os.Remove":    "file deletion",
	"os.RemoveAll": "recursive directory deletion",
	"os.Chmod":     "permission manipulation",
	"os.Rename":    "file race condition",
	"os.Symlink":   "symlink escape",
	"os.Create":    "file creation",
	"net.Listen":   "network listener",
	"net.Dial":     "network dialer",
	"plugin.Open":  "dynamic library loading",
	"runtime.Goexit":     "goroutine termination",
	"runtime.Stack":       "goroutine stack introspection",
	"runtime.Callers":     "call stack introspection",
	"runtime.Breakpoint":  "trigger debugger breakpoint",
	"os.Exit":            "hard process termination",
}

type SandboxResult struct {
	Passed          bool          `json:"passed"`
	Warnings        []string      `json:"warnings,omitempty"`
	RiskScore       float64       `json:"risk_score"`
	BlockedPatterns []string      `json:"blocked_patterns,omitempty"`
	Duration        time.Duration `json:"duration"`
}

type Sandbox struct {
	timeout  time.Duration
	execPath string
}

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

	blocked := s.checkDangerousPatternsAST(ctx, code)
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

func (s *Sandbox) checkDangerousPatternsAST(ctx context.Context, code string) []string {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	var blocked []string

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return s.checkDangerousPatternsFallback(code)
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if reason, isBlocked := blockedGoImports[importPath]; isBlocked {
			blocked = append(blocked, fmt.Sprintf("%s (%s)", importPath, reason))
		}
		for prefix, reason := range blockedGoImportPrefixes {
			if strings.HasPrefix(importPath, prefix) {
				if _, alreadyBlocked := blockedGoImports[importPath]; !alreadyBlocked {
					blocked = append(blocked, fmt.Sprintf("%s (%s)", importPath, reason))
				}
				break
			}
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		fullCall := ident.Name + "." + selExpr.Sel.Name
		if reason, isBlocked := blockedGoCalls[fullCall]; isBlocked {
			blocked = append(blocked, fmt.Sprintf("%s (%s)", fullCall, reason))
		}

		return true
	})

	return blocked
}

func (s *Sandbox) checkDangerousPatternsFallback(code string) []string {
	var blocked []string
	for pattern, reason := range blockedGoImports {
		if strings.Contains(code, pattern) {
			blocked = append(blocked, fmt.Sprintf("%s (%s)", pattern, reason))
		}
	}
	for pattern, reason := range blockedGoCalls {
		if strings.Contains(code, pattern) {
			blocked = append(blocked, fmt.Sprintf("%s (%s)", pattern, reason))
		}
	}
	return blocked
}

var obfuscationPatterns = []struct {
	name    string
	pattern *regexp.Regexp
	score   float64
}{
	{"base64_decode", regexp.MustCompile(`encoding/base64`), 0.3},
	{"base64_decode_call", regexp.MustCompile(`base64\.(?:StdEncoding|URLEncoding)\.Decode`), 0.5},
	{"hex_decode", regexp.MustCompile(`encoding/hex`), 0.2},
	{"hex_decode_call", regexp.MustCompile(`hex\.(?:DecodeString|Decode)`), 0.4},
	{"exec_command_dynamic", regexp.MustCompile(`exec\.Command\(.*\+`), 0.5},
	{"exec_command_variable", regexp.MustCompile(`exec\.Command\([a-zA-Z_]\w*\)`), 0.4},
	{"import_concat", regexp.MustCompile(`import\s*\(\s*[^)]*\+`), 0.6},
	{"reflect_value_call", regexp.MustCompile(`reflect\.Value\.\w+Call`), 0.4},
	{"plugin_open", regexp.MustCompile(`plugin\.Open`), 0.7},
	{"getenv_command", regexp.MustCompile(`os\.Getenv\(.*\).*exec`), 0.5},
	{"runtime_mmap", regexp.MustCompile(`syscall\.Mmap|Mmap`), 0.6},
	{"shell_redirect", regexp.MustCompile(`/proc/self|/dev/stdin`), 0.7},
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
