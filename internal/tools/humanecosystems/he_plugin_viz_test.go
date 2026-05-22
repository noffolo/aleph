package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PluginViz.Name
// =============================================================================

func TestPluginVizName(t *testing.T) {
	pv := NewPluginViz(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "he_plugin_viz", pv.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, pv.Name(), pv.Name())
	})

	t.Run("edge: non-empty snake_case", func(t *testing.T) {
		assert.NotEmpty(t, pv.Name())
	})
}

// =============================================================================
// PluginViz.Description
// =============================================================================

func TestPluginVizDescription(t *testing.T) {
	pv := NewPluginViz(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, pv.Description())
	})

	t.Run("happy: mentions visualizations and privacy", func(t *testing.T) {
		desc := pv.Description()
		assert.Contains(t, desc, "visualizations")
		assert.Contains(t, desc, "privacy-preserving")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, pv.Description(), pv.Description())
	})
}

// =============================================================================
// NewPluginViz
// =============================================================================

func TestNewPluginViz(t *testing.T) {
	t.Run("happy: creates with synthetic layer", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		require.NotNil(t, pv)
		assert.False(t, pv.db.IsAvailable())
	})

	t.Run("happy: creates with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pv := NewPluginViz(dbl)
		require.NotNil(t, pv)
		assert.True(t, pv.db.IsAvailable())
	})

	t.Run("edge: nil dbl creates working tool", func(t *testing.T) {
		pv := NewPluginViz(nil)
		require.NotNil(t, pv)
		assert.NotEmpty(t, pv.Name())
	})
}

// =============================================================================
// PluginViz.Execute
// =============================================================================

func TestPluginVizExecute(t *testing.T) {
	t.Run("happy: defaults to graph viz_type", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "graph", r["viz_type"])
		assert.Contains(t, r, "nodes")
		assert.Contains(t, r, "edges")
	})

	t.Run("happy: graph viz_type has nodes and edges", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "graph"})
		require.NoError(t, err)
		r := result.(map[string]any)
		nodes := r["nodes"].([]map[string]any)
		edges := r["edges"].([]map[string]any)
		assert.Greater(t, len(nodes), 0)
		assert.GreaterOrEqual(t, len(edges), 1)
	})

	t.Run("happy: heatmap viz_type has matrix", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "heatmap"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "heatmap", r["viz_type"])
		assert.Contains(t, r, "matrix")
	})

	t.Run("happy: timeline viz_type has events", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "timeline"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "timeline", r["viz_type"])
		assert.Contains(t, r, "events")
		events := r["events"].([]map[string]any)
		assert.Len(t, events, 2)
	})

	t.Run("edge: unknown viz_type defaults to graph behavior", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "unknown"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "unknown", r["viz_type"])
		assert.Contains(t, r, "nodes")
		assert.Contains(t, r, "edges")
	})

	t.Run("edge: scope parameter is passed through", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, err := pv.Execute(context.Background(), map[string]any{
			"viz_type": "graph",
			"scope":    "analysis",
		})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "analysis", r["scope"])
	})

	t.Run("edge: node IDs are SHA-256 hex strings", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		result, _ := pv.Execute(context.Background(), map[string]any{"viz_type": "graph"})
		r := result.(map[string]any)
		nodes := r["nodes"].([]map[string]any)
		for _, node := range nodes {
			id := node["id"].(string)
			assert.Len(t, id, 64)
		}
	})

	t.Run("error: never returns error with synthetic layer", func(t *testing.T) {
		pv := NewPluginViz(SyntheticDuckDBLayer())
		_, err := pv.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
	})
}

// =============================================================================
// PluginViz.queryViz (with real DuckDB)
// =============================================================================

func TestQueryViz(t *testing.T) {
	t.Run("happy: queries real DuckDB for graph viz", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pv := NewPluginViz(dbl)

		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "graph"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.False(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "nodes")
	})

	t.Run("happy: queries real DuckDB for heatmap viz", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pv := NewPluginViz(dbl)

		result, err := pv.Execute(context.Background(), map[string]any{
			"viz_type": "heatmap",
			"scope":    "all",
		})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "heatmap", r["viz_type"])
		assert.Contains(t, r, "matrix")
	})

	t.Run("edge: defaults to graph when viz_type empty with real DB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pv := NewPluginViz(dbl)

		result, err := pv.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "graph", r["viz_type"])
	})
}

