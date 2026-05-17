//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

func ExecuteIsolated(ctx context.Context, tmpDir string, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}

	slog.Debug("sandbox: namespace isolation applied",
		"pid", os.Getpid(),
		"tmpDir", tmpDir,
	)

	return nil
}

func prepareSandboxedCmd(ctx context.Context, cmd *exec.Cmd, execID string) (cgCleanup func(), err error) {
	if err := ExecuteIsolated(ctx, "", cmd); err != nil {
		return nil, fmt.Errorf("sandbox: namespace isolation failed: %w", err)
	}

	if err := LoadSeccompFilter(); err != nil {
		return nil, fmt.Errorf("sandbox: seccomp filter failed: %w", err)
	}

	cgPath, err := CreateCgroup(execID)
	if err != nil {
		slog.Warn("sandbox: cgroup creation failed, continuing without cgroup limits", "error", err)
		cgCleanup = func() {}
	} else {
		cgCleanup = func() {
			if cleanupErr := CleanupCgroup(cgPath); cleanupErr != nil {
				slog.Warn("sandbox: cgroup cleanup failed", "cgPath", cgPath, "error", cleanupErr)
			}
		}
	}

	return cgCleanup, nil
}