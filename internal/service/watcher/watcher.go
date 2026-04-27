package watcher

import (
	"context"
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

type IngestionRunner interface {
	RunTask(ctx context.Context, projectID string, task *v1.IngestionTask) error
}

type Watcher struct {
	projectsRoot string
	runner       IngestionRunner
}

func NewWatcher(projectsRoot string, runner IngestionRunner) *Watcher {
	return &Watcher{projectsRoot: projectsRoot, runner: runner}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil { return fmt.Errorf("newWatcher: %w", err) }
	defer watcher.Close()

	// Watch projects root for new files in 'drop' folders
	// Simplified: we scan existing projects and watch their 'drop' directory
	// In a real system, we'd watch the root and add new projects dynamically
	
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok { return }
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Printf("[Watcher] File detected: %s", event.Name)
					// Logic: If file is in data/projects/{id}/drop/, trigger auto-ingestion
				}
			case err, ok := <-watcher.Errors:
				if !ok { return }
				log.Println("Watcher error:", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// For now, let's keep it as a placeholder service that we can expand
	log.Println("[Watcher] Event-driven engine ready.")
	return nil
}
