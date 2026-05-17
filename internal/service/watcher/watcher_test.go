package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRunner struct {
	runTaskFn func(ctx context.Context, projectID string, task *v1.IngestionTask) error
}

func (m *mockRunner) RunTask(ctx context.Context, projectID string, task *v1.IngestionTask) error {
	if m.runTaskFn != nil {
		return m.runTaskFn(ctx, projectID, task)
	}
	return nil
}

func TestNewWatcher(t *testing.T) {
	t.Parallel()
	w := NewWatcher("/tmp/test-projects", &mockRunner{})
	require.NotNil(t, w)
	assert.Equal(t, "/tmp/test-projects", w.projectsRoot)
	assert.Empty(t, w.watchedDirs)
	assert.Nil(t, w.fw)
}

func TestAddProject_CreatesDropDir(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewWatcher(tmpDir, &mockRunner{})

	err := w.AddProject(context.Background(), "test-project")
	require.NoError(t, err)

	dropDir := filepath.Join(tmpDir, "test-project", "drop")
	_, err = os.Stat(dropDir)
	assert.NoError(t, err, "drop directory should exist")
}

func TestAddProject_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewWatcher(tmpDir, &mockRunner{})

	err := w.AddProject(context.Background(), "test-project")
	require.NoError(t, err)

	err = w.AddProject(context.Background(), "test-project")
	assert.NoError(t, err, "second add should be idempotent")
}

func TestWatchProject_DelegatesToAddProject(t *testing.T) {
	tmpDir := t.TempDir()
	w := NewWatcher(tmpDir, &mockRunner{})

	err := w.WatchProject(context.Background(), "test-project")
	require.NoError(t, err)
}

func TestClose_Idempotent(t *testing.T) {
	w := NewWatcher("/tmp", &mockRunner{})
	err := w.Close()
	assert.NoError(t, err)

	err = w.Close()
	assert.NoError(t, err, "second close should not panic")
}

func TestIsFileStable_NonExistentFile(t *testing.T) {
	stable := isFileStable("/tmp/nonexistent-file-12345-test")
	assert.False(t, stable)
}

func TestIsFileStable_StableFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(filePath, []byte("a,b,c\n1,2,3"), 0644)
	require.NoError(t, err)

	stable := isFileStable(filePath)
	assert.True(t, stable)
}

func TestKnownExts(t *testing.T) {
	assert.Equal(t, "csv", knownExts[".csv"])
	assert.Equal(t, "csv", knownExts[".tsv"])
	assert.Equal(t, "custom_code", knownExts[".json"])
	assert.Equal(t, "copy", knownExts[".parquet"])
}

func TestClose_WithWatcher(t *testing.T) {
	w := NewWatcher("/tmp", &mockRunner{})
	w.fw = nil
	err := w.Close()
	assert.NoError(t, err)
}
