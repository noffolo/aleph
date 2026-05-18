package sandbox

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultVerificationConfig(t *testing.T) {
	cfg := DefaultVerificationConfig()
	if cfg.TimeoutSeconds != 15 {
		t.Errorf("TimeoutSeconds = %d, want 15", cfg.TimeoutSeconds)
	}
	if cfg.MaxMemoryMB != 256 {
		t.Errorf("MaxMemoryMB = %d, want 256", cfg.MaxMemoryMB)
	}
	if cfg.MaxCPUSeconds != 10 {
		t.Errorf("MaxCPUSeconds = %f, want 10", cfg.MaxCPUSeconds)
	}
	if !cfg.NetworkBlocked {
		t.Error("NetworkBlocked = false, want true")
	}
}

func TestNewVerifier(t *testing.T) {
	logger := slog.Default()
	v := NewVerifier(logger, nil, "python3", "go")
	if v == nil {
		t.Fatal("NewVerifier returned nil")
	}
	if v.logger != logger {
		t.Error("logger not set correctly")
	}
	if v.pythonCmd != "python3" {
		t.Errorf("pythonCmd = %q, want %q", v.pythonCmd, "python3")
	}
	if v.goCmd != "go" {
		t.Errorf("goCmd = %q, want %q", v.goCmd, "go")
	}
	if v.metaRepo != nil {
		t.Error("metaRepo should be nil when passed nil")
	}
}

func TestVerifier_WithSandbox(t *testing.T) {
	logger := slog.Default()
	v := NewVerifier(logger, nil, "python3", "go")

	v2 := v.WithSandbox(nil)
	if v2 == nil {
		t.Fatal("WithSandbox returned nil")
	}
	if v2 == v {
		t.Error("WithSandbox should return a new copy, not the same pointer")
	}
	if v2.sandbox != nil {
		t.Error("sandbox should be nil when passed nil")
	}
	if v.sandbox != nil {
		t.Error("WithSandbox mutated the original verifier")
	}
}

func TestVerifier_WithSandbox_NonNil(t *testing.T) {
	logger := slog.Default()
	v := NewVerifier(logger, nil, "python3", "go")
	mockSB := NewExecSandbox(logger, nil, nil, "python3", "go")

	v2 := v.WithSandbox(mockSB)
	if v2.sandbox != mockSB {
		t.Error("sandbox not set correctly on copy")
	}
	if v.sandbox != nil {
		t.Error("WithSandbox mutated the original verifier's sandbox")
	}
}

func TestVerifier_VerifyToolCode_GoNoImport(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode(`package main
func main() { println("hello") }`)
	if !result.Passed {
		t.Errorf("expected Passed=true, got false: %s", result.Error)
	}
}

func TestVerifier_VerifyToolCode_BlockedPythonSubprocess(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode(`# python
import subprocess
subprocess.run(["ls"])`)
	if result.Passed {
		t.Error("expected Passed=false for subprocess import")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for subprocess import")
	}
}

func TestVerifier_VerifyToolCode_BlockedPythonSocket(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode(`# python
import socket
s = socket.socket()`)
	if result.Passed {
		t.Error("expected Passed=false for socket import")
	}
}

func TestVerifier_VerifyToolCode_BlockedPythonExec(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode("# python\nexec(\"import os\")")
	if result.Passed {
		t.Error("expected Passed=false for exec() call")
	}
}

func TestVerifier_VerifyToolCode_BlockedPythonEval(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode("# python\nx = eval(\"2+2\")")
	if result.Passed {
		t.Error("expected Passed=false for eval() call")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for eval() call")
	}
}

func TestVerifier_VerifyToolCode_BlockedPythonImaplib(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode(`# python
import imaplib
mail = imaplib.IMAP4_SSL("host")`)
	if result.Passed {
		t.Error("expected Passed=false for imaplib import")
	}
}

func TestVerifier_VerifyToolCode_EmptyCode(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode("")
	if !result.Passed {
		t.Errorf("empty code should pass static analysis, got: %s", result.Error)
	}
}

func TestVerifier_VerifyToolCode_PythonMultiLineBypass(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	result := v.VerifyToolCode("# python\nev\\\nal(\"2+2\")")
	if result.Passed {
		t.Error("expected Passed=false for multi-line eval bypass")
	}
}

func TestVerifier_isOutputSafe_Clean(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	safeOutputs := []string{"hello world", "result: 42", "processing complete", "", "ok"}
	for _, out := range safeOutputs {
		if !v.isOutputSafe(out) {
			t.Errorf("isOutputSafe(%q) = false, want true", out)
		}
	}
}

func TestVerifier_isOutputSafe_Suspicious(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	tests := []struct {
		name   string
		output string
	}{
		{"passwd", "reading /etc/passwd"},
		{"shadow", "cat /etc/shadow"},
		{"sudo", "running sudo command"},
		{"chmod", "chmod 777 file"},
		{"rm_rf", "rm -rf / important"},
		{"fork_bomb", "launching fork bomb"},
		{"root_colon", "root:password_hash"},
		{"uppercase", "SUDO rm -rf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if v.isOutputSafe(tc.output) {
				t.Errorf("isOutputSafe(%q) = true, want false", tc.output)
			}
		})
	}
}

