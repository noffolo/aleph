package humanecosystems

import (
	"context"
	"testing"

	"github.com/ff3300/aleph-v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDuckDBLayer creates an in-memory DuckDB with the system_tools table
// for testing queryViz/queryRelational/queryProfiles code paths.
func setupDuckDBLayer(t *testing.T) *DuckDBLayer {
	t.Helper()
	db, err := storage.NewDuckDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Cleanup() })

	dbl := NewDuckDBLayer(db)

	_, err = dbl.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS system_tools (
			id VARCHAR,
			name VARCHAR,
			category VARCHAR,
			source_type VARCHAR
		)`)
	require.NoError(t, err)

	// Insert test data
	_, err = dbl.ExecContext(context.Background(),
		`INSERT INTO system_tools VALUES
			('t1', 'Tool Alpha', 'human-ecosystems', 'package'),
			('t2', 'Tool Beta',  'human-ecosystems', 'package'),
			('t3', 'Tool Gamma', 'analysis', 'package'),
			('t4', 'Tool Delta', 'other', 'other')`)
	require.NoError(t, err)

	return dbl
}

// ---- queryViz code path ----

func TestQueryViz_WithDuckDB(t *testing.T) {
	dbl := setupDuckDBLayer(t)
	tool := NewPluginViz(dbl)

	result, err := tool.Execute(context.Background(), map[string]any{})
	require.NoError(t, err)
	r, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "graph", r["viz_type"])
	assert.False(t, r["is_synthetic"].(bool))
	assert.Contains(t, r, "nodes")
	assert.Contains(t, r, "edges")
	nodesRaw, ok := r["nodes"].([]map[string]any)
	if !ok {
		nodesIface, ok := r["nodes"].([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(nodesIface), 1)
	} else {
		assert.GreaterOrEqual(t, len(nodesRaw), 1)
	}
}

func TestQueryViz_WithScope(t *testing.T) {
	dbl := setupDuckDBLayer(t)
	tool := NewPluginViz(dbl)

	result, err := tool.Execute(context.Background(), map[string]any{
		"viz_type": "heatmap",
		"scope":    "analysis",
	})
	require.NoError(t, err)
	r := result.(map[string]any)
	assert.Equal(t, "heatmap", r["viz_type"])
	assert.Equal(t, "analysis", r["scope"])
}

// ---- queryRelational code path ----

func TestQueryRelational_WithDuckDB(t *testing.T) {
	dbl := setupDuckDBLayer(t)
	tool := NewRelationalEngine(dbl)

	result, err := tool.Execute(context.Background(), map[string]any{
		"entity": "ecosystem-alpha",
	})
	require.NoError(t, err)
	r, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ecosystem-alpha", r["entity"])
	assert.False(t, r["is_synthetic"].(bool))
	assert.Contains(t, r, "relations")
}

// ---- queryProfiles code path ----

func TestQueryProfiles_WithDuckDB(t *testing.T) {
	dbl := setupDuckDBLayer(t)
	tool := NewResearchProfiles(dbl)

	result, err := tool.Execute(context.Background(), map[string]any{
		"query": "ecosystem analysis",
	})
	require.NoError(t, err)
	r, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ecosystem analysis", r["query"])
	assert.False(t, r["is_synthetic"].(bool))
	assert.Contains(t, r, "profiles")
}

// ---- buildVizOutput default case (unknown viz_type) ----

func TestBuildVizOutput_DefaultCase(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewPluginViz(dbl)

	result, err := tool.Execute(context.Background(), map[string]any{
		"viz_type": "unknown_type",
	})
	require.NoError(t, err)
	r := result.(map[string]any)
	assert.Equal(t, "unknown_type", r["viz_type"])
	assert.Contains(t, r, "nodes")
	assert.Contains(t, r, "edges")
}

// ---- buildVizOutput heatmap case is already tested; ensure timeline ----

// ---- GetRelationalContext: multiple tool IDs ----

func TestGetRelationalContext_MultipleToolIDs(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	tut.RecordUsage(ctx, "user1", "tool_a", "analysis")
	tut.RecordUsage(ctx, "user1", "tool_b", "viz")

	rels, err := tut.GetRelationalContext(ctx, []string{"tool_a", "tool_b"})
	require.NoError(t, err)
	require.Contains(t, rels, "tool_a")
	require.Contains(t, rels, "tool_b")
	// Each tool should see the other as co-used
	assert.NotEmpty(t, rels["tool_a"])
}

func TestGetRelationalContext_NoCoUsage(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	// Record usage by different users at different times — no co-usage within 1 hour
	tut.RecordUsage(ctx, "user_a", "tool_alone", "test")
	tut.RecordUsage(ctx, "user_b", "tool_other", "test")

	rels, err := tut.GetRelationalContext(ctx, []string{"tool_alone"})
	require.NoError(t, err)
	assert.Empty(t, rels["tool_alone"])
}

func TestGetRelationalContext_ToolIDNotInSet(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	tut.RecordUsage(ctx, "user_a", "tool_x", "test")
	tut.RecordUsage(ctx, "user_a", "tool_y", "test")

	// Query for tool_z which has no entries
	rels, err := tut.GetRelationalContext(ctx, []string{"tool_z"})
	require.NoError(t, err)
	assert.Empty(t, rels["tool_z"])
}

// ---- GetTopUsers: limit > len(users) ----

func TestGetTopUsers_LimitExceedsUsers(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	tut.RecordUsage(ctx, "user1", "tool_rare", "test")
	tut.RecordUsage(ctx, "user1", "tool_rare", "test")

	users, err := tut.GetTopUsers(ctx, "tool_rare", 100)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "user1", users[0])
}

func TestGetTopUsers_NoUsers(t *testing.T) {
	tut := NewToolUsageTracker()
	ctx := context.Background()

	users, err := tut.GetTopUsers(ctx, "nonexistent_tool", 10)
	require.NoError(t, err)
	assert.Empty(t, users)
}

// ---- DuckDBLayer methods ----

func TestDuckDBLayer_IsAvailable(t *testing.T) {
	assert.False(t, SyntheticDuckDBLayer().IsAvailable())

	dbl := setupDuckDBLayer(t)
	assert.True(t, dbl.IsAvailable())
}

func TestDuckDBLayer_SchemaContext(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	ctx := dbl.SchemaContext(context.Background(), "")
	assert.Equal(t, context.Background(), ctx)

	ctx2 := dbl.SchemaContext(context.Background(), "proj-123")
	assert.NotNil(t, ctx2)
}

func TestDuckDBLayer_QueryContext_NilDB(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	rows, err := dbl.QueryContext(context.Background(), "SELECT 1")
	assert.Error(t, err)
	assert.Nil(t, rows)
	assert.Contains(t, err.Error(), "duckdb not available")
}

func TestDuckDBLayer_ExecContext_NilDB(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	res, err := dbl.ExecContext(context.Background(), "CREATE TABLE x (a INT)")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "duckdb not available")
}

func TestSyntheticDuckDBLayer(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	assert.NotNil(t, dbl)
	assert.Nil(t, dbl.db)
	assert.False(t, dbl.IsAvailable())
}

func TestSyntheticRowCount(t *testing.T) {
	rows := syntheticRowCount()
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0]["count"])
	assert.Equal(t, true, rows[0]["is_synthetic"])
	assert.Contains(t, rows[0]["message"].(string), "DuckDB unavailable")
	assert.NotEmpty(t, rows[0]["generated_at"])
}

// ---- RelationalEngine synthetic path (deterministic IDs) ----

func TestRelationalEngine_Synthetic_Deterministic(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewRelationalEngine(dbl)
	result, err := tool.Execute(context.Background(), map[string]any{"entity": "test_entity"})
	require.NoError(t, err)
	r := result.(map[string]any)
	assert.True(t, r["is_synthetic"].(bool))
	relations := r["relations"].([]map[string]any)
	assert.GreaterOrEqual(t, len(relations), 3)
}

// ---- ResearchProfiles synthetic path ----

func TestResearchProfiles_Synthetic_DefaultQuery(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewResearchProfiles(dbl)
	result, err := tool.Execute(context.Background(), map[string]any{})
	require.NoError(t, err)
	r := result.(map[string]any)
	assert.True(t, r["is_synthetic"].(bool))
	assert.Equal(t, "default ecosystem analysis", r["query"])
}

// ---- sha256Hash ----

func TestSHA256Hash(t *testing.T) {
	h1 := sha256Hash("hello")
	h2 := sha256Hash("hello")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64)
	h3 := sha256Hash("different")
	assert.NotEqual(t, h1, h3)
}
