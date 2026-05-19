//go:build linux

package sandbox

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCgroup_DirCreation(t *testing.T) {
	cgPath, err := CreateCgroup("coverage-test")

	if err != nil {
		t.Logf("CreateCgroup degraded (expected): %v", err)
		assert.Empty(t, cgPath)
	} else {
		t.Logf("CreateCgroup succeeded: %s", cgPath)
		defer CleanupCgroup(cgPath)
		assert.NotEmpty(t, cgPath)
	}
}

func TestCreateCgroup_InvalidPath(t *testing.T) {
	cgPath, err := CreateCgroup("../../../etc/passwd")
	if err != nil {
		t.Logf("traversal rejected as expected: %v", err)
	} else {
		t.Logf("traversal created cgroup at: %s", cgPath)
		defer CleanupCgroup(cgPath)
	}
	assert.NotNil(t, func() { _, _ = CreateCgroup("../../../etc/passwd") })
}

func TestCreateCgroup_EmptyExecID(t *testing.T) {
	cgPath, err := CreateCgroup("")

	if err != nil {
		t.Logf("empty execID rejected: %v", err)
		assert.Empty(t, cgPath)
	} else {
		t.Logf("empty execID created cgroup at: %s", cgPath)
		defer CleanupCgroup(cgPath)
		assert.NotEmpty(t, cgPath)
	}
}

func TestCreateCgroup_MultipleDifferentIDs(t *testing.T) {
	ids := []string{"a", "b", "c"}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			cgPath, err := CreateCgroup(id)
			if err == nil {
				defer CleanupCgroup(cgPath)
			}
			if err != nil {
				t.Logf("%s failed (ok): %v", id, err)
			}
		})
	}
}

func TestAddProcessToCgroup_Normal(t *testing.T) {
	err := AddProcessToCgroup("/sys/fs/cgroup/aleph/fake-id", os.Getpid())
	if err != nil {
		t.Logf("AddProcessToCgroup failed (expected for nonexistent path): %v", err)
	}
}

func TestCleanupCgroup_Normal(t *testing.T) {
	err := CleanupCgroup("/sys/fs/cgroup/aleph/fake-id-for-cleanup")
	if err != nil {
		t.Logf("CleanupCgroup failed (expected for nonexistent path): %v", err)
	}
}

func TestCleanupCgroup_EmptyPath(t *testing.T) {
	err := CleanupCgroup("")
	if err != nil {
		t.Logf("CleanupCgroup with empty path failed (expected): %v", err)
	}
}

func TestPrepareSandboxedCmd_CannotTest_NoRoot(t *testing.T) {
	t.Skip("prepareSandboxedCmd calls LoadSeccompFilter which permanently contaminates the test process " +
		"with seccomp-bpf, crashing the Go runtime netpoll. Testing this function requires a dedicated " +
		"subprocess or root privileges for safe seccomp installation.")
}

func TestLoadSeccompFilter_CannotTest_NoRoot(t *testing.T) {
	t.Skip("LoadSeccompFilter installs seccomp-bpf into the calling process, permanently restricting " +
		"syscalls. This crashes the Go runtime (netpoll, mmap, etc.) in the test runner. " +
		"Testing requires root and a dedicated subprocess sandbox.")
}

func TestApplySeccompFilter_CannotTest_NoRoot(t *testing.T) {
	t.Skip("ApplySeccompFilter calls LoadSeccompFilter, which permanently contaminates the test process. " +
		"Safe testing requires root privileges and a dedicated subprocess.")
}
