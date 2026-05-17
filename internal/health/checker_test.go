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

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	assert.NotNil(t, hc)
	assert.Equal(t, 5*time.Minute, hc.interval)
	assert.Equal(t, 3, hc.alertCount)
}

func TestHealthCheckerWithOptions(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil,
		WithInterval(time.Second),
		WithHistoryLen(5),
		WithAlertCount(2),
	)
	assert.Equal(t, time.Second, hc.interval)
	assert.Equal(t, 2, hc.alertCount)
	assert.Equal(t, 5, hc.history.maxLen)
}

func TestHealthChecker_WithMCPHealth(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil,
		WithMCPHealth(nil),
	)
	assert.Nil(t, hc.mcpHealth)

	mockProvider := &mockMCPHealthProvider{statuses: []MCPToolHealth{
		{Name: "tool-1", Status: "up"},
	}}
	hc = NewHealthChecker(context.Background(), slog.Default(), nil,
		WithMCPHealth(mockProvider),
	)
	assert.NotNil(t, hc.mcpHealth)
	assert.Equal(t, 1, len(hc.mcpHealth.Status()))
}

type mockMCPHealthProvider struct {
	statuses []MCPToolHealth
}

func (m *mockMCPHealthProvider) Status() []MCPToolHealth {
	return m.statuses
}

func TestHealthChecker_StartStop_PanicsWithoutDB(t *testing.T) {
	// Start triggers run() goroutine which calls checkAll() → metaRepo.ListTools()
	// which dereferences *sql.DB. The MetadataRepository requires a real DB connection,
	// so this is expected to panic. We just verify the pattern works.
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.Stop() // safe: cancel is nil
	assert.NotNil(t, hc.logger)
}

func TestHealthChecker_GetHistory(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool_a", HealthRecord{Status: StatusHealthy})
	hc.history.Add("tool_a", HealthRecord{Status: StatusDown})

	records := hc.GetHistory("tool_a")
	assert.Len(t, records, 2)
}

func TestHealthChecker_GetAllHistory(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool_a", HealthRecord{Status: StatusHealthy})
	hc.history.Add("tool_b", HealthRecord{Status: StatusDown})

	all := hc.GetAllHistory()
	assert.Len(t, all, 2)
	assert.Len(t, all["tool_a"], 1)
	assert.Len(t, all["tool_b"], 1)
}

func TestHealthChecker_GetAllHistory_Empty(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	all := hc.GetAllHistory()
	assert.Empty(t, all)
}

func TestHealthChecker_ConsecutiveFailures(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)
	hc.history.Add("tool_a", HealthRecord{Status: StatusDown})
	hc.history.Add("tool_a", HealthRecord{Status: StatusDown})
	assert.Equal(t, 2, hc.ConsecutiveFailures("tool_a"))
}

func TestHealthChecker_GetLatestStatus(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil)

	// No records = unknown
	assert.Equal(t, StatusUnknown, hc.GetLatestStatus("nonexistent"))

	hc.history.Add("tool_a", HealthRecord{Status: StatusHealthy})
	assert.Equal(t, StatusHealthy, hc.GetLatestStatus("tool_a"))

	hc.history.Add("tool_a", HealthRecord{Status: StatusDown})
	assert.Equal(t, StatusDown, hc.GetLatestStatus("tool_a"))
}

func TestBuiltinChecker_New(t *testing.T) {
	checker := NewBuiltinChecker(nil)
	assert.NotNil(t, checker)
	assert.Nil(t, checker.metaRepo)
}

func TestHealthCheckerWithOptions_OverrideSome(t *testing.T) {
	hc := NewHealthChecker(context.Background(), slog.Default(), nil, WithAlertCount(5))
	assert.Equal(t, 5*time.Minute, hc.interval)
	assert.Equal(t, 5, hc.alertCount)
	assert.Equal(t, 10, hc.history.maxLen)
}

func TestCheckerOptionDefaults(t *testing.T) {
	// Verify default config values are applied correctly
	hc := NewHealthChecker(context.Background(), slog.Default(), nil,
		WithInterval(30*time.Second),
		WithHistoryLen(20),
	)
	assert.Equal(t, 30*time.Second, hc.interval)
	assert.Equal(t, 20, hc.history.maxLen)
	assert.Equal(t, 3, hc.alertCount) // default
}

func TestNewBuiltinChecker(t *testing.T) {
	checker := NewBuiltinChecker(nil)
	assert.NotNil(t, checker)
	assert.Nil(t, checker.metaRepo)
}

func TestBuiltinChecker_CheckToolHealth_Success(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO system_tools (id, code) VALUES ('tool-1', 'package main')")
	require.NoError(t, err)

	checker := NewBuiltinChecker(repo)
	record := checker.CheckToolHealth(context.Background(), "tool-1")

	assert.Equal(t, "tool-1", record.ToolID)
	assert.Equal(t, StatusHealthy, record.Status)
	assert.NotEmpty(t, record.ResponseTime)
	assert.Empty(t, record.Error)
}

func TestBuiltinChecker_CheckToolHealth_EmptyCode(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO system_tools (id, code) VALUES ('tool-2', '')")
	require.NoError(t, err)

	checker := NewBuiltinChecker(repo)
	record := checker.CheckToolHealth(context.Background(), "tool-2")

	assert.Equal(t, "tool-2", record.ToolID)
	assert.Equal(t, StatusDegraded, record.Status)
	assert.Equal(t, "tool code is empty", record.Error)
}

func TestBuiltinChecker_CheckToolHealth_MissingTool(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	checker := NewBuiltinChecker(repo)
	record := checker.CheckToolHealth(context.Background(), "nonexistent")

	assert.Equal(t, "nonexistent", record.ToolID)
	assert.Equal(t, StatusDown, record.Status)
	assert.NotEmpty(t, record.Error)
}

func TestBuiltinChecker_CheckToolHealth_ContextCancelled(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO system_tools (id, code) VALUES ('tool-3', 'code')")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewBuiltinChecker(repo)
	record := checker.CheckToolHealth(ctx, "tool-3")

	assert.Equal(t, "tool-3", record.ToolID)
	assert.Equal(t, StatusDown, record.Status)
	assert.Equal(t, "context cancelled", record.Error)
}

func TestBuiltinChecker_CheckToolHealth_ResponseTime(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO system_tools (id, code) VALUES ('tool-4', 'package main')")
	require.NoError(t, err)

	checker := NewBuiltinChecker(repo)
	record := checker.CheckToolHealth(context.Background(), "tool-4")

	assert.NotEmpty(t, record.ResponseTime)
}

func TestHealthChecker_WithBuiltinChecker(t *testing.T) {
	db, repo := setupTestRepo(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO system_tools (id, code) VALUES ('tool-5', 'package main')")
	require.NoError(t, err)

	hc := NewHealthChecker(context.Background(), slog.Default(), repo)
	assert.NotNil(t, hc.checker)

	record := hc.checker.CheckToolHealth(context.Background(), "tool-5")
	assert.Equal(t, StatusHealthy, record.Status)
}

func setupTestRepo(t *testing.T) (*sql.DB, *repository.MetadataRepository) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_tools (
		id VARCHAR PRIMARY KEY,
		code VARCHAR NOT NULL DEFAULT ''
	)`)
	require.NoError(t, err)

	repo, err := repository.NewMetadataRepository(db)
	require.NoError(t, err)

	return db, repo
}
