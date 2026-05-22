package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GeographicContext.Name
// =============================================================================

func TestGeographicContextName(t *testing.T) {
	gc := NewGeographicContext(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "he_geographic_context", gc.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, gc.Name(), gc.Name())
	})

	t.Run("edge: non-empty snake_case", func(t *testing.T) {
		assert.NotEmpty(t, gc.Name())
		assert.Contains(t, gc.Name(), "_")
	})
}

// =============================================================================
// GeographicContext.Description
// =============================================================================

func TestGeographicContextDescription(t *testing.T) {
	gc := NewGeographicContext(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, gc.Description())
	})

	t.Run("happy: contains key terms", func(t *testing.T) {
		desc := gc.Description()
		assert.Contains(t, desc, "geographic")
		assert.Contains(t, desc, "privacy-preserving")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, gc.Description(), gc.Description())
	})
}

// =============================================================================
// NewGeographicContext
// =============================================================================

func TestNewGeographicContext(t *testing.T) {
	t.Run("happy: creates with synthetic layer", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		require.NotNil(t, gc)
		assert.False(t, gc.db.IsAvailable())
	})

	t.Run("happy: creates with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		gc := NewGeographicContext(dbl)
		require.NotNil(t, gc)
		assert.True(t, gc.db.IsAvailable())
	})

	t.Run("edge: nil DuckDB layer produces working tool", func(t *testing.T) {
		gc := NewGeographicContext(nil)
		require.NotNil(t, gc)
		assert.NotEmpty(t, gc.Name())
	})
}

// =============================================================================
// GeographicContext.Execute
// =============================================================================

func TestGeographicContextExecute(t *testing.T) {
	t.Run("happy: returns synthetic data for specific region", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		result, err := gc.Execute(context.Background(), map[string]any{"region": "patagonia"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "patagonia", r["region"])
		assert.True(t, r["is_synthetic"].(bool))
	})

	t.Run("happy: returns coordinates in valid range", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		result, err := gc.Execute(context.Background(), map[string]any{"region": "test"})
		require.NoError(t, err)
		r := result.(map[string]any)
		coords := r["coordinates"].(map[string]any)
		lat := coords["latitude"].(float64)
		lon := coords["longitude"].(float64)
		assert.GreaterOrEqual(t, lat, -90.0)
		assert.LessOrEqual(t, lat, 90.0)
		assert.GreaterOrEqual(t, lon, -180.0)
		assert.LessOrEqual(t, lon, 180.0)
	})

	t.Run("edge: defaults region when empty", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		result, err := gc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default", r["region"])
	})

	t.Run("edge: same region yields consistent coordinates (deterministic)", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		r1, _ := gc.Execute(context.Background(), map[string]any{"region": "fixed_region"})
		r2, _ := gc.Execute(context.Background(), map[string]any{"region": "fixed_region"})
		coords1 := r1.(map[string]any)["coordinates"].(map[string]any)
		coords2 := r2.(map[string]any)["coordinates"].(map[string]any)
		assert.Equal(t, coords1["latitude"], coords2["latitude"])
		assert.Equal(t, coords1["longitude"], coords2["longitude"])
	})

	t.Run("edge: different regions produce different coordinates", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		r1, _ := gc.Execute(context.Background(), map[string]any{"region": "region_a"})
		r2, _ := gc.Execute(context.Background(), map[string]any{"region": "region_b"})
		coords1 := r1.(map[string]any)["coordinates"].(map[string]any)
		coords2 := r2.(map[string]any)["coordinates"].(map[string]any)
		assert.NotEqual(t, coords1["latitude"], coords2["latitude"])
	})

	t.Run("edge: clusters present and non-empty", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		result, err := gc.Execute(context.Background(), map[string]any{"region": "any"})
		require.NoError(t, err)
		r := result.(map[string]any)
		clusters, ok := r["clusters"].([]map[string]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(clusters), 2)
		assert.LessOrEqual(t, len(clusters), 5)
	})

	t.Run("error: Execute never errors with synthetic layer", func(t *testing.T) {
		gc := NewGeographicContext(SyntheticDuckDBLayer())
		_, err := gc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
	})
}

