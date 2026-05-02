package watcher

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

// IngestionRunner abstracts the ingestion engine so the watcher
// can trigger ingestion without importing the engine package directly.
type IngestionRunner interface {
	RunTask(ctx context.Context, projectID string, task *v1.IngestionTask) error
}

// knownExts maps file extensions to ingestion SourceType values.
var knownExts = map[string]string{
	".csv":     "csv",
	".tsv":     "csv",
	".json":    "custom_code",
	".parquet": "copy",
}

// Watcher monitors project drop directories for new files and triggers
// automatic ingestion via the IngestionRunner.
type Watcher struct {
	projectsRoot string
	runner       IngestionRunner
	fw           *fsnotify.Watcher
	mu           sync.Mutex
	watchedDirs  map[string]bool
}

func NewWatcher(projectsRoot string, runner IngestionRunner) *Watcher {
	return &Watcher{
		projectsRoot: projectsRoot,
		runner:       runner,
		watchedDirs:  make(map[string]bool),
	}
}

// AddProject ensures the project's drop/ directory exists and starts
// watching it for new files. It is safe to call before Start() — the
// directory will be created and tracked, then wired to fsnotify once
// Start() has created the underlying watcher.
func (w *Watcher) AddProject(ctx context.Context, projectID string) error {
	dropDir := filepath.Join(w.projectsRoot, projectID, "drop")
	if err := os.MkdirAll(dropDir, 0755); err != nil {
		return fmt.Errorf("create drop dir %s: %w", dropDir, err)
	}

	w.mu.Lock()
	if w.watchedDirs[dropDir] {
		w.mu.Unlock()
		return nil // already watching
	}
	w.watchedDirs[dropDir] = true
	fw := w.fw
	w.mu.Unlock()

	if fw == nil {
		return nil // Start() not yet called; will be picked up there
	}
	return fw.Add(dropDir)
}

// WatchProject is a convenience wrapper around AddProject.
func (w *Watcher) WatchProject(ctx context.Context, projectID string) error {
	return w.AddProject(ctx, projectID)
}

// Start creates the fsnotify watcher, discovers existing project
// drop directories, and begins the event loop. Blocks until ctx is
// cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("newWatcher: %w", err)
	}

	// Wire the fsnotify watcher into the struct so AddProject can
	// register directories. Clear any pre-Start tracked dirs so that
	// AddProject below re-adds them to the live fsnotify watcher.
	w.mu.Lock()
	w.fw = watcher
	w.watchedDirs = make(map[string]bool)
	w.mu.Unlock()

	// Discover existing project directories on startup.
	entries, err := os.ReadDir(w.projectsRoot)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				if addErr := w.AddProject(ctx, entry.Name()); addErr != nil {
					log.Printf("[Watcher] Failed to watch project %s: %v", entry.Name(), addErr)
				}
			}
		}
	} else {
		log.Printf("[Watcher] Could not read projectsRoot %s: %v", w.projectsRoot, err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Watcher] event loop panic recovered: %v", r)
			}
		}()
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Printf("[Watcher] File detected: %s", event.Name)
					w.handleCreateEvent(ctx, event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Println("[Watcher] Event-driven engine ready.")
	return nil
}

// handleCreateEvent processes a single CREATE event. It checks the
// file extension, debounces until the file is stable (same mod time
// and size after 500ms), builds an IngestionTask, and launches
// ingestion in a background goroutine.
func (w *Watcher) handleCreateEvent(ctx context.Context, path string) {
	ext := strings.ToLower(filepath.Ext(path))
	sourceType, ok := knownExts[ext]
	if !ok {
		return // not a known file type
	}

	// Debounce: wait for the file to finish being written by
	// comparing mod time and size over a 500 ms interval.
	if !isFileStable(path) {
		log.Printf("[Watcher] File %s not stable after debounce, skipping", path)
		return
	}

	filename := filepath.Base(path)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Build ConfigJson — CSV/TSV get a tableName hint.
	config := map[string]string{"path": path}
	if sourceType == "csv" {
		config["tableName"] = nameWithoutExt
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Printf("[Watcher] Failed to marshal config for %s: %v", path, err)
		return
	}

	// Generate a random task ID.
	b := make([]byte, 16)
	if _, randErr := rand.Read(b); randErr != nil {
		log.Printf("[Watcher] Failed to generate task ID: %v", randErr)
		return
	}
	taskID := fmt.Sprintf("%x", b)

	task := &v1.IngestionTask{
		Id:         taskID,
		Name:       filename,
		SourceType: sourceType,
		ConfigJson: string(configBytes),
		Status:     "pending",
	}

	// Extract projectID from the path: projectsRoot/{projectID}/drop/{filename}
	rel, err := filepath.Rel(w.projectsRoot, path)
	if err != nil {
		log.Printf("[Watcher] Failed to get relative path for %s: %v", path, err)
		return
	}
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if len(parts) < 2 || parts[0] == "" {
		log.Printf("[Watcher] Unexpected path structure: %s", rel)
		return
	}
	projectID := parts[0]

	// Run ingestion in a background goroutine so the watcher stays
	// non-blocking.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Watcher] ingestion goroutine panic for %s: %v", filename, r)
			}
		}()
		taskCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		if err := w.runner.RunTask(taskCtx, projectID, task); err != nil {
			log.Printf("[Watcher] Ingestion failed for %s (project=%s): %v", filename, projectID, err)
		} else {
			log.Printf("[Watcher] Ingestion completed for %s (project=%s)", filename, projectID)
		}
	}()
}

// isFileStable checks that the file at path has not been modified for
// at least 500 ms by comparing ModTime and Size. Returns false if the
// file does not exist or is still being written to.
func isFileStable(path string) bool {
	fi1, err := os.Stat(path)
	if err != nil {
		return false
	}

	time.Sleep(500 * time.Millisecond)

	fi2, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fi1.ModTime().Equal(fi2.ModTime()) && fi1.Size() == fi2.Size()
}

// Close tears down the fsnotify watcher if it is running. Safe to
// call multiple times.
func (w *Watcher) Close() error {
	w.mu.Lock()
	fw := w.fw
	w.fw = nil
	w.mu.Unlock()

	if fw != nil {
		return fw.Close()
	}
	return nil
}
