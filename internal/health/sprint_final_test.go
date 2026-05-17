package health

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ff3300/aleph-v2/internal/repository"
)

func TestHealthChecker_StartStop(t *testing.T) {
	db, repo := setupFinalTestRepo(t)
	defer db.Close()

	hc := NewHealthChecker(context.Background(), slog.Default(), repo,
		WithInterval(10*time.Millisecond),
		WithAlertCount(2),
	)

	hc.Start(context.Background())
	assert.NotNil(t, hc.ctx)
	assert.NotNil(t, hc.cancel)

	time.Sleep(50 * time.Millisecond)

	hc.Stop()
	assert.NotNil(t, hc.ctx)
}

func TestHealthChecker_run_ContextCancel(t *testing.T) {
	db, repo := setupFinalTestRepo(t)
	defer db.Close()

	hc := NewHealthChecker(context.Background(), slog.Default(), repo,
		WithInterval(100*time.Millisecond),
	)

	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	hc.cancel()
	hc.run()
	assert.NotNil(t, hc.ctx)
}

func TestHealthChecker_checkAll_WithTools(t *testing.T) {
	db, repo := setupFinalTestRepo(t)
	defer db.Close()

	_, err := db.Exec(`INSERT INTO system_tools (id, name, description, code, category, version, health_status, source_type) 
		VALUES ('tool-check', 'Check Tool', 'desc', 'package main', 'utility', '1.0', 'unknown', 'package')`)
	require.NoError(t, err)

	hc := NewHealthChecker(context.Background(), slog.Default(), repo,
		WithAlertCount(1),
	)

	hc.checkAll()

	records := hc.GetHistory("tool-check")
	require.GreaterOrEqual(t, len(records), 1)
	assert.Equal(t, "tool-check", records[0].ToolID)
}

func TestHealthChecker_checkAll_EmptyTools(t *testing.T) {
	db, repo := setupFinalTestRepo(t)
	defer db.Close()

	hc := NewHealthChecker(context.Background(), slog.Default(), repo)
	hc.checkAll()
	assert.Empty(t, hc.GetAllHistory())
}

func TestHealthChecker_checkAll_ListToolsError(t *testing.T) {
	repo := setupRepoWithError(t)
	hc := NewHealthChecker(context.Background(), slog.Default(), repo)
	hc.checkAll()
	assert.Empty(t, hc.GetAllHistory())
}

func TestHealthChecker_checkMCPHealth_AllStatuses(t *testing.T) {
	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "mcp-up", Status: "up", LastPing: "1ms"},
		{Name: "mcp-down", Status: "down", LastError: "panic"},
		{Name: "mcp-unk", Status: "unknown"},
		{Name: "mcp-def", Status: "flapping"},
	}}
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithMCPHealth(mockProvider))
	hc.checkMCPHealth()

	all := hc.GetAllHistory()
	assert.Len(t, all, 4)
	assert.Equal(t, StatusHealthy, hc.GetLatestStatus("mcp-up"))
	assert.Equal(t, StatusDown, hc.GetLatestStatus("mcp-down"))
	assert.Equal(t, StatusUnknown, hc.GetLatestStatus("mcp-unk"))
	assert.Equal(t, StatusUnknown, hc.GetLatestStatus("mcp-def"))
}

func setupFinalTestRepo(t *testing.T) (*sql.DB, *repository.MetadataRepository) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_tools (
		id VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL DEFAULT '',
		description VARCHAR NOT NULL DEFAULT '',
		code VARCHAR NOT NULL DEFAULT '',
		category VARCHAR NOT NULL DEFAULT '',
		version VARCHAR NOT NULL DEFAULT '',
		health_status VARCHAR NOT NULL DEFAULT 'unknown',
		source_type VARCHAR NOT NULL DEFAULT ''
	)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)

	return db, repo
}

func setupRepoWithError(t *testing.T) *repository.MetadataRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)

	_, err = db.Exec("DROP TABLE IF EXISTS system_tools")
	require.NoError(t, err)

	return repo
}
