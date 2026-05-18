package sandbox

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
	_ "github.com/marcboeker/go-duckdb"
)

// mockSandboxManager implements SandboxManager for testing.
type mockSandboxManager struct {
	execResult ExecutionResult
	execErr    error
}

func (m *mockSandboxManager) ExecuteTool(ctx context.Context, toolID string, input map[string]any) (ExecutionResult, error) {
	return m.execResult, m.execErr
}

func (m *mockSandboxManager) RunSkill(ctx context.Context, skillID string, input map[string]any) (ExecutionResult, error) {
	return ExecutionResult{}, nil
}

func setupVerifyToolTest(t *testing.T) (*Verifier, *sql.DB) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory DuckDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE system_tools (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			code TEXT,
			category TEXT,
			version TEXT,
			health_status TEXT,
			source_type TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create system_tools: %v", err)
	}

	metaRepo, err := repository.NewMetadataRepository(db)
	if err != nil {
		t.Fatalf("NewMetadataRepository: %v", err)
	}

	logger := slog.Default()
	v := NewVerifier(logger, metaRepo, "python3", "go")
	return v, db
}

func TestVerifyTool_ToolNotFound(t *testing.T) {
	v, _ := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{}
	v2 := v.WithSandbox(mockSB)

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "nonexistent-tool", cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent tool, got nil")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestVerifyTool_BlockedGoImport(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('bad-py', 'BadPy', 'desc', 'import subprocess\nsubprocess.call(["ls"])', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "bad-py", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected tool to fail validation for blocked Python import")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestVerifyTool_InvalidPythonCode(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('bad-py', 'BadPy', 'desc', 'import os; os.system(\"ls\")', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "bad-py", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected Python tool with os.system to fail validation")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestVerifyTool_SandboxExecError(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{
		execResult: ExecutionResult{Stdout: "", Stderr: "exec failed", ExitCode: -1},
		execErr:    nil,
	}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('valid-tool', 'Valid', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "valid-tool", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected tool to fail due to sandbox execution error")
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestVerifyTool_SandboxExecFailure(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{
		execResult: ExecutionResult{Stdout: "", Stderr: "runtime error", ExitCode: 1},
		execErr:    nil,
	}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('failing-tool', 'Fail', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "failing-tool", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected tool to fail due to non-zero exit code")
	}
	if result.Error == "" {
		t.Error("expected error message for non-zero exit code")
	}
}

func TestVerifyTool_ContextExpired(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('any-tool', 'Any', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := v2.VerifyTool(ctx, "any-tool", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected error message about expired context")
	}
}

func TestVerifyTool_SuspiciousOutputDetected(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{
		execResult: ExecutionResult{Stdout: "/etc/passwd content", Stderr: "", ExitCode: 0},
		execErr:    nil,
	}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('escape-tool', 'Escape', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "escape-tool", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected tool to fail due to suspicious output")
	}
	if result.Error == "" {
		t.Error("expected error about suspicious output")
	}
}

func TestVerifyTool_WithSandbox_NilSandboxUsesExecSandbox(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	v2 := v.WithSandbox(nil) // exercises ExecSandbox fallback path

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('nil-sb', 'NilSB', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	cfg.TimeoutSeconds = 1 // prevent real exec
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "nil-sb", cfg)
	if err != nil {
		t.Logf("expected non-fatal: %v", err)
	}
	_ = result
}

func TestVerifyTool_EmptyCodeTriggersSandbox(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{
		execResult: ExecutionResult{Stdout: "", Stderr: "", ExitCode: 0},
	}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('empty-code', 'Empty', 'desc', '', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := DefaultVerificationConfig()
	ctx := context.Background()

	result, err := v2.VerifyTool(ctx, "empty-code", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (empty code passes validation)", result.ExitCode)
	}
}

func TestVerifyTool_TimeoutZeroUsesDefault(t *testing.T) {
	v, db := setupVerifyToolTest(t)
	mockSB := &mockSandboxManager{
		execResult: ExecutionResult{Stdout: "hello", Stderr: "", ExitCode: 0},
		execErr:    nil,
	}
	v2 := v.WithSandbox(mockSB)

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) VALUES ('timeout-zero', 'TZero', 'desc', 'package main\nfunc main() {}', 'cat', '1.0', 'unknown', 'builtin')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	cfg := VerificationConfig{TimeoutSeconds: 0, MaxMemoryMB: 256, MaxCPUSeconds: 10, NetworkBlocked: true}
	ctx := context.Background()

	start := time.Now()
	result, err := v2.VerifyTool(ctx, "timeout-zero", cfg)
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected tool to pass, got: %s", result.Error)
	}
	if dur > 30*time.Second {
		t.Errorf("test took too long: %v, default timeout not applied", dur)
	}
}
