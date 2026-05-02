package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/marcboeker/go-duckdb"
)

func setupAuditRepo(t *testing.T) *AuditRepository {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE SEQUENCE audit_log_id_seq START 1`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS audit_log (
		id BIGINT DEFAULT nextval('audit_log_id_seq'),
		user_id VARCHAR,
		action VARCHAR,
		resource_type VARCHAR,
		resource_id VARCHAR,
		project_id VARCHAR,
		timestamp TIMESTAMP,
		diff VARCHAR
	)`)
	require.NoError(t, err)

	return NewAuditRepository(db)
}

func TestNewAuditRepository(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo := NewAuditRepository(db)
	assert.NotNil(t, repo)
}

func TestAuditRepository_InsertAndQuery(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	entry := AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "create",
		ResourceType: "agent",
		ResourceID:   "agent-1",
		Timestamp:    now,
		Diff:         json.RawMessage(`{"name": "test-agent"}`),
	}

	err := repo.InsertAuditLog(ctx, entry)
	require.NoError(t, err)

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "user1", entries[0].UserID)
	assert.Equal(t, "create", entries[0].Action)
	assert.Equal(t, "agent", entries[0].ResourceType)
	assert.Equal(t, "agent-1", entries[0].ResourceID)
}

func TestAuditRepository_QueryByUser(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 3; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now,
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{UserID: "user1"})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestAuditRepository_QueryByResourceType(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 3; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now,
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{ResourceType: "agent"})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestAuditRepository_QueryByAction(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 2; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now,
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{Action: "create"})
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	entries, err = repo.QueryAuditLog(ctx, AuditFilters{Action: "delete"})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestAuditRepository_QueryByTimeRange(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 3; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now.Add(time.Duration(i) * time.Hour),
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{
		StartTime: now.Add(30 * time.Minute),
		EndTime:   now.Add(2 * time.Hour + 30 * time.Minute),
	})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestAuditRepository_QueryWithLimit(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 5; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now,
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestAuditRepository_QueryWithOffset(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 5; i++ {
		entry := AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now,
		}
		require.NoError(t, repo.InsertAuditLog(ctx, entry))
	}

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{Limit: 10, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestAuditRepository_EmptyDB(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestAuditRepository_InsertWithNilDiff(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	entry := AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "delete",
		ResourceType: "tool",
		ResourceID:   "tool-1",
		Timestamp:    time.Now(),
		Diff:         nil,
	}

	err := repo.InsertAuditLog(ctx, entry)
	assert.NoError(t, err)

	entries, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	// When diff is NULL in the database, QueryAuditLog returns an empty json.RawMessage
	assert.Empty(t, entries[0].Diff)
}