func TestVerifier_isOutputSafe_CaseInsensitive(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	if v.isOutputSafe("/ETC/PASSWD") {
		t.Error("isOutputSafe should detect /etc/passwd case-insensitively")
	}
	if v.isOutputSafe("SUDO rm -rf /") {
		t.Error("isOutputSafe should detect sudo case-insensitively")
	}
}

func TestVerifier_isOutputSafe_MultiplePatterns(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	output := strings.Repeat("rm -rf / and chmod 777 ", 100)
	if v.isOutputSafe(output) {
		t.Error("isOutputSafe should detect patterns in large output")
	}
}

func TestVerifier_CheckSandboxSecurity(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	issues := v.CheckSandboxSecurity()
	if len(issues) == 0 {
		t.Log("CheckSandboxSecurity: all security tools found, no issues reported")
	}
	for _, issue := range issues {
		if issue == "" {
			t.Error("security issues should not contain empty strings")
		}
	}
}

func TestVerifier_CheckSandboxSecurity_KnownPrefixes(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	issues := v.CheckSandboxSecurity()
	knownPrefixes := []string{"unshare", "timeout", "python3", "go"}
	for _, issue := range issues {
		found := false
		for _, prefix := range knownPrefixes {
			if strings.Contains(issue, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected security issue: %q", issue)
		}
	}
}

func TestVerifier_VerifyTool_ExpiredContext(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := v.VerifyTool(ctx, "tool-1", DefaultVerificationConfig())
	if err != nil {
		t.Errorf("VerifyTool with expired context should not return error: %v", err)
	}
	if result.Passed {
		t.Error("result should not pass when context is expired")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if !strings.Contains(result.Error, "context expired") {
		t.Errorf("Error should mention context expired, got: %q", result.Error)
	}
	if result.ToolID != "tool-1" {
		t.Errorf("ToolID = %q, want %q", result.ToolID, "tool-1")
	}
}

func TestVerifier_VerifyTool_DeadlineExceeded(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	result, err := v.VerifyTool(ctx, "tool-2", DefaultVerificationConfig())
	if err != nil {
		t.Errorf("VerifyTool with past deadline should not return error: %v", err)
	}
	if result.Passed {
		t.Error("result should not pass when deadline is exceeded")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("Error should be non-empty for expired context")
	}
}

func TestVerifier_VerifyTool_ZeroTimeout(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := VerificationConfig{
		TimeoutSeconds: 0,
		MaxMemoryMB:    64,
		MaxCPUSeconds:  5,
		NetworkBlocked: true,
	}

	result, err := v.VerifyTool(ctx, "tool-3", cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("result should not pass when context is expired")
	}
}

func TestVerifier_VerifyMultipleTools_EmptyList(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	results := v.VerifyMultipleTools(context.Background(), []string{}, DefaultVerificationConfig())
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty list, got %d", len(results))
	}
}

func TestVerifier_VerifyMultipleTools_ExpiredContext(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	toolIDs := []string{"a", "b", "c", "d", "e"}
	results := v.VerifyMultipleTools(ctx, toolIDs, DefaultVerificationConfig())

	if len(results) != len(toolIDs) {
		t.Errorf("expected %d results, got %d", len(toolIDs), len(results))
	}
	for i, r := range results {
		if r.Passed {
			t.Errorf("result[%d] should not pass with expired context", i)
		}
		if r.ExitCode != -1 {
			t.Errorf("result[%d] ExitCode = %d, want -1", i, r.ExitCode)
		}
		if !strings.Contains(r.Error, "context expired") {
			t.Errorf("result[%d] Error should mention context expired, got: %q", i, r.Error)
		}
	}
}

func TestVerifier_VerifyMultipleTools_ResultsOrder(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	toolIDs := []string{"first", "second", "third", "fourth"}
	results := v.VerifyMultipleTools(ctx, toolIDs, DefaultVerificationConfig())

	for i, r := range results {
		if r.ToolID != toolIDs[i] {
			t.Errorf("result[%d] ToolID = %q, want %q", i, r.ToolID, toolIDs[i])
		}
	}
}

func TestVerifier_VerifyMultipleTools_SingleTool(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := v.VerifyMultipleTools(ctx, []string{"only"}, DefaultVerificationConfig())
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolID != "only" {
		t.Errorf("ToolID = %q, want %q", results[0].ToolID, "only")
	}
}

func TestVerifier_VerifyMultipleTools_NoSharedState(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := v.VerifyMultipleTools(ctx, []string{"a", "b"}, DefaultVerificationConfig())
			if len(results) != 2 {
				t.Errorf("parallel caller got %d results", len(results))
			}
		}()
	}
	wg.Wait()
}

