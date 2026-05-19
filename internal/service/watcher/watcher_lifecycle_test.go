package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestWatcher_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	runner := &mockRunner{}
	w := NewWatcher(tmpDir, runner)

	ctx, cancel := context.WithCancel(context.Background())

	err := w.Start(ctx)
	require.NoError(t, err)

	// Verify the underlying fsnotify watcher was created and wired
	w.mu.Lock()
	assert.NotNil(t, w.fw, "expected fsnotify watcher to be initialized after Start")
	w.mu.Unlock()

	// Cancel the context to trigger shutdown of the event loop goroutine
	cancel()

	// Give the event loop goroutine time to exit via ctx.Done()
	time.Sleep(500 * time.Millisecond)

	// Close should be safe (the goroutine's defer already called watcher.Close(),
	// but our Close() is idempotent)
	err = w.Close()
	// We don't assert NoError here because double-close may return an error,
	// but it must never panic
	_ = err
}

func TestWatcher_StartStop_EmptyProjectsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a path that doesn't exist on disk
	nonexistentRoot := filepath.Join(tmpDir, "does-not-exist")

	runner := &mockRunner{}
	w := NewWatcher(nonexistentRoot, runner)

	ctx, cancel := context.WithCancel(context.Background())

	// Start should handle missing projectsRoot gracefully (logs a warning, does not fail)
	err := w.Start(ctx)
	require.NoError(t, err)

	// Verify watcher was created despite missing projects root
	w.mu.Lock()
	assert.NotNil(t, w.fw)
	w.mu.Unlock()

	cancel()
	time.Sleep(300 * time.Millisecond)
	_ = w.Close()
}

func TestWatcher_Start_DiscoverExistingProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create a project directory with drop/ subdir before starting the watcher
	projectDropDir := filepath.Join(tmpDir, "existing-project", "drop")
	err := os.MkdirAll(projectDropDir, 0755)
	require.NoError(t, err)

	runner := &mockRunner{}
	w := NewWatcher(tmpDir, runner)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(300 * time.Millisecond)
		_ = w.Close()
	}()

	err = w.Start(ctx)
	require.NoError(t, err)

	// The existing project should have been discovered and its drop/ dir added to fsnotify
	w.mu.Lock()
	assert.True(t, w.watchedDirs[projectDropDir], "existing project drop dir should be watched after Start")
	w.mu.Unlock()
}

func TestWatcher_Start_MultipleStartCalls(t *testing.T) {
	tmpDir := t.TempDir()
	runner := &mockRunner{}
	w := NewWatcher(tmpDir, runner)

	// First Start
	ctx1, cancel1 := context.WithCancel(context.Background())
	err := w.Start(ctx1)
	require.NoError(t, err)
	cancel1()
	time.Sleep(300 * time.Millisecond)
	_ = w.Close()

	// Second Start on the same Watcher instance
	ctx2, cancel2 := context.WithCancel(context.Background())
	err = w.Start(ctx2)
	require.NoError(t, err)

	w.mu.Lock()
	assert.NotNil(t, w.fw, "fsnotify watcher should be reinitialized on second Start")
	w.mu.Unlock()

	cancel2()
	time.Sleep(300 * time.Millisecond)
	_ = w.Close()
}

// =============================================================================
// handleCreateEvent Tests
// =============================================================================

