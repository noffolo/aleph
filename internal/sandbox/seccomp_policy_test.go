//go:build linux

package sandbox

import (
	"testing"

	seccomp "github.com/elastic/go-seccomp-bpf"
)

func TestSandboxPolicy_DefaultAction(t *testing.T) {
	policy := sandboxPolicy()

	if policy.DefaultAction != seccomp.ActionErrno {
		t.Errorf("expected DefaultAction %v, got %v", seccomp.ActionErrno, policy.DefaultAction)
	}
}

func TestSandboxPolicy_SyscallsNotEmpty(t *testing.T) {
	policy := sandboxPolicy()

	if len(policy.Syscalls) == 0 {
		t.Fatal("sandboxPolicy() returned zero Syscalls groups — expected at least one")
	}
}

func TestSandboxPolicy_SyscallsAllowAction(t *testing.T) {
	policy := sandboxPolicy()

	for i, group := range policy.Syscalls {
		if group.Action != seccomp.ActionAllow {
			t.Errorf("Syscalls[%d] Action = %v, want %v", i, group.Action, seccomp.ActionAllow)
		}
	}
}

func TestSandboxPolicy_SyscallsUseAllowedSyscalls(t *testing.T) {
	policy := sandboxPolicy()

	if len(policy.Syscalls) < 1 {
		t.Fatal("no syscall groups")
	}

	names := policy.Syscalls[0].Names

	if len(names) != len(allowedSyscalls) {
		t.Errorf("Syscalls[0].Names length = %d, want %d (same as allowedSyscalls)", len(names), len(allowedSyscalls))
	}

	for i, name := range names {
		if name != allowedSyscalls[i] {
			t.Errorf("Syscalls[0].Names[%d] = %q, want %q", i, name, allowedSyscalls[i])
		}
	}
}

func TestAllowedSyscalls_NotEmpty(t *testing.T) {
	if len(allowedSyscalls) == 0 {
		t.Fatal("allowedSyscalls is empty — the seccomp whitelist must define at least one syscall")
	}
}

func TestAllowedSyscalls_ContainsBasics(t *testing.T) {
	m := make(map[string]bool, len(allowedSyscalls))
	for _, s := range allowedSyscalls {
		m[s] = true
	}

	required := []string{
		"read", "write", "close",
		"mmap", "munmap",
		"exit", "exit_group",
		"futex",
		"openat",
		"clone", "clone3",
	}

	for _, name := range required {
		if !m[name] {
			t.Errorf("required syscall %q missing from allowedSyscalls", name)
		}
	}
}

func TestSandboxPolicy_AllSyscallGroupsNonEmpty(t *testing.T) {
	policy := sandboxPolicy()

	for i, group := range policy.Syscalls {
		if len(group.Names) == 0 {
			t.Errorf("Syscalls[%d] Names is empty — every group should define at least one syscall", i)
		}
	}
}

func TestSandboxPolicy_NoDuplicateSyscalls(t *testing.T) {
	policy := sandboxPolicy()

	seen := make(map[string]bool)
	for _, group := range policy.Syscalls {
		for _, name := range group.Names {
			if seen[name] {
				t.Errorf("duplicate syscall %q found in sandboxPolicy", name)
			}
			seen[name] = true
		}
	}
}
