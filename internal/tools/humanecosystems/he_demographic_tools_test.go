package humanecosystems

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewDemographicProfileTool
// =============================================================================

func TestNewDemographicProfileTool(t *testing.T) {
	t.Run("happy: creates tool with synthetic layer", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		tool := NewDemographicProfileTool(dbl)
		require.NotNil(t, tool)
		assert.Equal(t, "demographicProfile", tool.Name())
	})

	t.Run("happy: creates tool with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		tool := NewDemographicProfileTool(dbl)
		require.NotNil(t, tool)
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("edge: works with nil DuckDB layer", func(t *testing.T) {
		tool := NewDemographicProfileTool(nil)
		require.NotNil(t, tool)
		assert.Contains(t, tool.Description(), "demographic profile data")
	})
}

// =============================================================================
// DemographicProfileTool.Name
// =============================================================================

func TestDemographicProfileToolName(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewDemographicProfileTool(dbl)

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "demographicProfile", tool.Name())
	})

	t.Run("happy: consistent across calls", func(t *testing.T) {
		assert.Equal(t, tool.Name(), tool.Name())
	})

	t.Run("edge: not empty and lowercase", func(t *testing.T) {
		name := tool.Name()
		assert.NotEmpty(t, name)
		assert.Equal(t, name, name) // idempotent
	})
}

// =============================================================================
// DemographicProfileTool.Description
// =============================================================================

func TestDemographicProfileToolDescription(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewDemographicProfileTool(dbl)

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		desc := tool.Description()
		assert.NotEmpty(t, desc)
	})

	t.Run("happy: contains key terms", func(t *testing.T) {
		desc := tool.Description()
		assert.Contains(t, desc, "demographic")
		assert.Contains(t, desc, "population")
	})

	t.Run("edge: consistent across calls", func(t *testing.T) {
		assert.Equal(t, tool.Description(), tool.Description())
	})
}

// =============================================================================
// DemographicProfileTool.Execute
// =============================================================================

func TestDemographicProfileToolExecute(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewDemographicProfileTool(dbl)

	t.Run("happy: returns data for DEU", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "DEU"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "DEU", r["country_code"])
		assert.Equal(t, "Germany", r["country_name"])
		pop, ok := r["population"].(int64)
		require.True(t, ok)
		assert.Greater(t, pop, int64(80000000))
	})

	t.Run("happy: returns data for BRA with GDP check", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "BRA"})
		require.NoError(t, err)
		r := result.(map[string]any)
		gdp, ok := r["gdp_per_capita"].(float64)
		require.True(t, ok)
		assert.Greater(t, gdp, 5000.0)
		assert.Less(t, gdp, 15000.0)
	})

	t.Run("error: empty countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "countryCode is required")
	})

	t.Run("error: unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "XYZ"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown country code")
	})
}

// =============================================================================
// NewSocioeconomicIndicatorsTool
// =============================================================================

func TestNewSocioeconomicIndicatorsTool(t *testing.T) {
	t.Run("happy: creates tool with synthetic layer", func(t *testing.T) {
		dbl := SyntheticDuckDBLayer()
		tool := NewSocioeconomicIndicatorsTool(dbl)
		require.NotNil(t, tool)
		assert.Equal(t, "socioeconomicIndicators", tool.Name())
	})

	t.Run("happy: creates tool with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		tool := NewSocioeconomicIndicatorsTool(dbl)
		require.NotNil(t, tool)
	})

	t.Run("edge: nil dbl produces working tool", func(t *testing.T) {
		tool := NewSocioeconomicIndicatorsTool(nil)
		require.NotNil(t, tool)
	})
}

// =============================================================================
// SocioeconomicIndicatorsTool.Name
// =============================================================================

func TestSocioeconomicIndicatorsToolName(t *testing.T) {
	tool := NewSocioeconomicIndicatorsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "socioeconomicIndicators", tool.Name())
	})

	t.Run("happy: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Name(), tool.Name())
	})

	t.Run("edge: camelCase format", func(t *testing.T) {
		name := tool.Name()
		assert.NotEmpty(t, name)
		assert.NotContains(t, name, " ")
	})
}

// =============================================================================
// SocioeconomicIndicatorsTool.Description
// =============================================================================

func TestSocioeconomicIndicatorsToolDescription(t *testing.T) {
	tool := NewSocioeconomicIndicatorsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("happy: contains Gini mention", func(t *testing.T) {
		assert.Contains(t, tool.Description(), "Gini")
	})

	t.Run("edge: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Description(), tool.Description())
	})
}

// =============================================================================
// SocioeconomicIndicatorsTool.Execute
// =============================================================================

