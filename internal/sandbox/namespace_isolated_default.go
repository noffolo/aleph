//go:build !linux

package sandbox

import (
	"context"
	"os/exec"
)

func ExecuteIsolated(_ context.Context, _ string, cmd *exec.Cmd) error {
	return cmd.Run()
}

func prepareSandboxedCmd(_ context.Context, _ *exec.Cmd, _ string) (func(), error) {
	return func() {}, nil
}

func LoadSeccompFilter() error { return nil }

func ApplySeccompFilter() {}

func CreateCgroup(_ string) (string, error) { return "", nil }

func AddProcessToCgroup(_ string, _ int) error { return nil }

func CleanupCgroup(_ string) error { return nil }