// =============================================================================
// PluginViz.syntheticViz
// =============================================================================

func TestSyntheticViz(t *testing.T) {
	pv := NewPluginViz(SyntheticDuckDBLayer())

	t.Run("happy: synthetic graph has 5 nodes", func(t *testing.T) {
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "graph"})
		require.NoError(t, err)
		r := result.(map[string]any)
		require.True(t, r["is_synthetic"].(bool))
		nodes := r["nodes"].([]map[string]any)
		assert.Len(t, nodes, 5)
	})

	t.Run("happy: synthetic heatmap has matrix with columns", func(t *testing.T) {
		result, err := pv.Execute(context.Background(), map[string]any{"viz_type": "heatmap"})
		require.NoError(t, err)
		r := result.(map[string]any)
		matrix := r["matrix"].(map[string]any)
		cols, ok := matrix["columns"].([]string)
		require.True(t, ok)
		assert.Len(t, cols, 3)
	})

	t.Run("edge: edge count equals nodes-1 for graph", func(t *testing.T) {
		result, _ := pv.Execute(context.Background(), map[string]any{"viz_type": "graph"})
		r := result.(map[string]any)
		nodes := r["nodes"].([]map[string]any)
		edges := r["edges"].([]map[string]any)
		assert.Len(t, edges, len(nodes)-1)
	})
}

// =============================================================================
// buildVizOutput
// =============================================================================

func TestBuildVizOutput(t *testing.T) {
	nodes := []map[string]any{
		{"id": "n1", "label": "A"},
		{"id": "n2", "label": "B"},
	}

	t.Run("happy: builds graph output with edges", func(t *testing.T) {
		out := buildVizOutput("graph", "all", nodes, true)
		assert.Equal(t, "graph", out["viz_type"])
		assert.Equal(t, "all", out["scope"])
		assert.True(t, out["is_synthetic"].(bool))
		assert.Contains(t, out, "nodes")
		assert.Contains(t, out, "edges")
	})

	t.Run("happy: builds heatmap output with matrix", func(t *testing.T) {
		out := buildVizOutput("heatmap", "narrow", nodes, false)
		assert.Equal(t, "heatmap", out["viz_type"])
		matrix := out["matrix"].(map[string]any)
		assert.NotNil(t, matrix)
		assert.Equal(t, "2x3 matrix", matrix["values"])
	})

	t.Run("happy: builds timeline output with events", func(t *testing.T) {
		out := buildVizOutput("timeline", "scope", nodes, true)
		assert.Equal(t, "timeline", out["viz_type"])
		events := out["events"].([]map[string]any)
		assert.Len(t, events, 2)
	})

	t.Run("edge: unknown viz_type defaults to same as graph", func(t *testing.T) {
		out := buildVizOutput("unknown_viz", "default", nodes, false)
		assert.Equal(t, "unknown_viz", out["viz_type"])
		assert.Contains(t, out, "nodes")
		assert.Contains(t, out, "edges")
	})
}

// =============================================================================
// buildSyntheticEdges
// =============================================================================

func TestBuildSyntheticEdges(t *testing.T) {
	t.Run("happy: creates edge for each adjacent pair", func(t *testing.T) {
		nodes := []map[string]any{
			{"id": "a"},
			{"id": "b"},
			{"id": "c"},
			{"id": "d"},
		}
		edges := buildSyntheticEdges(nodes, true)
		assert.Len(t, edges, 3)
		assert.Equal(t, "a", edges[0]["from"])
		assert.Equal(t, "b", edges[0]["to"])
		assert.True(t, edges[0]["is_synthetic"].(bool))
	})

	t.Run("happy: returns empty slice for single node", func(t *testing.T) {
		nodes := []map[string]any{{"id": "only"}}
		edges := buildSyntheticEdges(nodes, false)
		assert.Empty(t, edges)
	})

	t.Run("edge: returns empty slice for empty nodes", func(t *testing.T) {
		edges := buildSyntheticEdges([]map[string]any{}, true)
		assert.Empty(t, edges)
	})

	t.Run("edge: is_synthetic flag propagates correctly", func(t *testing.T) {
		nodes := []map[string]any{{"id": "x"}, {"id": "y"}}
		edges := buildSyntheticEdges(nodes, false)
		assert.False(t, edges[0]["is_synthetic"].(bool))
	})
}
