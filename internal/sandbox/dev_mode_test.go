package sandbox

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewToolWatcher(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultDevModeConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.WatchDir = t.TempDir()

	var callCount atomic.Int32
	handler := func(ctx context.Context, filename string, code string) error {
		callCount.Add(1)
		return nil
	}

	w := NewToolWatcher(logger, cfg, handler)
	if w == nil {
		t.Fatal("NewToolWatcher returned nil")
	}
	defer w.Stop()
}

func TestToolWatcher_DetectsNewFile(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultDevModeConfig()
	cfg.PollInterval = 100 * time.Millisecond
	cfg.WatchDir = t.TempDir()

	var callCount atomic.Int32
	handler := func(ctx context.Context, filename string, code string) error {
		callCount.Add(1)
		return nil
	}

	w := NewToolWatcher(logger, cfg, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	goCode := []byte("package main\nfunc main() {}\n")
	err := os.WriteFile(filepath.Join(cfg.WatchDir, "test_tool.go"), goCode, 0644)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)

	if callCount.Load() == 0 {
		t.Error("expected at least 1 callback invocation for new file")
	}
}

func TestToolWatcher_WatchedFiles(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultDevModeConfig()
	cfg.WatchDir = t.TempDir()

	w := NewToolWatcher(logger, cfg, nil)

	pyCode := []byte("# python\nprint('hello')\n")
	os.WriteFile(filepath.Join(cfg.WatchDir, "tool.py"), pyCode, 0644)

	w.scanAndLoad(context.Background())

	files := w.WatchedFiles()
	if len(files) != 1 {
		t.Errorf("expected 1 watched file, got %d", len(files))
	}
	if _, ok := files["tool.py"]; !ok {
		t.Error("expected tool.py in watched files")
	}
}

func TestToolWatcher_IgnoresNonToolFiles(t *testing.T) {
	logger := slog.Default()
	cfg := DefaultDevModeConfig()
	cfg.WatchDir = t.TempDir()

	var callCount atomic.Int32
	handler := func(ctx context.Context, filename string, code string) error {
		callCount.Add(1)
		return nil
	}

	w := NewToolWatcher(logger, cfg, handler)

	os.WriteFile(filepath.Join(cfg.WatchDir, "readme.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(cfg.WatchDir, "data.json"), []byte("{}"), 0644)

	w.scanAndLoad(context.Background())

	if callCount.Load() != 0 {
		t.Error("expected 0 callbacks for non-tool files")
	}
}

func TestDefaultDevModeConfig(t *testing.T) {
	cfg := DefaultDevModeConfig()
	if cfg.WatchDir != "./tools/dev" {
		t.Errorf("expected WatchDir ./tools/dev, got %q", cfg.WatchDir)
	}
	if cfg.PollInterval != 2*time.Second {
		t.Errorf("expected PollInterval 2s, got %v", cfg.PollInterval)
	}
}