func TestVerificationResult_Defaults(t *testing.T) {
	var r VerificationResult
	if r.ToolID != "" {
		t.Errorf("zero-value ToolID should be empty, got %q", r.ToolID)
	}
	if r.ToolName != "" {
		t.Errorf("zero-value ToolName should be empty, got %q", r.ToolName)
	}
	if r.Stdout != "" {
		t.Errorf("zero-value Stdout should be empty, got %q", r.Stdout)
	}
	if r.Stderr != "" {
		t.Errorf("zero-value Stderr should be empty, got %q", r.Stderr)
	}
	if r.ExitCode != 0 {
		t.Errorf("zero-value ExitCode should be 0, got %d", r.ExitCode)
	}
	if r.Duration != 0 {
		t.Errorf("zero-value Duration should be 0, got %v", r.Duration)
	}
	if r.MemoryUsedMB != 0 {
		t.Errorf("zero-value MemoryUsedMB should be 0, got %f", r.MemoryUsedMB)
	}
	if r.CPUUsedSeconds != 0 {
		t.Errorf("zero-value CPUUsedSeconds should be 0, got %f", r.CPUUsedSeconds)
	}
	if r.Passed {
		t.Error("zero-value Passed should be false")
	}
	if r.Error != "" {
		t.Errorf("zero-value Error should be empty, got %q", r.Error)
	}
}

func TestVerifier_VerifyToolCode_TableDriven(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	tests := []struct {
		name     string
		code     string
		wantPass bool
	}{
		{"go_no_import", `package main
func main() {}`, true},
		{"go_no_import_with_body", `package main
func add(a, b int) int { return a + b }`, true},
		{"python_blocked_subprocess", `# python
import subprocess
subprocess.run(["ls"])`, false},
		{"python_blocked_socket", `# python
import socket
s = socket.socket()`, false},
		{"python_blocked_exec", "# python\nexec(\"import os\")", false},
		{"python_blocked_eval", "# python\nx = eval(\"2+2\")", false},
		{"python_blocked_imaplib", `# python
import imaplib
mail = imaplib.IMAP4_SSL("host")`, false},
		{"python_blocked_ctypes", `# python
import ctypes
lib = ctypes.CDLL("libc.so.6")`, false},
		{"python_blocked_os_system", `# python
import os
os.system("ls")`, false},
		{"python_blocked_open_http", "# python\nf = open(\"http://example.com\")", false},
		{"python_blocked_open_https", "# python\nf = open(\"https://example.com\")", false},
		{"python_blocked_open_ftp", "# python\nf = open(\"ftp://example.com\")", false},
		{"python_blocked___import__", "# python\nmod = __import__(\"subprocess\")", false},
		{"python_group_all", `# python
def hello():
    print("world")`, true},
		{"python_open_local", "# python\nf = open(\"/tmp/data.txt\")", true},
		{"python_clean_def", `# python
def process(data):
    return sorted(data)`, true},
		{"python_blocked_from_import", `# python
from subprocess import run
run(["ls"])`, false},
		{"python_blocked_requests", `# python
import requests
r = requests.get("http://x")`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := v.VerifyToolCode(tc.code)
			if result.Passed != tc.wantPass {
				t.Errorf("Passed = %v, want %v (error: %q)", result.Passed, tc.wantPass, result.Error)
			}
			if !tc.wantPass && result.Error == "" {
				t.Error("expected non-empty Error for failing validation")
			}
		})
	}
}

func TestVerifier_VerifyMultipleTools_CardinalityProperty(t *testing.T) {
	v := NewVerifier(slog.Default(), nil, "python3", "go")
	sizes := []int{0, 1, 2, 5, 10, 50, 100}
	for _, size := range sizes {
		t.Run("size_"+string(rune('0'+size%10)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			ids := make([]string, size)
			for i := range size {
				ids[i] = "id-" + strings.Repeat("x", i%3)
			}

			results := v.VerifyMultipleTools(ctx, ids, DefaultVerificationConfig())
			if len(results) != size {
				t.Errorf("size %d: got %d results", size, len(results))
			}
		})
	}
}

func FuzzVerifier_VerifyToolCode(f *testing.F) {
	seeds := []string{
		`package main
func main() {}`,
		`# python
import subprocess`,
		`# python
print("hello")`,
		`# python
ev\
al("2+2")`,
		``,
		`package main`,
		`package main
func f() { if true { return } }`,
		`not go or python code ; DROP TABLE users`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, code string) {
		v := NewVerifier(slog.Default(), nil, "python3", "go")
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("VerifyToolCode panicked on %q: %v", code, r)
			}
		}()
		result := v.VerifyToolCode(code)
		_ = result.Passed
		_ = result.Error
	})
}

func FuzzVerifier_isOutputSafe(f *testing.F) {
	seeds := []string{
		"hello",
		"/etc/passwd",
		"rm -rf /",
		"sudo ls",
		"fork bomb",
		"",
		"normal output",
		"\x00\x01\x02",
		strings.Repeat("a", 10000),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, output string) {
		v := NewVerifier(slog.Default(), nil, "python3", "go")
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("isOutputSafe panicked on %q: %v", output, r)
			}
		}()
		_ = v.isOutputSafe(output)
	})
}
