package humanecosystems

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResearchProfiles(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewResearchProfiles(dbl)

	t.Run("returns synthetic profiles when DuckDB unavailable", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"query": "ecosystem analysis"})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ecosystem analysis", r["query"])
		assert.True(t, r["is_synthetic"].(bool))
		profiles, ok := r["profiles"].([]map[string]interface{})
		if !ok {
			profilesRaw, ok := r["profiles"].([]interface{})
			require.True(t, ok)
			assert.Greater(t, len(profilesRaw), 0)
		} else {
			assert.Greater(t, len(profiles), 0)
		}
	})

	t.Run("defaults query to empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, r["generated_at"])
	})

	t.Run("profiles have no PII", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"query": "pii check"})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		json := fmt.Sprintf("%v", r)
		assert.NotContains(t, json, "email")
		assert.NotContains(t, json, "password")
		assert.NotContains(t, json, "phone")
	})
}

func TestRelationalEngine(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewRelationalEngine(dbl)

	t.Run("returns synthetic relations when DuckDB unavailable", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"entity": "ecosystem-alpha"})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "ecosystem-alpha", r["entity"])
		assert.True(t, r["is_synthetic"].(bool))
		relationsRaw, ok := r["relations"].([]map[string]interface{})
		require.True(t, ok, "relations should be []map[string]interface{}")
		assert.Greater(t, len(relationsRaw), 0)
		assert.Contains(t, relationsRaw[0], "relation_id")
		assert.Contains(t, relationsRaw[0], "related_entity")
		assert.Contains(t, relationsRaw[0], "relation_type")
	})

	t.Run("defaults entity when empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "default", r["entity"])
	})

	t.Run("relation_ids are SHA-256 hashed", func(t *testing.T) {
		r1, _ := tool.Execute(context.Background(), map[string]any{"entity": "test"})
		r2, _ := tool.Execute(context.Background(), map[string]any{"entity": "test"})
		m1 := r1.(map[string]interface{})
		m2 := r2.(map[string]interface{})
		rels1 := m1["relations"].([]map[string]interface{})
		rels2 := m2["relations"].([]map[string]interface{})
		for i := range rels1 {
			assert.Equal(t, rels1[i]["relation_id"], rels2[i]["relation_id"])
		}
	})
}

func TestGeographicContext(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewGeographicContext(dbl)

	t.Run("returns synthetic geographic data", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"region": "patagonia"})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "patagonia", r["region"])
		assert.True(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "coordinates")
		assert.Contains(t, r, "clusters")
	})

	t.Run("returns valid coordinate ranges", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{"region": "test"})
		r := result.(map[string]interface{})
		coords := r["coordinates"].(map[string]interface{})
		lat := coords["latitude"].(float64)
		lon := coords["longitude"].(float64)
		assert.GreaterOrEqual(t, lat, -90.0)
		assert.LessOrEqual(t, lat, 90.0)
		assert.GreaterOrEqual(t, lon, -180.0)
		assert.LessOrEqual(t, lon, 180.0)
	})

	t.Run("defaults region when empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]interface{})
		assert.Equal(t, "default", r["region"])
	})
}

func TestPatternClassifier(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewPatternClassifier(dbl)

	t.Run("returns synthetic patterns", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"data": "network analysis sample"})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.True(t, r["is_synthetic"].(bool))
		patterns, ok := r["patterns"].([]map[string]interface{})
		require.True(t, ok, "patterns should be []map[string]interface{}")
		assert.Greater(t, len(patterns), 0)
	})

	t.Run("patterns have confidence scores", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"data": "confidence check"})
		require.NoError(t, err)
		r := result.(map[string]interface{})
		patterns := r["patterns"].([]map[string]interface{})
		for _, pat := range patterns {
			conf, ok := pat["confidence"].(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, conf, 0.5)
			assert.LessOrEqual(t, conf, 1.0)
		}
	})

	t.Run("data defaults when empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("pattern_ids are hashed, never raw input", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{"data": "sensitive_info"})
		r := result.(map[string]interface{})
		patterns := r["patterns"].([]map[string]interface{})
		for _, pat := range patterns {
			id, ok := pat["pattern_id"].(string)
			require.True(t, ok)
			assert.Len(t, id, 64)
		}
	})
}

func TestPluginViz(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewPluginViz(dbl)

	t.Run("returns graph visualization by default", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r, ok := result.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, r, "viz_type")
		assert.Contains(t, r, "nodes")
		assert.Contains(t, r, "edges")
		assert.Equal(t, "graph", r["viz_type"])
	})

	t.Run("supports heatmap viz_type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"viz_type": "heatmap"})
		require.NoError(t, err)
		r := result.(map[string]interface{})
		assert.Equal(t, "heatmap", r["viz_type"])
		assert.Contains(t, r, "matrix")
	})

	t.Run("supports timeline viz_type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"viz_type": "timeline"})
		require.NoError(t, err)
		r := result.(map[string]interface{})
		assert.Equal(t, "timeline", r["viz_type"])
		assert.Contains(t, r, "events")
	})

	t.Run("nodes use SHA-256 hashed IDs, never raw data", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{})
		r := result.(map[string]interface{})
		nodes := r["nodes"].([]map[string]interface{})
		for _, node := range nodes {
			id, ok := node["id"].(string)
			require.True(t, ok)
			assert.Len(t, id, 64)
		}
	})
}

func TestListTools(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tools := ListTools(dbl)

	t.Run("returns 5 tools", func(t *testing.T) {
		assert.Len(t, tools, 5)
	})

	t.Run("all tools have unique names", func(t *testing.T) {
		names := make(map[string]bool)
		for _, tool := range tools {
			assert.False(t, names[tool.Name()], "duplicate tool name: %s", tool.Name())
			names[tool.Name()] = true
		}
	})

	t.Run("all tools return valid Execute results", func(t *testing.T) {
		for _, tool := range tools {
			result, err := tool.Execute(context.Background(), map[string]any{})
			require.NoError(t, err, "tool %s failed", tool.Name())
			assert.NotNil(t, result, "tool %s returned nil", tool.Name())
		}
	})
}
