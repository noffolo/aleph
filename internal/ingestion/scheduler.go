package ingestion

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ff3300/aleph-v2/internal/repository"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/robfig/cron/v3"
	"github.com/google/uuid"
)

// Scheduler wraps cron.Cron to run scheduled ingestion tasks.
// Tasks with a non-empty Schedule field in system_tasks are loaded
// and executed on the cron schedule. Refresh() reconciles the in-memory
// cron entries with the database after CreateTask/DeleteTask.
type Scheduler struct {
	cron    *cron.Cron
	engine  *Engine
	metaRepo *repository.MetadataRepository
	entries map[string]cron.EntryID // taskID → cron entry ID for targeted removal
	mu      sync.Mutex
}

// NewScheduler creates a Scheduler for the given Engine.
// The cron runner uses local timezone and logs recoverable panics.
func NewScheduler(engine *Engine, metaRepo *repository.MetadataRepository) *Scheduler {
	return &Scheduler{
		cron: cron.New(
			cron.WithSeconds(),
			cron.WithLogger(cron.VerbosePrintfLogger(log.New(
				log.Writer(), "[scheduler] ", log.LstdFlags,
			))),
		),
		engine:   engine,
		metaRepo: metaRepo,
		entries:  make(map[string]cron.EntryID),
	}
}

// Start begins the cron scheduler in the background.
func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	if err := s.Refresh(ctx); err != nil {
		log.Printf("[scheduler] initial refresh failed: %v", err)
	}
	log.Println("[scheduler] started")

	// Periodically reconcile with DB every 5 minutes as a safety net.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Refresh(ctx); err != nil {
					log.Printf("[scheduler] periodic refresh: %v", err)
				}
			}
		}
	}()
}

// Stop stops the cron scheduler. Pending jobs are cancelled.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// Refresh loads all tasks with non-empty schedules from the database
// and reconciles the cron entries. Existing entries for removed tasks
// are removed; new entries for added tasks are registered.
func (s *Scheduler) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.metaRepo.ListScheduledTasks()
	if err != nil {
		return fmt.Errorf("list scheduled tasks: %w", err)
	}

	desired := make(map[string]*repository.IngestionTaskRecord, len(tasks))
	for i := range tasks {
		if tasks[i].Schedule != "" {
			desired[tasks[i].ID] = &tasks[i]
		}
	}

	for taskID, entryID := range s.entries {
		if _, ok := desired[taskID]; !ok {
			s.cron.Remove(entryID)
			delete(s.entries, taskID)
			log.Printf("[scheduler] removed schedule for task %s", taskID)
		}
	}

	for taskID, record := range desired {
		if _, ok := s.entries[taskID]; ok {
			continue
		}
		entryID, err := s.registerEntry(ctx, record)
		if err != nil {
			log.Printf("[scheduler] failed to register task %s: %v", taskID, err)
			continue
		}
		s.entries[taskID] = entryID
	}

	return nil
}

// AddTask registers a single task's cron schedule. Called after CreateTask.
func (s *Scheduler) AddTask(ctx context.Context, record *repository.IngestionTaskRecord) error {
	if record.Schedule == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[record.ID]; ok {
		s.cron.Remove(entryID)
	}

	entryID, err := s.registerEntry(ctx, record)
	if err != nil {
		return fmt.Errorf("register task %s: %w", record.ID, err)
	}
	s.entries[record.ID] = entryID
	log.Printf("[scheduler] added schedule %q for task %s", record.Schedule, record.ID)
	return nil
}

// RemoveTask removes a task's cron schedule. Called after DeleteTask.
func (s *Scheduler) RemoveTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
		log.Printf("[scheduler] removed schedule for task %s", taskID)
	}
}

// registerEntry creates a cron entry that calls engine.RunTask.
func (s *Scheduler) registerEntry(ctx context.Context, record *repository.IngestionTaskRecord) (cron.EntryID, error) {
	v1Task := &v1.IngestionTask{
		Id:         record.ID,
		Name:       record.Name,
		SourceType: record.SourceType,
		ConfigJson: record.ConfigJSON,
		Schedule:   record.Schedule,
	}

	spec := record.Schedule
	entryID, err := s.cron.AddFunc(spec, func() {
		runID := uuid.NewString()
		log.Printf("[scheduler] executing task %s (run %s) on schedule %q", record.ID, runID, spec)

		runCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		if err := s.engine.RunTask(runCtx, record.ProjectID, v1Task); err != nil {
			log.Printf("[scheduler] task %s (run %s) failed: %v", record.ID, runID, err)
		} else {
			log.Printf("[scheduler] task %s (run %s) completed", record.ID, runID)
		}
	})
	if err != nil {
		return 0, fmt.Errorf("cron.AddFunc(%q): %w", spec, err)
	}
	return entryID, nil
}
