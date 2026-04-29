package tracker

import (
	"context"
	"time"
)

type ToolUsage struct {
	ID         string    `db:"id"          json:"id"`
	UserID     string    `db:"user_id"     json:"user_id"`
	ProjectID  string    `db:"project_id"  json:"project_id"`
	ToolName   string    `db:"tool_name"   json:"tool_name"`
	InputHash  string    `db:"input_hash"  json:"input_hash"`
	DurationMs int64     `db:"duration_ms" json:"duration_ms"`
	Success    bool      `db:"success"     json:"success"`
	ErrorMsg   string    `db:"error_msg"   json:"error_msg"`
	Timestamp  time.Time `db:"timestamp"   json:"timestamp"`
}

type ToolUsageStat struct {
	ToolName    string  `db:"tool_name"      json:"tool_name"`
	Count       int     `db:"count"           json:"count"`
	AvgDuration float64 `db:"avg_duration_ms" json:"avg_duration_ms"`
	SuccessRate float64 `db:"success_rate"    json:"success_rate"`
}

type Tracker interface {
	Record(ctx context.Context, usage ToolUsage) error
	MostUsedTools(ctx context.Context, userID string, limit int, since time.Time) ([]ToolUsageStat, error)
	ToolSequences(ctx context.Context, userID string, limit int) ([][]string, error)
}
