package tracker

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock DB client for error path testing ──────────────────────────

type errorDBClient struct{}

func (e *errorDBClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errors.New("mock exec error")
}

func (e *errorDBClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, errors.New("mock query error")
}

// ── NewDuckDBTracker ──────────────────────────────────────────────

func TestNewDuckDBTracker_Happy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	assert.NotNil(t, tr, "tracker should be non-nil")
	assert.Equal(t, duckDBClient(db), tr.db, "db field should match input")
}

func TestNewDuckDBTracker_NilDB(t *testing.T) {
	tr := NewDuckDBTracker(nil)
	assert.NotNil(t, tr, "tracker should still be non-nil with nil db")
	assert.Nil(t, tr.db, "db field should be nil")
}

func TestNewDuckDBTracker_ImplementsTrackerInterface(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	var _ Tracker = tr // compile-time check; verify at runtime
	assert.Implements(t, (*Tracker)(nil), tr, "DuckDBTracker must implement Tracker")
}

// ── generateID ─────────────────────────────────────────────────────

func TestGenerateID_Happy(t *testing.T) {
	id := generateID()
	assert.NotEmpty(t, id, "generateID() must return a non-empty string")

	// Should contain both unix nano prefix (hex) and random bytes (hex)
	assert.Greater(t, len(id), 16, "ID should be longer than 16 hex chars (8 bytes unix + 8 bytes random)")
}

func TestGenerateID_Unique(t *testing.T) {
	const iterations = 100
	seen := make(map[string]bool, iterations)
	for range iterations {
		id := generateID()
		assert.NotContains(t, seen, id, "duplicate ID generated: %s", id)
		seen[id] = true
	}
}

func TestGenerateID_LengthConsistency(t *testing.T) {
	for range 50 {
		id := generateID()
		assert.Len(t, id, 32, "generateID() must return exactly 32 hex chars (16 bytes)")
	}
}

// ── Record ─────────────────────────────────────────────────────────

func TestRecord_HappyWithAllFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()

	usage := ToolUsage{
		ID:         "custom-id-001",
		UserID:     "user-record-1",
		ProjectID:  "proj-record-1",
		ToolName:   "search",
		InputHash:  "abc123hash",
		DurationMs: 250,
		Success:    true,
		ErrorMsg:   "",
		Timestamp:  now,
	}

	err := tr.Record(ctx, usage)
	require.NoError(t, err)

	var (
		storedID     string
		storedUser   string
		storedProj   string
		storedTool   string
		storedHash   string
		storedDur    int64
		storedOk     bool
		storedErrMsg string
	)
	err = db.QueryRow(
		`SELECT id, user_id, project_id, tool_name, input_hash, duration_ms, success, error_msg
		 FROM tool_usage WHERE id = 'custom-id-001'`,
	).Scan(&storedID, &storedUser, &storedProj, &storedTool, &storedHash, &storedDur, &storedOk, &storedErrMsg)
	require.NoError(t, err)

	assert.Equal(t, "custom-id-001", storedID)
	assert.Equal(t, "user-record-1", storedUser)
	assert.Equal(t, "proj-record-1", storedProj)
	assert.Equal(t, "search", storedTool)
	assert.Equal(t, "abc123hash", storedHash)
	assert.Equal(t, int64(250), storedDur)
	assert.True(t, storedOk)
	assert.Empty(t, storedErrMsg)
}

func TestRecord_EdgeDefaultTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	beforeRecord := time.Now()
	usage := ToolUsage{
		ID:        "edge-ts-001",
		UserID:    "user-edge-ts",
		ProjectID: "proj-edge-ts",
		ToolName:  "edge_tool",
		Success:   true,
		// Timestamp left zero — should be auto-filled
	}
	err := tr.Record(ctx, usage)
	require.NoError(t, err)
	afterRecord := time.Now()

	var storedTs time.Time
	err = db.QueryRow("SELECT timestamp FROM tool_usage WHERE id = 'edge-ts-001'").Scan(&storedTs)
	require.NoError(t, err)

	assert.False(t, storedTs.IsZero(), "timestamp should be auto-filled")
	assert.True(t, !storedTs.Before(beforeRecord.Add(-time.Second)), "timestamp should be >= before record")
	assert.True(t, !storedTs.After(afterRecord.Add(time.Second)), "timestamp should be <= after record")
}

func TestRecord_ErrorDBExecFails(t *testing.T) {
	mock := &errorDBClient{}
	tr := NewDuckDBTracker(mock)

	usage := ToolUsage{
		ID:        "will-fail",
		UserID:    "u1",
		ProjectID: "p1",
		ToolName:  "tool",
		Timestamp: time.Now(),
	}
	err := tr.Record(context.Background(), usage)
	assert.Error(t, err, "Record must return error when DB ExecContext fails")
	assert.Contains(t, err.Error(), "record tool usage", "error must wrap with context")
	assert.Contains(t, err.Error(), "mock exec error", "error must contain underlying error")
}

// ── MostUsedTools ─────────────────────────────────────────────────