// =============================================================================
// GeographicContext.queryGeographic (with real DuckDB)
// =============================================================================

func TestQueryGeographic(t *testing.T) {
	t.Run("happy: queries real DuckDB successfully", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		gc := NewGeographicContext(dbl)

		result, err := gc.Execute(context.Background(), map[string]any{"region": "test_region"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "test_region", r["region"])
		assert.False(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "tool_density")
	})

	t.Run("edge: defaults region when empty with real DB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		gc := NewGeographicContext(dbl)

		result, err := gc.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "default", r["region"])
	})
}

// =============================================================================
// GeographicContext.syntheticGeographic
// =============================================================================

func TestSyntheticGeographic(t *testing.T) {
	gc := NewGeographicContext(SyntheticDuckDBLayer())

	t.Run("happy: returns full synthetic response", func(t *testing.T) {
		result, err := gc.Execute(context.Background(), map[string]any{"region": "sahara"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "sahara", r["region"])
		assert.True(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "clusters")
		assert.Contains(t, r, "coordinates")
		assert.Contains(t, r, "generated_at")
	})

	t.Run("edge: each cluster has required fields", func(t *testing.T) {
		result, err := gc.Execute(context.Background(), map[string]any{"region": "validate"})
		require.NoError(t, err)
		r := result.(map[string]any)
		clusters := r["clusters"].([]map[string]any)
		for _, cluster := range clusters {
			assert.Contains(t, cluster, "cluster_id")
			assert.Contains(t, cluster, "latitude")
			assert.Contains(t, cluster, "longitude")
			assert.Contains(t, cluster, "density")
			assert.Contains(t, cluster, "label")
		}
	})

	t.Run("edge: cluster labels are valid categories", func(t *testing.T) {
		result, err := gc.Execute(context.Background(), map[string]any{"region": "labels"})
		require.NoError(t, err)
		r := result.(map[string]any)
		clusters := r["clusters"].([]map[string]any)
		validLabels := map[string]bool{"high": true, "medium": true, "low": true}
		for _, cluster := range clusters {
			label := cluster["label"].(string)
			assert.True(t, validLabels[label], "unexpected label: %s", label)
		}
	})
}

// =============================================================================
// roundFloat
// =============================================================================

func TestRoundFloat(t *testing.T) {
	t.Run("happy: rounds to 4 decimal places", func(t *testing.T) {
		assert.Equal(t, 1.2346, roundFloat(1.23456, 4))
	})

	t.Run("happy: rounds to 2 decimal places", func(t *testing.T) {
		assert.Equal(t, 3.14, roundFloat(3.14159, 2))
	})

	t.Run("happy: rounds to 0 decimal places", func(t *testing.T) {
		assert.Equal(t, 5.0, roundFloat(5.49, 0))
		assert.Equal(t, 6.0, roundFloat(5.5, 0))
	})

	t.Run("edge: zero decimals", func(t *testing.T) {
		assert.Equal(t, 100.0, roundFloat(99.99, 0))
	})

	t.Run("edge: negative number", func(t *testing.T) {
		assert.Equal(t, -1.2345, roundFloat(-1.23456, 4))
	})

	t.Run("edge: zero value", func(t *testing.T) {
		assert.Equal(t, 0.0, roundFloat(0.0, 4))
	})
}

// =============================================================================
// itoa
// =============================================================================

func TestItoa(t *testing.T) {
	t.Run("happy: converts positive integer", func(t *testing.T) {
		assert.Equal(t, "42", itoa(42))
	})

	t.Run("happy: converts zero", func(t *testing.T) {
		assert.Equal(t, "0", itoa(0))
	})

	t.Run("happy: converts large integer", func(t *testing.T) {
		assert.Equal(t, "1234567", itoa(1234567))
	})

	t.Run("edge: small single digit", func(t *testing.T) {
		assert.Equal(t, "7", itoa(7))
	})

	t.Run("edge: maximum reasonable value", func(t *testing.T) {
		result := itoa(999999)
		assert.Equal(t, "999999", result)
	})
}
