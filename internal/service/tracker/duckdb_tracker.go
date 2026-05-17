package tracker

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

// duckDBClient abstracts the database connection to avoid
// direct *sql.DB dependency, enabling write-through serialization.
type duckDBClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// DuckDBTracker implements the Tracker interface backed by DuckDB.
type DuckDBTracker struct {
	db duckDBClient
}

// NewDuckDBTracker creates a new DuckDB-backed tracker.
func NewDuckDBTracker(db duckDBClient) *DuckDBTracker {
	return &DuckDBTracker{db: db}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x%x", time.Now().UnixNano(), b)
}

// Record inserts a tool usage record into DuckDB.
func (t *DuckDBTracker) Record(ctx context.Context, usage ToolUsage) error {
	if usage.ID == "" {
		usage.ID = generateID()
	}
	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	_, err := t.db.ExecContext(ctx,
		`INSERT INTO tool_usage (id, user_id, project_id, tool_name, input_hash, duration_ms, success, error_msg, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usage.ID, usage.UserID, usage.ProjectID, usage.ToolName,
		usage.InputHash, usage.DurationMs, usage.Success, usage.ErrorMsg, usage.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("record tool usage: %w", err)
	}
	return nil
}

// MostUsedTools returns the most frequently used tools for a user since a given time.
func (t *DuckDBTracker) MostUsedTools(ctx context.Context, userID string, limit int, since time.Time) ([]ToolUsageStat, error) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT tool_name,
		        COUNT(*) as count,
		        AVG(duration_ms) as avg_duration_ms,
		        AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) as success_rate
		 FROM tool_usage
		 WHERE user_id = ? AND timestamp >= ?
		 GROUP BY tool_name
		 ORDER BY count DESC
		 LIMIT ?`,
		userID, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query most used tools: %w", err)
	}
	defer rows.Close()

	var stats []ToolUsageStat
	for rows.Next() {
		var stat ToolUsageStat
		if err := rows.Scan(&stat.ToolName, &stat.Count, &stat.AvgDuration, &stat.SuccessRate); err != nil {
			return nil, fmt.Errorf("scan tool usage stat: %w", err)
		}
		stats = append(stats, stat)
	}
	if stats == nil {
		stats = []ToolUsageStat{}
	}
	return stats, rows.Err()
}

// ToolSequences returns consecutive tool call sequences for a user.
// Each inner slice is a sequence of tool names ordered by timestamp.
func (t *DuckDBTracker) ToolSequences(ctx context.Context, userID string, limit int) ([][]string, error) {
	rows, err := t.db.QueryContext(ctx,
		`SELECT tool_name, timestamp
		 FROM tool_usage
		 WHERE user_id = ?
		 ORDER BY timestamp ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query tool sequences: %w", err)
	}
	defer rows.Close()

	type entry struct {
		name      string
		timestamp time.Time
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.name, &e.timestamp); err != nil {
			return nil, fmt.Errorf("scan tool sequence entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Group consecutive tool calls within 5-minute windows into sequences
	var sequences [][]string
	const window = 5 * time.Minute

	for i := 0; i < len(entries); i++ {
		seq := []string{entries[i].name}
		for j := i + 1; j < len(entries); j++ {
			if entries[j].timestamp.Sub(entries[j-1].timestamp) <= window {
				seq = append(seq, entries[j].name)
				i = j
			} else {
				break
			}
		}
		if len(seq) > 0 {
			sequences = append(sequences, seq)
		}
	}

	if sequences == nil {
		sequences = [][]string{}
	}

	if limit > 0 && len(sequences) > limit {
		sequences = sequences[len(sequences)-limit:]
	}

	return sequences, nil
}
