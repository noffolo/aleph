//go:build linux

package sandbox

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestExecuteIsolated_NilCmd(t *testing.T) {
	// ExecuteIsolated accesses cmd.SysProcAttr without nil guard.
	// This test documents the behavior — it will panic.
	tmpDir := t.TempDir()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil cmd, got none")
		}
	}()
	_ = ExecuteIsolated(context.Background(), tmpDir, nil) //nolint:staticcheck
}

func TestExecuteIsolated_EmptyTmpDir(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for namespace isolation")
	}
	cmd := exec.Command("echo", "test")
	if err := ExecuteIsolated(context.Background(), "", cmd); err != nil {
		t.Fatalf("ExecuteIsolated with empty tmpDir failed: %v", err)
	}
}

func TestPrepareSandboxedCmd_NamespaceFailure(t *testing.T) {
	t.Skip("skipping: calls LoadSeccompFilter which installs a destructive BPF filter")
	cmd := exec.Command("echo", "test")
	_, err := prepareSandboxedCmd(context.Background(), cmd, "test-id")
	if err != nil {
		t.Fatalf("prepareSandboxedCmd failed: %v", err)
	}
}