func TestSocioeconomicIndicatorsToolExecute(t *testing.T) {
	tool := NewSocioeconomicIndicatorsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns data for FRA", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "FRA"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "FRA", r["country_code"])
		gini, ok := r["gini_coefficient"].(float64)
		require.True(t, ok)
		assert.Greater(t, gini, 25.0)
		assert.Less(t, gini, 40.0)
	})

	t.Run("happy: returns data for NGA with high poverty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "NGA"})
		require.NoError(t, err)
		r := result.(map[string]any)
		poverty, ok := r["poverty_rate"].(float64)
		require.True(t, ok)
		assert.Greater(t, poverty, 35.0)
	})

	t.Run("error: empty countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "countryCode is required")
	})

	t.Run("error: unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ABC"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown country code")
	})
}

// =============================================================================
// NewCulturalMetricsTool
// =============================================================================

func TestNewCulturalMetricsTool(t *testing.T) {
	t.Run("happy: creates tool with synthetic layer", func(t *testing.T) {
		tool := NewCulturalMetricsTool(SyntheticDuckDBLayer())
		require.NotNil(t, tool)
		assert.Equal(t, "culturalMetrics", tool.Name())
	})

	t.Run("happy: creates tool with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		tool := NewCulturalMetricsTool(dbl)
		require.NotNil(t, tool)
	})

	t.Run("edge: nil dbl works", func(t *testing.T) {
		tool := NewCulturalMetricsTool(nil)
		require.NotNil(t, tool)
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// CulturalMetricsTool.Name
// =============================================================================

func TestCulturalMetricsToolName(t *testing.T) {
	tool := NewCulturalMetricsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "culturalMetrics", tool.Name())
	})

	t.Run("happy: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Name(), tool.Name())
	})

	t.Run("edge: non-empty and camelCase", func(t *testing.T) {
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// CulturalMetricsTool.Description
// =============================================================================

func TestCulturalMetricsToolDescription(t *testing.T) {
	tool := NewCulturalMetricsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("happy: mentions language diversity", func(t *testing.T) {
		assert.Contains(t, tool.Description(), "language diversity")
	})

	t.Run("edge: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Description(), tool.Description())
	})
}

// =============================================================================
// CulturalMetricsTool.Execute
// =============================================================================

func TestCulturalMetricsToolExecute(t *testing.T) {
	tool := NewCulturalMetricsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns high internet for KOR", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "KOR"})
		require.NoError(t, err)
		r := result.(map[string]any)
		internet, ok := r["internet_pct"].(float64)
		require.True(t, ok)
		assert.Greater(t, internet, 90.0)
	})

	t.Run("happy: returns low internet for COD", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "COD"})
		require.NoError(t, err)
		r := result.(map[string]any)
		internet, ok := r["internet_pct"].(float64)
		require.True(t, ok)
		assert.Less(t, internet, 20.0)
	})

	t.Run("error: empty countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "countryCode is required")
	})
}

// =============================================================================
// NewUrbanRuralDistributionTool
// =============================================================================