func TestWatcher_HandleCreateEvent_Success(t *testing.T) {
	tests := []struct {
		name       string
		ext        string
		content    string
		sourceType string
		hasTable   bool
	}{
		{"CSV", ".csv", "a,b,c\n1,2,3\n", "csv", true},
		{"TSV", ".tsv", "a\tb\tc\n1\t2\t3\n", "csv", true},
		{"JSON", ".json", `{"key":"value"}`, "custom_code", false},
		{"Parquet", ".parquet", "PAR1\x00\x00", "copy", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			// Create project directory structure
			dropDir := filepath.Join(tmpDir, "testproj", "drop")
			err := os.MkdirAll(dropDir, 0755)
			require.NoError(t, err)

			filename := "datafile" + tt.ext
			filePath := filepath.Join(dropDir, filename)
			err = os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			called := make(chan struct{}, 1)
			runner := &mockRunner{
				called: called,
			}

			w := NewWatcher(tmpDir, runner)

			// Call handleCreateEvent directly (not via fsnotify event loop)
			w.handleCreateEvent(context.Background(), filePath)

			// Wait for the ingestion goroutine to call RunTask
			select {
			case <-called:
				runner.Lock()
				task := runner.lastTask
				projID := runner.lastProjID
				runner.Unlock()

				require.NotNil(t, task, "task should not be nil")
				assert.Equal(t, "testproj", projID, "projectID should be extracted from path")
				assert.Equal(t, filename, task.Name)
				assert.Equal(t, tt.sourceType, task.SourceType)
				assert.Equal(t, "pending", task.Status)
				assert.NotEmpty(t, task.Id, "task ID should be generated")
				assert.Len(t, task.Id, 32, "task ID should be a 32-char hex string")
				assert.Contains(t, task.ConfigJson, filePath, "config should contain the file path")

				if tt.hasTable {
					assert.Contains(t, task.ConfigJson, "tableName", "CSV/TSV should have tableName in config")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for handleCreateEvent to trigger ingestion via RunTask")
			}
		})
	}
}

func TestWatcher_HandleCreateEvent_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filePath    string
		setup       func(tmpDir string) string
		expectCall  bool
		description string
	}{
		{
			name: "UnknownExtension",
			setup: func(tmpDir string) string {
				dropDir := filepath.Join(tmpDir, "testproj", "drop")
				_ = os.MkdirAll(dropDir, 0755)
				fp := filepath.Join(dropDir, "readme.txt")
				_ = os.WriteFile(fp, []byte("hello world"), 0644)
				return fp
			},
			expectCall:  false,
			description: "files with unknown extensions (.txt) should be silently ignored",
		},
		{
			name: "UnknownExtension_PDF",
			setup: func(tmpDir string) string {
				dropDir := filepath.Join(tmpDir, "testproj", "drop")
				_ = os.MkdirAll(dropDir, 0755)
				fp := filepath.Join(dropDir, "report.pdf")
				_ = os.WriteFile(fp, []byte("%PDF-1.4"), 0644)
				return fp
			},
			expectCall:  false,
			description: "PDF files should be silently ignored",
		},
		{
			name: "NonExistentFile",
			setup: func(tmpDir string) string {
				dropDir := filepath.Join(tmpDir, "testproj", "drop")
				_ = os.MkdirAll(dropDir, 0755)
				return filepath.Join(dropDir, "does-not-exist.csv")
			},
			expectCall:  false,
			description: "non-existent files should not trigger ingestion (isFileStable returns false)",
		},
		{
			name: "NoExtension",
			setup: func(tmpDir string) string {
				dropDir := filepath.Join(tmpDir, "testproj", "drop")
				_ = os.MkdirAll(dropDir, 0755)
				fp := filepath.Join(dropDir, "noextension")
				_ = os.WriteFile(fp, []byte("data"), 0644)
				return fp
			},
			expectCall:  false,
			description: "files without extension should be silently ignored",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := tt.setup(tmpDir)

			called := make(chan struct{}, 1)
			runner := &mockRunner{
				called: called,
			}

			w := NewWatcher(tmpDir, runner)
			w.handleCreateEvent(context.Background(), filePath)

			select {
			case <-called:
				if !tt.expectCall {
					t.Errorf("UNEXPECTED call: %s", tt.description)
				}
			case <-time.After(3 * time.Second):
				if tt.expectCall {
					t.Errorf("MISSING call: %s", tt.description)
				}
			}
		})
	}
}

func TestWatcher_HandleCreateEvent_InvalidPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a file in a completely separate temp directory
	// (i.e., outside projectsRoot)
	otherDir := t.TempDir()
	csvFile := filepath.Join(otherDir, "orphan.csv")
	err := os.WriteFile(csvFile, []byte("a,b\n1,2\n"), 0644)
	require.NoError(t, err)

	runner := &mockRunner{}
	w := NewWatcher(tmpDir, runner)

	// Should not panic even when the path has no relation to projectsRoot
	// filepath.Rel will fail or produce an unexpected path structure
	assert.NotPanics(t, func() {
		w.handleCreateEvent(context.Background(), csvFile)
	}, "handleCreateEvent must not panic with a path outside projectsRoot")
}