func TestMostUsedTools_HappyReturnsSortedStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()
	since := now.Add(-1 * time.Hour)

	// Insert: "search" (3 calls), "analyze" (2 calls), "report" (1 call)
	entries := []ToolUsage{
		{ID: "mu-1", UserID: "mu-user", ProjectID: "mu-p", ToolName: "search", DurationMs: 100, Success: true, Timestamp: now},
		{ID: "mu-2", UserID: "mu-user", ProjectID: "mu-p", ToolName: "analyze", DurationMs: 50, Success: true, Timestamp: now.Add(time.Minute)},
		{ID: "mu-3", UserID: "mu-user", ProjectID: "mu-p", ToolName: "search", DurationMs: 200, Success: false, Timestamp: now.Add(2 * time.Minute)},
		{ID: "mu-4", UserID: "mu-user", ProjectID: "mu-p", ToolName: "search", DurationMs: 150, Success: true, Timestamp: now.Add(3 * time.Minute)},
		{ID: "mu-5", UserID: "mu-user", ProjectID: "mu-p", ToolName: "analyze", DurationMs: 80, Success: true, Timestamp: now.Add(4 * time.Minute)},
		{ID: "mu-6", UserID: "mu-user", ProjectID: "mu-p", ToolName: "report", DurationMs: 500, Success: true, Timestamp: now.Add(5 * time.Minute)},
	}
	for _, e := range entries {
		insertUsage(t, db, e)
	}

	stats, err := tr.MostUsedTools(ctx, "mu-user", 10, since)
	require.NoError(t, err)
	require.Len(t, stats, 3, "should return 3 distinct tools")

	// Ordered by count desc: search(3), analyze(2), report(1)
	assert.Equal(t, "search", stats[0].ToolName)
	assert.Equal(t, 3, stats[0].Count)
	assert.InDelta(t, 150.0, stats[0].AvgDuration, 0.01)     // (100+200+150)/3
	assert.InDelta(t, 2.0/3.0, stats[0].SuccessRate, 0.001)     // 2 of 3 succeeded

	assert.Equal(t, "analyze", stats[1].ToolName)
	assert.Equal(t, 2, stats[1].Count)
	assert.InDelta(t, 65.0, stats[1].AvgDuration, 0.01)         // (50+80)/2
	assert.InDelta(t, 1.0, stats[1].SuccessRate, 0.001)

	assert.Equal(t, "report", stats[2].ToolName)
	assert.Equal(t, 1, stats[2].Count)
}

func TestMostUsedTools_EdgeUserNotExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	// User with no records + zero time since
	stats, err := tr.MostUsedTools(ctx, "ghost-user", 10, time.Time{})
	require.NoError(t, err)
	assert.Empty(t, stats, "should return empty slice for unknown user")
	assert.NotNil(t, stats, "should return non-nil empty slice")
}

func TestMostUsedTools_ErrorDBQueryFails(t *testing.T) {
	mock := &errorDBClient{}
	tr := NewDuckDBTracker(mock)

	stats, err := tr.MostUsedTools(context.Background(), "any-user", 10, time.Time{})
	assert.Error(t, err, "MostUsedTools must return error on DB failure")
	assert.Nil(t, stats, "stats must be nil on error")
	assert.Contains(t, err.Error(), "query most used tools")
	assert.Contains(t, err.Error(), "mock query error")
}

// ── ToolSequences ─────────────────────────────────────────────────

func TestToolSequences_HappyGroupedSequences(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()

	// Within-window group: search → analyze → search (all within 5 min)
	// Separate group:  report → export (within 5 min, but 1 hour after first group)
	entries := []ToolUsage{
		{ID: "ts-1", UserID: "ts-user", ProjectID: "ts-p", ToolName: "search", Timestamp: now, Success: true},
		{ID: "ts-2", UserID: "ts-user", ProjectID: "ts-p", ToolName: "analyze", Timestamp: now.Add(time.Minute), Success: true},
		{ID: "ts-3", UserID: "ts-user", ProjectID: "ts-p", ToolName: "search", Timestamp: now.Add(2 * time.Minute), Success: true},
		{ID: "ts-4", UserID: "ts-user", ProjectID: "ts-p", ToolName: "report", Timestamp: now.Add(time.Hour), Success: true},
		{ID: "ts-5", UserID: "ts-user", ProjectID: "ts-p", ToolName: "export", Timestamp: now.Add(time.Hour + 2*time.Minute), Success: true},
	}
	for _, e := range entries {
		insertUsage(t, db, e)
	}

	seqs, err := tr.ToolSequences(ctx, "ts-user", 10)
	require.NoError(t, err)
	require.Len(t, seqs, 2, "should have 2 sequences (within-window grouped)")

	assert.Equal(t, []string{"search", "analyze", "search"}, seqs[0], "first sequence should be search→analyze→search")
	assert.Equal(t, []string{"report", "export"}, seqs[1], "second sequence should be report→export")
}

func TestToolSequences_EdgeNoMatchingUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	// Insert data for a different user to ensure filtering works
	insertUsage(t, db, ToolUsage{
		ID: "other-seq", UserID: "other-user", ProjectID: "p", ToolName: "x",
		Timestamp: time.Now(), Success: true,
	})

	seqs, err := tr.ToolSequences(ctx, "no-seq-user", 10)
	require.NoError(t, err)
	assert.Empty(t, seqs, "should return empty for user with no records")
	assert.NotNil(t, seqs, "should return non-nil empty slice")
}

func TestToolSequences_ErrorDBQueryFails(t *testing.T) {
	mock := &errorDBClient{}
	tr := NewDuckDBTracker(mock)

	seqs, err := tr.ToolSequences(context.Background(), "any-user", 10)
	assert.Error(t, err, "ToolSequences must return error on DB failure")
	assert.Nil(t, seqs, "sequences must be nil on error")
	assert.Contains(t, err.Error(), "query tool sequences")
	assert.Contains(t, err.Error(), "mock query error")
}