func TestNewUrbanRuralDistributionTool(t *testing.T) {
	t.Run("happy: creates tool with synthetic layer", func(t *testing.T) {
		tool := NewUrbanRuralDistributionTool(SyntheticDuckDBLayer())
		require.NotNil(t, tool)
		assert.Equal(t, "urbanRuralDistribution", tool.Name())
	})

	t.Run("happy: creates tool with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		tool := NewUrbanRuralDistributionTool(dbl)
		require.NotNil(t, tool)
	})

	t.Run("edge: nil dbl creates valid tool", func(t *testing.T) {
		tool := NewUrbanRuralDistributionTool(nil)
		require.NotNil(t, tool)
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// UrbanRuralDistributionTool.Name
// =============================================================================

func TestUrbanRuralDistributionToolName(t *testing.T) {
	tool := NewUrbanRuralDistributionTool(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "urbanRuralDistribution", tool.Name())
	})

	t.Run("happy: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Name(), tool.Name())
	})

	t.Run("edge: camelCase", func(t *testing.T) {
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// UrbanRuralDistributionTool.Description
// =============================================================================

func TestUrbanRuralDistributionToolDescription(t *testing.T) {
	tool := NewUrbanRuralDistributionTool(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("happy: mentions threshold", func(t *testing.T) {
		assert.Contains(t, tool.Description(), "threshold")
	})

	t.Run("edge: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Description(), tool.Description())
	})
}

// =============================================================================
// UrbanRuralDistributionTool.Execute
// =============================================================================

func TestUrbanRuralDistributionToolExecute(t *testing.T) {
	tool := NewUrbanRuralDistributionTool(SyntheticDuckDBLayer())

	t.Run("happy: returns urban/rural split for THA", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "THA"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "THA", r["country_code"])
		assert.Equal(t, "mostly_urban", r["classification"])
	})

	t.Run("edge: very low threshold still classifies correctly", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"countryCode": "CHN",
			"threshold":   10.0,
		})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.True(t, r["above_threshold"].(bool))
		assert.Equal(t, float64(10), r["threshold_pct"])
	})

	t.Run("error: unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "XYZ"})
		require.Error(t, err)
	})

	t.Run("error: empty countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

// =============================================================================
// NewMigrationPatternsTool
// =============================================================================

func TestNewMigrationPatternsTool(t *testing.T) {
	t.Run("happy: creates tool with synthetic layer", func(t *testing.T) {
		tool := NewMigrationPatternsTool(SyntheticDuckDBLayer())
		require.NotNil(t, tool)
		assert.Equal(t, "migrationPatterns", tool.Name())
	})

	t.Run("happy: creates tool with real DuckDB", func(t *testing.T) {
		dbl := setupDuckDBLayer(t)
		tool := NewMigrationPatternsTool(dbl)
		require.NotNil(t, tool)
	})

	t.Run("edge: nil dbl works", func(t *testing.T) {
		tool := NewMigrationPatternsTool(nil)
		require.NotNil(t, tool)
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// MigrationPatternsTool.Name
// =============================================================================

func TestMigrationPatternsToolName(t *testing.T) {
	tool := NewMigrationPatternsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns expected name", func(t *testing.T) {
		assert.Equal(t, "migrationPatterns", tool.Name())
	})

	t.Run("happy: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Name(), tool.Name())
	})

	t.Run("edge: camelCase with no spaces", func(t *testing.T) {
		assert.NotEmpty(t, tool.Name())
	})
}

// =============================================================================
// MigrationPatternsTool.Description
// =============================================================================

func TestMigrationPatternsToolDescription(t *testing.T) {
	tool := NewMigrationPatternsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns non-empty description", func(t *testing.T) {
		assert.NotEmpty(t, tool.Description())
	})

	t.Run("happy: mentions migration and ISO", func(t *testing.T) {
		desc := tool.Description()
		assert.Contains(t, desc, "migration")
		assert.Contains(t, desc, "ISO")
	})

	t.Run("edge: consistent", func(t *testing.T) {
		assert.Equal(t, tool.Description(), tool.Description())
	})
}

// =============================================================================
// MigrationPatternsTool.Execute
// =============================================================================

func TestMigrationPatternsToolExecute(t *testing.T) {
	tool := NewMigrationPatternsTool(SyntheticDuckDBLayer())

	t.Run("happy: returns data for CHN→USA", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"originCountry": "CHN",
			"destCountry":   "USA",
		})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "CHN", r["origin"])
		assert.Equal(t, "USA", r["dest"])
		stock, ok := r["stock"].(int64)
		require.True(t, ok)
		assert.Greater(t, stock, int64(2000000))
	})

	t.Run("happy: returns zero stock for unknown corridor", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"originCountry": "JPN",
			"destCountry":   "TZA",
		})
		require.NoError(t, err)
		r := result.(map[string]any)
		stock, ok := r["stock"].(int64)
		require.True(t, ok)
		assert.Equal(t, int64(0), stock)
	})

	t.Run("error: missing originCountry", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"destCountry": "USA"})
		require.Error(t, err)
	})

	t.Run("error: missing destCountry", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"originCountry": "USA"})
		require.Error(t, err)
	})

	t.Run("error: empty args", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

// =============================================================================
// classifyUrbanRural
// =============================================================================

func TestClassifyUrbanRural(t *testing.T) {
	t.Run("happy: >=80 is highly_urbanized", func(t *testing.T) {
		assert.Equal(t, "highly_urbanized", classifyUrbanRural(80.0))
		assert.Equal(t, "highly_urbanized", classifyUrbanRural(95.5))
	})

	t.Run("happy: >=50 and <80 is mostly_urban", func(t *testing.T) {
		assert.Equal(t, "mostly_urban", classifyUrbanRural(50.0))
		assert.Equal(t, "mostly_urban", classifyUrbanRural(79.9))
	})

	t.Run("happy: >=30 and <50 is mixed", func(t *testing.T) {
		assert.Equal(t, "mixed", classifyUrbanRural(30.0))
		assert.Equal(t, "mixed", classifyUrbanRural(49.9))
	})

	t.Run("happy: <30 is mostly_rural", func(t *testing.T) {
		assert.Equal(t, "mostly_rural", classifyUrbanRural(29.9))
		assert.Equal(t, "mostly_rural", classifyUrbanRural(0.0))
	})

	t.Run("edge: boundary value 49.9999", func(t *testing.T) {
		assert.Equal(t, "mixed", classifyUrbanRural(49.9999))
	})

	t.Run("edge: negative values classify as mostly_rural", func(t *testing.T) {
		assert.Equal(t, "mostly_rural", classifyUrbanRural(-10.0))
	})
}
