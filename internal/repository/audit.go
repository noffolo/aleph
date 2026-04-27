package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

type AuditEntry struct {
	ID           int64
	UserID       string
	Action       string // "create", "update", "delete"
	ResourceType string // "agent", "tool", "skill", "ingestion", "task", "api_key", "notification"
	ResourceID   string
	Timestamp    time.Time
	Diff         json.RawMessage // nullable JSON diff
}

type AuditFilters struct {
	UserID       string
	ResourceType string
	ResourceID   string
	Action       string
	StartTime    time.Time
	EndTime      time.Time
	Limit        int
	Offset       int
}

func (r *AuditRepository) InsertAuditLog(ctx context.Context, entry AuditEntry) error {
	query := `
	INSERT INTO audit_log (user_id, action, resource_type, resource_id, timestamp, diff)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		entry.UserID,
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		entry.Timestamp,
		entry.Diff,
	)
	return fmt.Errorf("insertAuditLog: %w", err)
}

func (r *AuditRepository) QueryAuditLog(ctx context.Context, filters AuditFilters) ([]AuditEntry, error) {
	query := `
	SELECT id, user_id, action, resource_type, resource_id, timestamp, diff
	FROM audit_log
	WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filters.UserID != "" {
		query += " AND user_id = $" + string(rune('0'+argIdx))
		args = append(args, filters.UserID)
		argIdx++
	}
	if filters.ResourceType != "" {
		query += " AND resource_type = $" + string(rune('0'+argIdx))
		args = append(args, filters.ResourceType)
		argIdx++
	}
	if filters.ResourceID != "" {
		query += " AND resource_id = $" + string(rune('0'+argIdx))
		args = append(args, filters.ResourceID)
		argIdx++
	}
	if filters.Action != "" {
		query += " AND action = $" + string(rune('0'+argIdx))
		args = append(args, filters.Action)
		argIdx++
	}
	if !filters.StartTime.IsZero() {
		query += " AND timestamp >= $" + string(rune('0'+argIdx))
		args = append(args, filters.StartTime)
		argIdx++
	}
	if !filters.EndTime.IsZero() {
		query += " AND timestamp <= $" + string(rune('0'+argIdx))
		args = append(args, filters.EndTime)
		argIdx++
	}

	query += " ORDER BY timestamp DESC, id DESC"

	if filters.Limit > 0 {
		query += " LIMIT $" + string(rune('0'+argIdx))
		args = append(args, filters.Limit)
		argIdx++
	}
	if filters.Offset > 0 {
		query += " OFFSET $" + string(rune('0'+argIdx))
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var diff sql.NullString
		err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.Action,
			&entry.ResourceType,
			&entry.ResourceID,
			&entry.Timestamp,
			&diff,
		)
		if err != nil {
			return nil, err
		}
		if diff.Valid {
			entry.Diff = json.RawMessage(diff.String)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}