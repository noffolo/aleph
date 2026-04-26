package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DevModeConfig controls the interactive development mode behavior.
type DevModeConfig struct {
	WatchDir     string        // Directory to watch for tool source files
	PollInterval time.Duration // How often to check for changes
	PythonCmd    string        // Python interpreter path
	GoCmd        string        // Go compiler path
}

// DefaultDevModeConfig returns sensible defaults for development mode.
func DefaultDevModeConfig() DevModeConfig {
	return DevModeConfig{
		WatchDir:     "./tools/dev",
		PollInterval: 2 * time.Second,
		PythonCmd:    "python3",
		GoCmd:        "go",
	}
}

// ToolChangeHandler is called when a tool file is created, modified, or deleted.
type ToolChangeHandler func(ctx context.Context, filename string, code string) error

// ToolWatcher watches a directory for tool file changes and auto-reloads them.
type ToolWatcher struct {
	config     DevModeConfig
	logger     *slog.Logger
	onChange   ToolChangeHandler
	checksums  map[string]string // filename -> sha256 hex digest
	stopCh     chan struct{}
}

// NewToolWatcher creates a new development-mode tool watcher.
func NewToolWatcher(logger *slog.Logger, config DevModeConfig, onChange ToolChangeHandler) *ToolWatcher {
	return &ToolWatcher{
		config:    config,
		logger:    logger,
		onChange:  onChange,
		checksums: make(map[string]string),
		stopCh:    make(chan struct{}),
	}
}

// Start begins polling the watch directory for changes. It blocks until the
// context is cancelled or Stop is called. The watch directory is created if
// it does not exist.
func (w *ToolWatcher) Start(ctx context.Context) error {
	if err := os.MkdirAll(w.config.WatchDir, 0755); err != nil {
		return fmt.Errorf("create watch dir: %w", err)
	}

	w.logger.Info("dev mode watcher started",
		"watch_dir", w.config.WatchDir,
		"poll_interval", w.config.PollInterval,
	)

	w.scanAndLoad(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.scanAndLoad(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		}
	}
}

// Stop halts the file watcher gracefully.
func (w *ToolWatcher) Stop() {
	close(w.stopCh)
}

// WatchedFiles returns the set of currently tracked filenames and their checksums.
func (w *ToolWatcher) WatchedFiles() map[string]string {
	result := make(map[string]string, len(w.checksums))
	for k, v := range w.checksums {
		result[k] = v
	}
	return result
}

// scanAndLoad reads all .go and .py files in the watch directory and calls
// the change handler for any new or modified files.
func (w *ToolWatcher) scanAndLoad(ctx context.Context) {
	entries, err := os.ReadDir(w.config.WatchDir)
	if err != nil {
		w.logger.Warn("dev mode: cannot read watch dir", "error", err)
		return
	}

	seen := make(map[string]bool, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".py") {
			continue
		}

		seen[name] = true
		fullPath := filepath.Join(w.config.WatchDir, name)

		raw, err := os.ReadFile(fullPath)
		if err != nil {
			w.logger.Warn("dev mode: cannot read file", "file", name, "error", err)
			continue
		}

		checksum := sha256Hex(raw)
		prevChecksum, exists := w.checksums[name]
		if exists && checksum == prevChecksum {
			continue // no change
		}

		w.checksums[name] = checksum
		codeStr := string(raw)

		// Auto-detect language from extension; Python files need a shebang or
		// the "# python" prefix that the sandbox recognises.
		if strings.HasSuffix(name, ".py") && !strings.HasPrefix(codeStr, "#") {
			codeStr = "# python\n" + codeStr
		}

		w.logger.Info("dev mode: tool changed",
			"file", name,
			"new", !exists,
			"size", len(raw),
		)

		if w.onChange != nil {
			if err := w.onChange(ctx, name, codeStr); err != nil {
				w.logger.Warn("dev mode: change handler failed",
					"file", name, "error", err,
				)
			}
		}
	}

	// Detect deleted files
	for name := range w.checksums {
		if !seen[name] {
			w.logger.Info("dev mode: tool removed", "file", name)
			delete(w.checksums, name)
		}
	}
}

// sha256Hex returns the SHA-256 hex digest of a byte slice.
func sha256Hex(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}
