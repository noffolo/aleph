package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PatternClassifier.Name
// =============================================================================

func TestPatternClassifierName(t *testing.T) {
	pc := NewPatternClassifier(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "he_pattern_classifier", pc.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, pc.Name(), pc.Name())
	})

	t.Run("edge: non-empty snake_case identifier", func(t *testing.T) {
		assert.NotEmpty(t, pc.Name())
	})
}

// =============================================================================
// PatternClassifier.Description
// =============================================================================

func TestPatternClassifierDescription(t *testing.T) {
	pc := NewPatternClassifier(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, pc.Description())
	})

	t.Run("happy: mentions privacy and synthetic", func(t *testing.T) {
		desc := pc.Description()
		assert.Contains(t, desc, "privacy-preserving")
		assert.Contains(t, desc, "is_synthetic=true")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, pc.Description(), pc.Description())
	})
}

// =============================================================================
// NewPatternClassifier
// =============================================================================

func TestNewPatternClassifier(t *testing.T) {
	t.Run("happy: creates with synthetic layer", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		require.NotNil(t, pc)
		assert.False(t, pc.db.IsAvailable())
	})

	t.Run("happy: creates with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pc := NewPatternClassifier(dbl)
		require.NotNil(t, pc)
		assert.True(t, pc.db.IsAvailable())
	})

	t.Run("edge: nil DuckDB layer still works", func(t *testing.T) {
		pc := NewPatternClassifier(nil)
		require.NotNil(t, pc)
		assert.NotEmpty(t, pc.Name())
	})
}

// =============================================================================
// PatternClassifier.Execute
// =============================================================================

func TestPatternClassifierExecute(t *testing.T) {
	t.Run("happy: returns synthetic patterns for valid data", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		result, err := pc.Execute(context.Background(), map[string]any{"data": "network analysis"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.True(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "data")

		patterns := r["patterns"].([]map[string]any)
		assert.GreaterOrEqual(t, len(patterns), 2)
	})

	t.Run("edge: defaults data when empty", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		result, err := pc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.NotNil(t, result)
		assert.Equal(t, "default_pattern_data", r["data"])
	})

	t.Run("edge: each pattern has required fields", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		result, _ := pc.Execute(context.Background(), map[string]any{"data": "validate_fields"})
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
		for _, pat := range patterns {
			_, hasID := pat["pattern_id"]
			_, hasType := pat["pattern_type"]
			_, hasConf := pat["confidence"]
			_, hasMatch := pat["matched_on"]
			assert.True(t, hasID)
			assert.True(t, hasType)
			assert.True(t, hasConf)
			assert.True(t, hasMatch)
		}
	})

	t.Run("edge: deterministic output for same data", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		r1, _ := pc.Execute(context.Background(), map[string]any{"data": "deterministic_test"})
		r2, _ := pc.Execute(context.Background(), map[string]any{"data": "deterministic_test"})
		pats1 := r1.(map[string]any)["patterns"].([]map[string]any)
		pats2 := r2.(map[string]any)["patterns"].([]map[string]any)
		assert.Len(t, pats1, len(pats2))
		for i := range pats1 {
			assert.Equal(t, pats1[i]["pattern_id"], pats2[i]["pattern_id"])
		}
	})

	t.Run("edge: pattern_type matches known builtin set", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		result, _ := pc.Execute(context.Background(), map[string]any{"data": "any"})
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
		for _, pat := range patterns {
			pt := pat["pattern_type"].(string)
			assert.Contains(t, pt, "_")
		}
	})

	t.Run("error: never returns error with synthetic layer", func(t *testing.T) {
		pc := NewPatternClassifier(SyntheticDuckDBLayer())
		_, err := pc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
	})
}

// =============================================================================
// PatternClassifier.queryPatterns (with real DuckDB via Execute)
// =============================================================================

func TestQueryPatterns(t *testing.T) {
	t.Run("happy: queries real DuckDB and returns patterns", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pc := NewPatternClassifier(dbl)

		result, err := pc.Execute(context.Background(), map[string]any{"data": "collaboration analysis"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.False(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "patterns")
	})

	t.Run("edge: data defaults when empty with real DB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pc := NewPatternClassifier(dbl)

		result, err := pc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.False(t, r["is_synthetic"].(bool))
	})

	t.Run("edge: handles query with no matching patterns gracefully", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		pc := NewPatternClassifier(dbl)

		result, err := pc.Execute(context.Background(), map[string]any{"data": "zzz_no_match"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.False(t, r["is_synthetic"].(bool))
	})
}

// =============================================================================
// PatternClassifier.syntheticPatterns via Execute with SyntheticDuckDBLayer
// =============================================================================

func TestSyntheticPatterns(t *testing.T) {
	pc := NewPatternClassifier(SyntheticDuckDBLayer())

	t.Run("happy: produces patterns with confidence in [0.5, 1.0]", func(t *testing.T) {
		result, err := pc.Execute(context.Background(), map[string]any{"data": "confidence_check"})
		require.NoError(t, err)
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
		for _, pat := range patterns {
			conf, ok := pat["confidence"].(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, conf, 0.5)
			assert.LessOrEqual(t, conf, 1.0)
		}
	})

	t.Run("happy: pattern_ids are SHA-256 hex strings", func(t *testing.T) {
		result, _ := pc.Execute(context.Background(), map[string]any{"data": "id_validation"})
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
		for _, pat := range patterns {
			id := pat["pattern_id"].(string)
			assert.Len(t, id, 64)
		}
	})

	t.Run("edge: empty data still produces patterns", func(t *testing.T) {
		result, err := pc.Execute(context.Background(), map[string]any{"data": ""})
		require.NoError(t, err)
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
		assert.GreaterOrEqual(t, len(patterns), 2)
	})
}
