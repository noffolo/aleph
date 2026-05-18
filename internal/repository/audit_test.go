package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		EndTime:   now.Add(2*time.Hour + 30*time.Minute),
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

func TestAuditRepository_QueryByResourceID(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	entries := []AuditEntry{
		{ID: 1, UserID: "user1", Action: "create", ResourceType: "agent", ResourceID: "agent-1", Timestamp: now},
		{ID: 2, UserID: "user1", Action: "create", ResourceType: "agent", ResourceID: "agent-2", Timestamp: now},
		{ID: 3, UserID: "user1", Action: "update", ResourceType: "agent", ResourceID: "agent-1", Timestamp: now},
	}
	for _, e := range entries {
		require.NoError(t, repo.InsertAuditLog(ctx, e))
	}

	results, err := repo.QueryAuditLog(ctx, AuditFilters{ResourceID: "agent-1"})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "agent-1", r.ResourceID)
	}

	results, err = repo.QueryAuditLog(ctx, AuditFilters{ResourceID: "nonexistent"})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestAuditRepository_QueryCombinedFilters(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	inserts := []AuditEntry{
		{ID: 1, UserID: "alice", Action: "create", ResourceType: "agent", ResourceID: "a1", Timestamp: now},
		{ID: 2, UserID: "alice", Action: "update", ResourceType: "agent", ResourceID: "a1", Timestamp: now.Add(time.Hour)},
		{ID: 3, UserID: "bob", Action: "create", ResourceType: "tool", ResourceID: "t1", Timestamp: now},
		{ID: 4, UserID: "alice", Action: "delete", ResourceType: "skill", ResourceID: "s1", Timestamp: now.Add(2 * time.Hour)},
	}
	for _, e := range inserts {
		require.NoError(t, repo.InsertAuditLog(ctx, e))
	}

	tests := []struct {
		name    string
		filters AuditFilters
		want    int
	}{
		{"UserID + Action", AuditFilters{UserID: "alice", Action: "create"}, 1},
		{"UserID + ResourceType", AuditFilters{UserID: "alice", ResourceType: "agent"}, 2},
		{"Action + ResourceType", AuditFilters{Action: "create", ResourceType: "agent"}, 1},
		{"UserID + Action + ResourceType", AuditFilters{UserID: "alice", Action: "update", ResourceType: "agent"}, 1},
		{"UserID + TimeRange", AuditFilters{UserID: "alice", StartTime: now, EndTime: now.Add(90 * time.Minute)}, 2},
		{"All filters", AuditFilters{UserID: "bob", Action: "create", ResourceType: "tool", ResourceID: "t1"}, 1},
		{"NoMatch all filters", AuditFilters{UserID: "alice", Action: "delete", ResourceType: "agent"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.QueryAuditLog(ctx, tt.filters)
			require.NoError(t, err)
			assert.Len(t, results, tt.want)
		})
	}
}

func TestAuditRepository_InsertWithProjectID(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	entry := AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "create",
		ResourceType: "agent",
		ResourceID:   "agent-1",
		ProjectID:    "project-abc",
		Timestamp:    now,
	}

	err := repo.InsertAuditLog(ctx, entry)
	require.NoError(t, err)

	results, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "project-abc", results[0].ProjectID)
}

func TestAuditRepository_DiffRoundtrip(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	diff := json.RawMessage(`{"name":"test-agent","version":2,"tags":["ai","data"]}`)
	entry := AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "create",
		ResourceType: "agent",
		ResourceID:   "agent-1",
		Timestamp:    now,
		Diff:         diff,
	}

	err := repo.InsertAuditLog(ctx, entry)
	require.NoError(t, err)

	results, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.NotEmpty(t, results[0].Diff)
	assert.Contains(t, string(results[0].Diff), "test-agent")
	assert.Contains(t, string(results[0].Diff), "ai")
}

func TestAuditRepository_InsertError_ContextCancelled(t *testing.T) {
	repo := setupAuditRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	entry := AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "create",
		ResourceType: "agent",
		ResourceID:   "agent-1",
		Timestamp:    time.Now(),
	}

	err := repo.InsertAuditLog(ctx, entry)
	if err != nil {
		assert.Contains(t, err.Error(), "insertAuditLog")
	}
}

func TestAuditRepository_QueryError_ContextCancelled(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.InsertAuditLog(ctx, AuditEntry{
		ID:           1,
		UserID:       "user1",
		Action:       "create",
		ResourceType: "agent",
		ResourceID:   "agent-1",
		Timestamp:    time.Now(),
	}))

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.QueryAuditLog(cancelledCtx, AuditFilters{})
	if err != nil {
		assert.Error(t, err)
	}
}

func TestAuditRepository_QueryPagination(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	for i := 1; i <= 10; i++ {
		require.NoError(t, repo.InsertAuditLog(ctx, AuditEntry{
			ID:           int64(i),
			UserID:       "user1",
			Action:       "create",
			ResourceType: "agent",
			ResourceID:   "agent-1",
			Timestamp:    now.Add(time.Duration(i) * time.Minute),
		}))
	}

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantLen int
	}{
		{"first page", 3, 0, 3},
		{"second page", 3, 3, 3},
		{"third page", 3, 6, 3},
		{"fourth page partial", 3, 9, 1},
		{"beyond data", 5, 15, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.QueryAuditLog(ctx, AuditFilters{Limit: tt.limit, Offset: tt.offset})
			require.NoError(t, err)
			assert.Len(t, results, tt.wantLen)
		})
	}

	results, err := repo.QueryAuditLog(ctx, AuditFilters{Limit: 3})
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.True(t, results[0].Timestamp.After(results[1].Timestamp) || results[0].Timestamp.Equal(results[1].Timestamp))
}

func TestAuditRepository_MultipleResourceTypes(t *testing.T) {
	repo := setupAuditRepo(t)
	ctx := context.Background()
	now := time.Now()

	types := []string{"agent", "tool", "skill", "ingestion", "task", "api_key", "project"}
	for i, rt := range types {
		require.NoError(t, repo.InsertAuditLog(ctx, AuditEntry{
			ID:           int64(i + 1),
			UserID:       "user1",
			Action:       "create",
			ResourceType: rt,
			ResourceID:   rt + "-1",
			Timestamp:    now,
		}))
	}

	all, err := repo.QueryAuditLog(ctx, AuditFilters{})
	require.NoError(t, err)
	assert.Len(t, all, len(types))

	for _, rt := range types {
		results, err := repo.QueryAuditLog(ctx, AuditFilters{ResourceType: rt})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, rt, results[0].ResourceType)
	}
}