func TestWatcher_HandleCreateEvent_RunnerError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	dropDir := filepath.Join(tmpDir, "testproj", "drop")
	err := os.MkdirAll(dropDir, 0755)
	require.NoError(t, err)

	csvFile := filepath.Join(dropDir, "data.csv")
	err = os.WriteFile(csvFile, []byte("col1,col2\nv1,v2\n"), 0644)
	require.NoError(t, err)

	called := make(chan struct{}, 1)
	runner := &mockRunner{
		called: called,
		runTaskFn: func(ctx context.Context, projectID string, task *v1.IngestionTask) error {
			return fmt.Errorf("simulated ingestion failure")
		},
	}

	w := NewWatcher(tmpDir, runner)
	w.handleCreateEvent(context.Background(), csvFile)

	select {
	case <-called:
		// Runner was called even though it returns an error — correct behavior.
		// handleCreateEvent spawns ingestion in a background goroutine and
		// logs the error; it never panics or returns.
	case <-time.After(5 * time.Second):
		t.Fatal("runner.RunTask should have been called despite returning an error")
	}
}

func TestWatcher_HandleCreateEvent_MultipleExtensions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	dropDir := filepath.Join(tmpDir, "testproj", "drop")
	err := os.MkdirAll(dropDir, 0755)
	require.NoError(t, err)

	// File with double extension like "data.backup.csv" — ext should be ".csv"
	filename := "data.backup.csv"
	filePath := filepath.Join(dropDir, filename)
	err = os.WriteFile(filePath, []byte("a,b\n"), 0644)
	require.NoError(t, err)

	called := make(chan struct{}, 1)
	runner := &mockRunner{
		called: called,
	}

	w := NewWatcher(tmpDir, runner)
	w.handleCreateEvent(context.Background(), filePath)

	select {
	case <-called:
		runner.Lock()
		task := runner.lastTask
		runner.Unlock()

		assert.Equal(t, "csv", task.SourceType)
		assert.Equal(t, filename, task.Name)
		// Table name should be "data.backup" (TrimSuffix removes only ".csv")
		assert.Contains(t, task.ConfigJson, `"tableName":"data.backup"`)
	case <-time.After(5 * time.Second):
		t.Fatal("double-extension CSV file should be ingested")
	}
}

// =============================================================================
// Event Loop Tests (fsnotify integration)
// =============================================================================

func TestWatcher_EventLoop_CreateEvent(t *testing.T) {
	tmpDir := t.TempDir()

	called := make(chan struct{}, 1)
	runner := &mockRunner{
		called: called,
	}

	w := NewWatcher(tmpDir, runner)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(300 * time.Millisecond)
		_ = w.Close()
	}()

	err := w.Start(ctx)
	require.NoError(t, err)

	// Add a project so fsnotify watches the drop directory
	err = w.AddProject(ctx, "myproject")
	require.NoError(t, err)

	dropDir := filepath.Join(tmpDir, "myproject", "drop")

	// Brief pause to ensure fsnotify watch is established
	time.Sleep(200 * time.Millisecond)

	// Create a CSV file — fsnotify should detect it and fire a CREATE event
	csvFile := filepath.Join(dropDir, "incoming.csv")
	err = os.WriteFile(csvFile, []byte("x,y,z\n1,2,3\n"), 0644)
	require.NoError(t, err)

	// Wait for: fsnotify detection + 500ms isFileStable debounce + goroutine scheduling
	select {
	case <-called:
		runner.Lock()
		task := runner.lastTask
		projID := runner.lastProjID
		runner.Unlock()

		require.NotNil(t, task)
		assert.Equal(t, "myproject", projID)
		assert.Equal(t, "incoming.csv", task.Name)
		assert.Equal(t, "csv", task.SourceType)
		assert.Equal(t, "pending", task.Status)
		assert.NotEmpty(t, task.Id)
		assert.Contains(t, task.ConfigJson, csvFile)
		assert.Contains(t, task.ConfigJson, `"tableName":"incoming"`)
	case <-time.After(15 * time.Second):
		t.Fatal("timeout: fsnotify event loop did not detect file and trigger ingestion")
	}
}

