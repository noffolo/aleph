//go:build linux

package sandbox

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestNamespaceSeccompIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "echo", "hello")
	cleanup, err := prepareSandboxedCmd(ctx, cmd, "test-integration-001")
	if err != nil {
		t.Fatalf("prepareSandboxedCmd failed: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("cmd.Output failed: %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", string(output))
	}
}

func TestNamespaceSeccomp_BlockedSyscall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "true")
	cleanup, err := prepareSandboxedCmd(ctx, cmd, "test-blocked-002")
	if err != nil {
		t.Fatalf("prepareSandboxedCmd failed: %v", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	err = cmd.Run()
	if err != nil {
		t.Fatalf("basic command should run: %v", err)
	}
}
