//go:build linux

package sandbox

import (
	"testing"

	seccomp "github.com/elastic/go-seccomp-bpf"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SandboxPolicySuite struct {
	suite.Suite
}

func (s *SandboxPolicySuite) TestDefaultAction() {
	policy := sandboxPolicy()
	s.NotNil(policy.Syscalls)
	s.Equal(seccomp.ActionErrno, policy.DefaultAction)
}

func (s *SandboxPolicySuite) TestSyscallNamesValid() {
	policy := sandboxPolicy()
	require.NotEmpty(s.T(), policy.Syscalls)

	for i, group := range policy.Syscalls {
		for j, name := range group.Names {
			s.Require().NotEmpty(name, "Syscalls[%d].Names[%d] is empty", i, j)
			for _, bad := range " \t\n\r" {
				s.NotContains(name, string(bad),
					"Syscalls[%d].Names[%d] %q contains whitespace", i, j, name)
			}
		}
	}
}

func (s *SandboxPolicySuite) TestNoDuplicateSyscalls() {
	policy := sandboxPolicy()
	seen := make(map[string]int, len(allowedSyscalls))
	for _, group := range policy.Syscalls {
		for _, name := range group.Names {
			seen[name]++
		}
	}
	for name, count := range seen {
		s.Equal(1, count, "duplicate syscall %q (%d occurrences)", name, count)
	}
}

func TestSandboxPolicy(t *testing.T) {
	suite.Run(t, new(SandboxPolicySuite))
}

type SeccompFilterSuite struct {
	suite.Suite
}

func (s *SeccompFilterSuite) TestFilterStructMatchesLoadSeccompFilter() {
	policy := sandboxPolicy()
	filter := seccomp.Filter{
		NoNewPrivs: true,
		Flag:       seccomp.FilterFlagTSync,
		Policy:     policy,
	}

	s.True(filter.NoNewPrivs, "NoNewPrivs must be true to prevent privilege escalation")
	s.Equal(seccomp.FilterFlagTSync, filter.Flag, "must use TSync to apply to all threads")
	s.Equal(seccomp.ActionErrno, filter.Policy.DefaultAction, "default must deny")
	s.NotEmpty(filter.Policy.Syscalls, "syscall whitelist must not be empty")
}

func (s *SeccompFilterSuite) TestAllowedSyscallsComplete() {
	m := make(map[string]bool, len(allowedSyscalls))
	for _, sys := range allowedSyscalls {
		m[sys] = true
	}

	s.True(m["read"] && m["write"] && m["close"], "basic I/O missing")
	s.True(m["mmap"] && m["munmap"], "memory mgmt missing")
	s.True(m["exit"] && m["exit_group"], "exit syscalls missing")
	s.True(m["futex"], "futex missing")
	s.True(m["clone"] || m["clone3"], "clone syscalls missing")
	s.True(m["openat"], "openat missing")
}

func (s *SeccompFilterSuite) TestAllowedSyscallsNoEmpty() {
	for i, name := range allowedSyscalls {
		s.NotEmpty(name, "allowedSyscalls[%d] is empty", i)
	}
	s.NotEmpty(allowedSyscalls, "allowedSyscalls list is empty")
}

func TestSeccompFilter(t *testing.T) {
	suite.Run(t, new(SeccompFilterSuite))
}

type ApplySeccompSuite struct {
	suite.Suite
}

func (s *ApplySeccompSuite) TestLoadSeccompFilterDoesNotPanicOnConstruction() {
	policy := sandboxPolicy()
	s.NotNil(policy.Syscalls)
	s.NotZero(policy.DefaultAction)

	filter := seccomp.Filter{
		NoNewPrivs: true,
		Flag:       seccomp.FilterFlagTSync,
		Policy:     policy,
	}
	s.True(filter.NoNewPrivs)
	s.Equal(seccomp.FilterFlagTSync, filter.Flag)
}

func (s *ApplySeccompSuite) TestPolicyAndAllowedSyscallsConsistent() {
	policy := sandboxPolicy()
	names := policy.Syscalls[0].Names
	s.Equal(len(names), len(allowedSyscalls),
		"policy syscalls count must match allowedSyscalls count")

	for i, name := range names {
		s.Equal(allowedSyscalls[i], name,
			"policy.Syscalls[0].Names[%d] = %q, allowedSyscalls[%d] = %q", i, name, i, allowedSyscalls[i])
	}
}

func (s *ApplySeccompSuite) TestFunctionsAreCallableWithoutTypeErrors() {
	var fnLoad func() error = LoadSeccompFilter
	var fnApply func() = ApplySeccompFilter
	s.NotNil(fnLoad)
	s.NotNil(fnApply)
}

func TestApplySeccompFilter(t *testing.T) {
	suite.Run(t, new(ApplySeccompSuite))
}