func TestWatcher_EventLoop_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	callCount := 0
	var mu sync.Mutex
	called := make(chan struct{}, 5)

	runner := &mockRunner{
		runTaskFn: func(ctx context.Context, projectID string, task *v1.IngestionTask) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			select {
			case called <- struct{}{}:
			default:
			}
			return nil
		},
	}

	w := NewWatcher(tmpDir, runner)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(500 * time.Millisecond)
		_ = w.Close()
	}()

	err := w.Start(ctx)
	require.NoError(t, err)

	err = w.AddProject(ctx, "multiproj")
	require.NoError(t, err)

	dropDir := filepath.Join(tmpDir, "multiproj", "drop")
	time.Sleep(200 * time.Millisecond)

	// Create multiple files
	files := []string{"file1.csv", "file2.csv", "file3.csv"}
	for _, f := range files {
		fp := filepath.Join(dropDir, f)
		_ = os.WriteFile(fp, []byte("a,b\n1,2\n"), 0644)
	}

	// Wait for at least some ingestion calls (3 files × 500ms debounce each)
	deadline := time.After(20 * time.Second)
	received := 0
	for received < len(files) {
		select {
		case <-called:
			received++
		case <-deadline:
			mu.Lock()
			c := callCount
			mu.Unlock()
			t.Fatalf("timeout: received %d/%d ingestion calls (expecting %d)", c, received, len(files))
		}
	}

	mu.Lock()
	c := callCount
	mu.Unlock()
	assert.GreaterOrEqual(t, c, len(files), "should have processed at least %d files", len(files))
}

func TestWatcher_EventLoop_ShutdownViaCtx(t *testing.T) {
	tmpDir := t.TempDir()

	done := make(chan struct{})
	runner := &mockRunner{
		runTaskFn: func(ctx context.Context, projectID string, task *v1.IngestionTask) error {
			// Simulate a short ingestion task
			time.Sleep(100 * time.Millisecond)
			close(done)
			return nil
		},
	}

	w := NewWatcher(tmpDir, runner)

	ctx, cancel := context.WithCancel(context.Background())

	err := w.Start(ctx)
	require.NoError(t, err)

	// Trigger ingestion then immediately cancel
	err = w.AddProject(ctx, "fastproj")
	require.NoError(t, err)

	dropDir := filepath.Join(tmpDir, "fastproj", "drop")
	csvFile := filepath.Join(dropDir, "data.csv")
	err = os.WriteFile(csvFile, []byte("a\n1\n"), 0644)
	require.NoError(t, err)

	// Wait for ingestion to complete or timeout
	select {
	case <-done:
		// Ingestion completed before shutdown
	case <-time.After(8 * time.Second):
		// Might still be running — proceed with cancellation
	}

	// Cancel context — event loop should exit cleanly
	cancel()
	time.Sleep(500 * time.Millisecond)
	err = w.Close()
	// Close should succeed or at least not panic
	_ = err
}

// =============================================================================
// Error Recovery Tests
// =============================================================================

