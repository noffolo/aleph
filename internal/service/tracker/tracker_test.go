package tracker

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "github.com/marcboeker/go-duckdb"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tool_usage (
		id          VARCHAR PRIMARY KEY,
		user_id     VARCHAR NOT NULL,
		project_id  VARCHAR NOT NULL,
		tool_name   VARCHAR NOT NULL,
		input_hash  VARCHAR,
		duration_ms BIGINT,
		success     BOOLEAN DEFAULT TRUE,
		error_msg   VARCHAR DEFAULT '',
		timestamp   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func insertUsage(t *testing.T, db *sql.DB, usage ToolUsage) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO tool_usage (id, user_id, project_id, tool_name, input_hash, duration_ms, success, error_msg, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usage.ID, usage.UserID, usage.ProjectID, usage.ToolName,
		usage.InputHash, usage.DurationMs, usage.Success, usage.ErrorMsg, usage.Timestamp,
	)
	if err != nil {
		t.Fatalf("failed to insert usage: %v", err)
	}
}

func TestRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	usage := ToolUsage{
		UserID:     "user1",
		ProjectID:  "proj1",
		ToolName:   "search_data",
		DurationMs: 150,
		Success:    true,
		Timestamp:  time.Now(),
	}

	err := tr.Record(ctx, usage)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tool_usage WHERE user_id = ?", "user1").Scan(&count)
	if err != nil {
		t.Fatalf("query count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestRecordWithGeneratedID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	usage := ToolUsage{
		UserID:    "user2",
		ProjectID: "proj2",
		ToolName:  "analyze",
	}

	err := tr.Record(ctx, usage)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	var id string
	err = db.QueryRow("SELECT id FROM tool_usage WHERE user_id = ?", "user2").Scan(&id)
	if err != nil {
		t.Fatalf("query id failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty generated id")
	}
}

func TestMostUsedTools(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()

	entries := []ToolUsage{
		{ID: "1", UserID: "u1", ProjectID: "p1", ToolName: "search", DurationMs: 100, Success: true, Timestamp: now},
		{ID: "2", UserID: "u1", ProjectID: "p1", ToolName: "search", DurationMs: 200, Success: true, Timestamp: now.Add(1 * time.Minute)},
		{ID: "3", UserID: "u1", ProjectID: "p1", ToolName: "search", DurationMs: 300, Success: false, Timestamp: now.Add(2 * time.Minute)},
		{ID: "4", UserID: "u1", ProjectID: "p1", ToolName: "analyze", DurationMs: 50, Success: true, Timestamp: now.Add(3 * time.Minute)},
		{ID: "5", UserID: "u2", ProjectID: "p2", ToolName: "search", DurationMs: 100, Success: true, Timestamp: now},
	}

	for _, e := range entries {
		insertUsage(t, db, e)
	}

	stats, err := tr.MostUsedTools(ctx, "u1", 10, now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("MostUsedTools() failed: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 tool stats, got %d", len(stats))
	}

	if stats[0].ToolName != "search" {
		t.Errorf("expected top tool 'search', got '%s'", stats[0].ToolName)
	}
	if stats[0].Count != 3 {
		t.Errorf("expected count 3, got %d", stats[0].Count)
	}
	if stats[0].SuccessRate != 2.0/3.0 {
		t.Errorf("expected success_rate %.2f, got %.2f", 2.0/3.0, stats[0].SuccessRate)
	}

	if stats[1].ToolName != "analyze" {
		t.Errorf("expected second tool 'analyze', got '%s'", stats[1].ToolName)
	}
}

func TestMostUsedToolsEmptySince(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	stats, err := tr.MostUsedTools(ctx, "nonexistent", 10, time.Now())
	if err != nil {
		t.Fatalf("MostUsedTools() failed: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d", len(stats))
	}
}

func TestToolSequences(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()

	entries := []ToolUsage{
		{ID: "1", UserID: "u1", ProjectID: "p1", ToolName: "search", Timestamp: now},
		{ID: "2", UserID: "u1", ProjectID: "p1", ToolName: "analyze", Timestamp: now.Add(1 * time.Minute)},
		{ID: "3", UserID: "u1", ProjectID: "p1", ToolName: "search", Timestamp: now.Add(2 * time.Minute)},
		{ID: "4", UserID: "u1", ProjectID: "p1", ToolName: "report", Timestamp: now.Add(1 * time.Hour)},
		{ID: "5", UserID: "u1", ProjectID: "p1", ToolName: "export", Timestamp: now.Add(1*time.Hour + 2*time.Minute)},
	}

	for _, e := range entries {
		insertUsage(t, db, e)
	}

	sequences, err := tr.ToolSequences(ctx, "u1", 10)
	if err != nil {
		t.Fatalf("ToolSequences() failed: %v", err)
	}

	if len(sequences) != 2 {
		t.Fatalf("expected 2 sequences, got %d", len(sequences))
	}

	if len(sequences[0]) != 3 {
		t.Errorf("expected first sequence length 3, got %d: %v", len(sequences[0]), sequences[0])
	}
	if sequences[0][0] != "search" || sequences[0][1] != "analyze" || sequences[0][2] != "search" {
		t.Errorf("unexpected first sequence: %v", sequences[0])
	}

	if len(sequences[1]) != 2 {
		t.Errorf("expected second sequence length 2, got %d: %v", len(sequences[1]), sequences[1])
	}
}

func TestToolSequencesLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 3; i++ {
		base := now.Add(time.Duration(i) * 2 * time.Hour)
		insertUsage(t, db, ToolUsage{
			ID: fmt.Sprintf("s%d-a", i), UserID: "u1", ProjectID: "p1",
			ToolName: "search", Timestamp: base,
		})
		insertUsage(t, db, ToolUsage{
			ID: fmt.Sprintf("s%d-b", i), UserID: "u1", ProjectID: "p1",
			ToolName: "analyze", Timestamp: base.Add(1 * time.Minute),
		})
	}

	sequences, err := tr.ToolSequences(ctx, "u1", 1)
	if err != nil {
		t.Fatalf("ToolSequences() failed: %v", err)
	}

	if len(sequences) != 1 {
		t.Errorf("expected 1 sequence with limit=1, got %d", len(sequences))
	}
}

func TestToolSequencesEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tr := NewDuckDBTracker(db)
	ctx := context.Background()

	sequences, err := tr.ToolSequences(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("ToolSequences() failed: %v", err)
	}
	if len(sequences) != 0 {
		t.Errorf("expected empty sequences, got %d", len(sequences))
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" || id2 == "" {
		t.Error("generateID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateID() should produce unique IDs")
	}
}

func TestExtractToolNameFromSpec(t *testing.T) {
	tests := []struct {
		procedure string
		expected  string
	}{
		{"/aleph.v1.ToolService/ExecuteTool", "toolservice.executeTool"},
		{"/aleph.v1.QueryService/Chat", "queryservice.chat"},
		{"", ""},
		{"/aleph.v1.ToolService/", "toolservice."},
	}

	for _, tt := range tests {
		spec := connect.Spec{Procedure: tt.procedure}
		got := extractToolName(spec)
		if got != tt.expected {
			t.Errorf("extractToolName(%q) = %q, want %q", tt.procedure, got, tt.expected)
		}
	}
}
