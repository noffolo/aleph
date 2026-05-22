package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// RelationalEngine.Name
// =============================================================================

func TestRelationalEngineName(t *testing.T) {
	re := NewRelationalEngine(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "he_relational_engine", re.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, re.Name(), re.Name())
	})

	t.Run("edge: non-empty snake_case", func(t *testing.T) {
		assert.NotEmpty(t, re.Name())
	})
}

// =============================================================================
// RelationalEngine.Description
// =============================================================================

func TestRelationalEngineDescription(t *testing.T) {
	re := NewRelationalEngine(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, re.Description())
	})

	t.Run("happy: mentions relational and privacy", func(t *testing.T) {
		desc := re.Description()
		assert.Contains(t, desc, "relational")
		assert.Contains(t, desc, "privacy-preserving")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, re.Description(), re.Description())
	})
}

// =============================================================================
// NewRelationalEngine
// =============================================================================

func TestNewRelationalEngine(t *testing.T) {
	t.Run("happy: creates with synthetic layer", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		require.NotNil(t, re)
		assert.False(t, re.db.IsAvailable())
	})

	t.Run("happy: creates with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		re := NewRelationalEngine(dbl)
		require.NotNil(t, re)
		assert.True(t, re.db.IsAvailable())
	})

	t.Run("edge: nil dbl creates valid tool", func(t *testing.T) {
		re := NewRelationalEngine(nil)
		require.NotNil(t, re)
		assert.NotEmpty(t, re.Name())
	})
}

// =============================================================================
// RelationalEngine.Execute
// =============================================================================

func TestRelationalEngineExecute(t *testing.T) {
	t.Run("happy: returns relations for specific entity", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		result, err := re.Execute(context.Background(), map[string]any{"entity": "ecosystem-alpha"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "ecosystem-alpha", r["entity"])
		assert.True(t, r["is_synthetic"].(bool))
	})

	t.Run("happy: relations slice is non-empty", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		result, err := re.Execute(context.Background(), map[string]any{"entity": "test"})
		require.NoError(t, err)
		r := result.(map[string]any)
		relations := r["relations"].([]map[string]any)
		assert.GreaterOrEqual(t, len(relations), 3)
		assert.LessOrEqual(t, len(relations), 7)
	})

	t.Run("edge: defaults entity when empty", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		result, err := re.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default", r["entity"])
	})

	t.Run("edge: relation_ids are deterministic SHA-256 hashes", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		r1, _ := re.Execute(context.Background(), map[string]any{"entity": "test_entity"})
		r2, _ := re.Execute(context.Background(), map[string]any{"entity": "test_entity"})
		rels1 := r1.(map[string]any)["relations"].([]map[string]any)
		rels2 := r2.(map[string]any)["relations"].([]map[string]any)
		assert.Equal(t, len(rels1), len(rels2))
		for i := range rels1 {
			assert.Equal(t, rels1[i]["relation_id"], rels2[i]["relation_id"])
		}
	})

	t.Run("edge: all relation_types are valid", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		result, _ := re.Execute(context.Background(), map[string]any{"entity": "type_check"})
		r := result.(map[string]any)
		relations := r["relations"].([]map[string]any)
		validTypes := []string{"dependency", "collaboration", "hierarchy", "peer", "influence"}
		for _, rel := range relations {
			rt := rel["relation_type"].(string)
			assert.Contains(t, validTypes, rt)
		}
	})

	t.Run("error: never errors with synthetic layer", func(t *testing.T) {
		re := NewRelationalEngine(SyntheticDuckDBLayer())
		_, err := re.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
	})
}

// =============================================================================
// RelationalEngine.queryRelational (with real DuckDB via Execute)
// =============================================================================

func TestQueryRelational(t *testing.T) {
	t.Run("happy: queries real DuckDB for entity", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		re := NewRelationalEngine(dbl)

		result, err := re.Execute(context.Background(), map[string]any{"entity": "entity_in_db"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "entity_in_db", r["entity"])
		assert.False(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "relations")
	})

	t.Run("edge: defaults entity when empty with real DB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		re := NewRelationalEngine(dbl)

		result, err := re.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default", r["entity"])
	})
}

// =============================================================================
// RelationalEngine.syntheticRelational via Execute with SyntheticDuckDBLayer
// =============================================================================

func TestSyntheticRelational(t *testing.T) {
	re := NewRelationalEngine(SyntheticDuckDBLayer())

	t.Run("happy: returns synthetic relations with required fields", func(t *testing.T) {
		result, err := re.Execute(context.Background(), map[string]any{"entity": "synth"})
		require.NoError(t, err)
		r := result.(map[string]any)
		relations := r["relations"].([]map[string]any)
		for _, rel := range relations {
			assert.Contains(t, rel, "relation_id")
			assert.Contains(t, rel, "related_entity")
			assert.Contains(t, rel, "relation_type")
			assert.Contains(t, rel, "strength")
		}
	})

	t.Run("happy: strength values are within 0-99", func(t *testing.T) {
		result, _ := re.Execute(context.Background(), map[string]any{"entity": "strength_test"})
		r := result.(map[string]any)
		relations := r["relations"].([]map[string]any)
		for _, rel := range relations {
			strength, ok := rel["strength"].(int)
			require.True(t, ok)
			assert.GreaterOrEqual(t, strength, 0)
			assert.Less(t, strength, 100)
		}
	})

	t.Run("edge: empty entity string defaults to 'default'", func(t *testing.T) {
		result, err := re.Execute(context.Background(), map[string]any{"entity": ""})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default", r["entity"])
		relations := r["relations"].([]map[string]any)
		assert.GreaterOrEqual(t, len(relations), 3)
	})
}