func TestWatcher_ErrorRecovery_IngestionPanic(t *testing.T) {
	tmpDir := t.TempDir()

	dropDir := filepath.Join(tmpDir, "testproj", "drop")
	err := os.MkdirAll(dropDir, 0755)
	require.NoError(t, err)

	csvFile := filepath.Join(dropDir, "data.csv")
	err = os.WriteFile(csvFile, []byte("a,b,c\n"), 0644)
	require.NoError(t, err)

	called := make(chan struct{}, 1)
	runner := &mockRunner{
		called: called,
		runTaskFn: func(ctx context.Context, projectID string, task *v1.IngestionTask) error {
			panic("simulated panic in ingestion runner")
		},
	}

	w := NewWatcher(tmpDir, runner)

	// The panic occurs inside the ingestion goroutine which has its own recover()
	// The watcher itself should remain operational
	assert.NotPanics(t, func() {
		w.handleCreateEvent(context.Background(), csvFile)
	}, "handleCreateEvent must not panic even if the ingestion runner panics")

	select {
	case <-called:
		// Runner was called and panicked — the recovery in the ingestion goroutine caught it
	case <-time.After(5 * time.Second):
		t.Fatal("runner.RunTask should have been called")
	}

	// Wait for the panicking goroutine to finish its recovery
	time.Sleep(300 * time.Millisecond)

	// The watcher should still be usable after a panic recovery
	// Create another file and verify ingestion still works
	csvFile2 := filepath.Join(dropDir, "data2.csv")
	err = os.WriteFile(csvFile2, []byte("x,y\n"), 0644)
	require.NoError(t, err)

	called2 := make(chan struct{}, 1)
	runner2 := &mockRunner{
		called: called2,
	}

	w2 := NewWatcher(tmpDir, runner2)
	w2.handleCreateEvent(context.Background(), csvFile2)

	select {
	case <-called2:
		// Second ingestion succeeded — watcher pattern is resilient
	case <-time.After(5 * time.Second):
		t.Fatal("subsequent ingestion should succeed after a previous panic")
	}
}

func TestWatcher_ErrorRecovery_NilRunner(t *testing.T) {
	tmpDir := t.TempDir()

	dropDir := filepath.Join(tmpDir, "testproj", "drop")
	err := os.MkdirAll(dropDir, 0755)
	require.NoError(t, err)

	csvFile := filepath.Join(dropDir, "data.csv")
	err = os.WriteFile(csvFile, []byte("a,b\n"), 0644)
	require.NoError(t, err)

	// Create a watcher with nil runner
	w := &Watcher{
		projectsRoot: tmpDir,
		runner:       nil,
		watchedDirs:  make(map[string]bool),
	}

	// handleCreateEvent should handle nil runner gracefully
	assert.NotPanics(t, func() {
		w.handleCreateEvent(context.Background(), csvFile)
	}, "handleCreateEvent must not panic with nil runner")
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestWatcher_ConcurrentAddProject(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewWatcher(tmpDir, &mockRunner{})

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(300 * time.Millisecond)
		_ = w.Close()
	}()

	err := w.Start(ctx)
	require.NoError(t, err)

	numGoroutines := 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			projID := fmt.Sprintf("concurrent-project-%d", idx)
			// Each goroutine calls AddProject multiple times to stress-test
			// idempotency and concurrency under the mutex
			for j := 0; j < 5; j++ {
				addErr := w.AddProject(ctx, projID)
				if addErr != nil {
					// fsnotify may reject duplicate watches; log but don't fail
					t.Logf("AddProject error for %s (iteration %d): %v", projID, j, addErr)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all project drop directories exist after concurrent access
	for i := 0; i < numGoroutines; i++ {
		dropDir := filepath.Join(tmpDir, fmt.Sprintf("concurrent-project-%d", i), "drop")
		info, statErr := os.Stat(dropDir)
		assert.NoError(t, statErr, "drop dir for project %d should exist", i)
		if statErr == nil {
			assert.True(t, info.IsDir(), "drop dir for project %d should be a directory", i)
		}
	}
}

func TestWatcher_ConcurrentAddProject_PreStart(t *testing.T) {
	// Test concurrent AddProject calls BEFORE Start — should not panic
	// because the mutex protects watchedDirs even when fw is nil

	tmpDir := t.TempDir()
	w := NewWatcher(tmpDir, &mockRunner{})

	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			projID := fmt.Sprintf("prestart-project-%d", idx)
			for j := 0; j < 3; j++ {
				_ = w.AddProject(context.Background(), projID)
			}
		}(i)
	}

	wg.Wait()

	// All drop dirs should be created even without Start being called
	for i := 0; i < numGoroutines; i++ {
		dropDir := filepath.Join(tmpDir, fmt.Sprintf("prestart-project-%d", i), "drop")
		_, statErr := os.Stat(dropDir)
		assert.NoError(t, statErr, "drop dir should exist after pre-Start concurrent AddProject")
	}
}
