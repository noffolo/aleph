package sandbox

import (
	"context"
	"os/exec"
	"testing"
)

func TestExecuteIsolated_NonLinux(t *testing.T) {
	cmd := exec.Command("echo", "test")
	err := ExecuteIsolated(context.Background(), "/tmp", cmd)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPrepareSandboxedCmd_NonLinux(t *testing.T) {
	cmd := exec.Command("echo", "test")
	cleanup, err := prepareSandboxedCmd(context.Background(), cmd, "exec-1")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if cleanup == nil {
		t.Error("expected non-nil cleanup function")
	} else {
		cleanup() // verify no-op doesn't panic
	}
}

func TestLoadSeccompFilter_NonLinux(t *testing.T) {
	err := LoadSeccompFilter()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestApplySeccompFilter_NonLinux(t *testing.T) {
	// void function — verify no panic
	ApplySeccompFilter()
}

func TestCgroupStubs_NonLinux(t *testing.T) {
	t.Run("CreateCgroup", func(t *testing.T) {
		cgPath, err := CreateCgroup("exec-1")
		if err != nil {
			t.Errorf("expected nil err, got %v", err)
		}
		if cgPath != "" {
			t.Errorf("expected empty path, got %q", cgPath)
		}
	})
	t.Run("AddProcessToCgroup", func(t *testing.T) {
		err := AddProcessToCgroup("", 0)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
	t.Run("CleanupCgroup", func(t *testing.T) {
		err := CleanupCgroup("")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
