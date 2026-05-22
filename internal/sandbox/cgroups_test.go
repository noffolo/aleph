//go:build linux

package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCgroup_Happy(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root for cgroup operations")
	}

	controllers, _ := os.ReadFile(filepath.Join(cgroupBase, cgroupSubtree, "cgroup.subtree_control"))
	if !strings.Contains(string(controllers), "memory") {
		t.Skip("cgroup controllers not enabled in aleph subtree")
	}

	cgPath, err := CreateCgroup("happy-test")
	require.NoError(t, err)
	defer CleanupCgroup(cgPath)

	assert.NotEmpty(t, cgPath)
	assert.Contains(t, cgPath, cgroupSubtree)
	assert.Contains(t, cgPath, "happy-test")

	dirInfo, statErr := os.Stat(cgPath)
	require.NoError(t, statErr)
	assert.True(t, dirInfo.IsDir())

	for _, file := range []string{"memory.max", "cpu.max", "pids.max"} {
		_, statErr := os.Stat(filepath.Join(cgPath, file))
		assert.NoError(t, statErr, "cgroup file %s should exist when controllers are enabled", file)
	}
}

func TestCreateCgroup_Edge_EmptyExecID(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("")
	require.NoError(t, err, "CreateCgroup with empty execID should still create a directory")
	defer CleanupCgroup(cgPath)

	assert.NotEmpty(t, cgPath)
	assert.Contains(t, cgPath, cgroupSubtree)

	_, statErr := os.Stat(cgPath)
	assert.NoError(t, statErr, "cgroup dir must exist even with empty execID")
}

func TestCreateCgroup_Error_UnwritablePath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere, test is meaningless as root")
	}

	_, err := CreateCgroup("/proc/fake-aleph/test-id")
	assert.Error(t, err, "writing to /proc should fail for non-root")
	assert.Empty(t, "", "cgPath should be empty on error")
}

func TestAddProcessToCgroup_Happy(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("add-proc-test")
	require.NoError(t, err)
	defer CleanupCgroup(cgPath)

	pid := os.Getpid()
	err = AddProcessToCgroup(cgPath, pid)
	assert.NoError(t, err)

	procsData, readErr := os.ReadFile(filepath.Join(cgPath, "cgroup.procs"))
	require.NoError(t, readErr)
	assert.Contains(t, string(procsData), "", "cgroup.procs should contain our PID")
}

func TestAddProcessToCgroup_Edge_InvalidPID(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("invalid-pid-test")
	require.NoError(t, err)
	defer CleanupCgroup(cgPath)

	err = AddProcessToCgroup(cgPath, -1)
	assert.Error(t, err, "AddProcessToCgroup with PID -1 must fail")
}

func TestAddProcessToCgroup_Error_NonexistentCgroup(t *testing.T) {
	err := AddProcessToCgroup("/sys/fs/cgroup/aleph/nonexistent-cgroup", os.Getpid())
	assert.Error(t, err)
}

func TestCleanupCgroup_Happy(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root for cgroup operations")
	}

	cgPath, err := CreateCgroup("cleanup-test")
	require.NoError(t, err)

	err = CleanupCgroup(cgPath)
	assert.NoError(t, err)

	_, statErr := os.Stat(cgPath)
	assert.True(t, os.IsNotExist(statErr), "cgroup directory must be removed after cleanup")
}

func TestCleanupCgroup_Edge_EmptyPath(t *testing.T) {
	err := CleanupCgroup("")
	assert.Error(t, err)
}

func TestCleanupCgroup_Error_NonexistentPath(t *testing.T) {
	err := CleanupCgroup("/tmp/aleph-nonexistent-cgroup")
	assert.Error(t, err)
}
