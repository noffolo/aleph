//go:build linux

package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCreateCgroup(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("test-isolation")
	if err != nil {
		t.Fatalf("CreateCgroup failed: %v", err)
	}
	defer CleanupCgroup(cgPath)

	for _, file := range []string{"memory.max", "cpu.max", "pids.max"} {
		fullPath := filepath.Join(cgPath, file)
		if _, statErr := os.Stat(fullPath); statErr != nil {
			t.Errorf("cgroup file %s missing: %v", file, statErr)
		}
	}
}

func TestAddProcessToCgroup(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("test-add-proc")
	if err != nil {
		t.Fatalf("CreateCgroup failed: %v", err)
	}
	defer CleanupCgroup(cgPath)

	if addErr := AddProcessToCgroup(cgPath, os.Getpid()); addErr != nil {
		t.Errorf("AddProcessToCgroup failed: %v", addErr)
	}
}

func TestExecuteIsolated(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for namespace isolation")
	}

	cmd := exec.Command("echo", "hello-isolated")
	tmpDir := t.TempDir()

	if err := ExecuteIsolated(nil, tmpDir, cmd); err != nil {
		t.Fatalf("ExecuteIsolated failed: %v", err)
	}

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should be set after ExecuteIsolated")
	}

	expectedFlags := syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
		syscall.CLONE_NEWNS | syscall.CLONE_NEWNET |
		syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER

	if cmd.SysProcAttr.Cloneflags != expectedFlags {
		t.Errorf("expected clone flags %v, got %v", expectedFlags, cmd.SysProcAttr.Cloneflags)
	}
}

func TestLoadSeccompFilter(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for seccomp operations")
	}

	if err := LoadSeccompFilter(); err != nil {
		t.Fatalf("LoadSeccompFilter failed: %v", err)
	}
}