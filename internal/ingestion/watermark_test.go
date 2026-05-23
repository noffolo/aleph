package ingestion

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestWatermarkLifecycle(t *testing.T) {
	db := setupTestDB(t)
	wm := NewWatermarkManager(db)

	// Initially no watermark
	_, err := wm.Get("test_source")
	assert.ErrorIs(t, err, ErrWatermarkNotFound)

	// Set watermark
	now := time.Now()
	err = wm.Set("test_source", now, "cursor_123", `{"key":"val"}`)
	require.NoError(t, err)

	// Get watermark
	got, err := wm.Get("test_source")
	require.NoError(t, err)
	assert.Equal(t, "test_source", got.SourceName)
	assert.WithinDuration(t, now, got.LastRun, time.Millisecond)
	assert.Equal(t, "cursor_123", got.Cursor)
	assert.Equal(t, `{"key":"val"}`, got.Metadata)

	// Update watermark
	now2 := now.Add(time.Hour)
	err = wm.Set("test_source", now2, "cursor_456", "")
	require.NoError(t, err)

	// List all
	all, err := wm.ListAll()
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestWatermarkNotFound(t *testing.T) {
	db := setupTestDB(t)
	wm := NewWatermarkManager(db)

	_, err := wm.Get("nonexistent")
	assert.ErrorIs(t, err, ErrWatermarkNotFound)
}

func TestWatermarkMultipleSources(t *testing.T) {
	db := setupTestDB(t)
	wm := NewWatermarkManager(db)

	now := time.Now()
	require.NoError(t, wm.Set("source_a", now, "c1", ""))
	require.NoError(t, wm.Set("source_b", now.Add(time.Minute), "c2", ""))

	all, err := wm.ListAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestMigrations(t *testing.T) {
	db := setupTestDB(t)
	mm := NewMigrationManager(db)

	// Initially no migrations applied
	v, err := mm.CurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 0, v)

	// Register migrations
	mm.Register(Migration{
		Version: 1,
		Name:    "create_watermark_table",
		Up:      "CREATE TABLE IF NOT EXISTS ingestion_watermark (source_name TEXT PRIMARY KEY, last_run TIMESTAMP NOT NULL, cursor TEXT DEFAULT '', metadata TEXT DEFAULT '')",
	})

	// Run pending
	err = mm.Up()
	require.NoError(t, err)

	// Verify version
	current, err := mm.CurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, current)

	// Running again should be no-op
	err = mm.Up()
	require.NoError(t, err)
	current, err = mm.CurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 1, current)
}

func TestMigrationsOrdered(t *testing.T) {
	db := setupTestDB(t)
	mm := NewMigrationManager(db)

	mm.Register(Migration{Version: 2, Name: "second", Up: "CREATE TABLE IF NOT EXISTS t2 (id INT)"})
	mm.Register(Migration{Version: 1, Name: "first", Up: "CREATE TABLE IF NOT EXISTS t1 (id INT)"})

	err := mm.Up()
	require.NoError(t, err)

	current, err := mm.CurrentVersion()
	require.NoError(t, err)
	assert.Equal(t, 2, current)
}

func TestWatermarkListAllEmpty(t *testing.T) {
	db := setupTestDB(t)
	wm := NewWatermarkManager(db)

	all, err := wm.ListAll()
	require.NoError(t, err)
	assert.Empty(t, all)
}
