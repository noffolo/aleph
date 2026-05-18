//go:build linux

package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// runningInContainer detects if the test is running inside a container.
func runningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil && !strings.Contains(string(data), "0::/") {
		return true
	}
	// Check if PID 1 is a container-aware init
	data, err = os.ReadFile("/proc/self/cgroup")
	if err == nil && strings.Contains(string(data), "system.slice") {
		return true
	}
	return false
}

func TestCreateCgroup(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for cgroup operations")
	}

	// Check if the aleph subtree has controllers enabled.
	controllers, err := os.ReadFile(filepath.Join(cgroupBase, cgroupSubtree, "cgroup.subtree_control"))
	if err != nil || !strings.Contains(string(controllers), "memory") {
		t.Skip("skipping: cgroup controllers not enabled in aleph subtree")
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

	controllers, err := os.ReadFile(filepath.Join(cgroupBase, cgroupSubtree, "cgroup.subtree_control"))
	if err != nil || !strings.Contains(string(controllers), "memory") {
		t.Skip("skipping: cgroup controllers not enabled in aleph subtree")
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
	if r := runningInContainer(); r {
		t.Skip("skipping: user namespace cloning requires host-level capabilities in nested container")
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

	if cmd.SysProcAttr.Cloneflags != uintptr(expectedFlags) {
		t.Errorf("expected clone flags %v, got %v", expectedFlags, cmd.SysProcAttr.Cloneflags)
	}
}

func TestLoadSeccompFilter(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: requires root for seccomp operations")
	}
	if r := runningInContainer(); r {
		t.Skip("skipping: seccomp filter installation contaminates test process and requires host netns")
	}

	if err := LoadSeccompFilter(); err != nil {
		t.Fatalf("LoadSeccompFilter failed: %v", err)
	}
}
