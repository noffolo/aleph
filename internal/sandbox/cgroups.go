//go:build linux

package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	cgroupBase    = "/sys/fs/cgroup"
	cgroupSubtree = "aleph"
	memoryMax     = "268435456"     // 256 MB
	cpuMax        = "50000 100000" // 50% of 1 CPU
	pidsMax       = "32"
)

func CreateCgroup(execID string) (string, error) {
	cgPath := filepath.Join(cgroupBase, cgroupSubtree, execID)

	if err := os.MkdirAll(cgPath, 0755); err != nil {
		return "", fmt.Errorf("sandbox: create cgroup dir %s: %w", cgPath, err)
	}

	writes := []struct {
		file    string
		content string
	}{
		{"memory.max", memoryMax},
		{"cpu.max", cpuMax},
		{"pids.max", pidsMax},
	}

	for _, w := range writes {
		fullPath := filepath.Join(cgPath, w.file)
		if err := os.WriteFile(fullPath, []byte(w.content), 0644); err != nil {
			// Some cgroup controllers may not be available; degrade gracefully
			// by writing "max" (no limit) as fallback for cpu and memory.
			if strings.Contains(w.file, "cpu.max") || strings.Contains(w.file, "memory.max") {
				_ = os.WriteFile(fullPath, []byte("max"), 0644)
			}
		}
	}

	return cgPath, nil
}

func AddProcessToCgroup(cgPath string, pid int) error {
	procsFile := filepath.Join(cgPath, "cgroup.procs")
	return os.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0644)
}

func CleanupCgroup(cgPath string) error {
	procs, err := os.ReadFile(filepath.Join(cgPath, "cgroup.procs"))
	if err == nil && len(strings.TrimSpace(string(procs))) > 0 {
		return fmt.Errorf("sandbox: cgroup %s still has active processes", cgPath)
	}
	return os.Remove(cgPath)
}