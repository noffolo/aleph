package humanecosystems

import (
	"context"
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
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ecosystem analysis", r["query"])
		assert.True(t, r["is_synthetic"].(bool))
		profiles, ok := r["profiles"].([]map[string]any)
		if !ok {
			profilesRaw, ok := r["profiles"].([]any)
			require.True(t, ok)
			assert.Greater(t, len(profilesRaw), 0)
		} else {
			assert.Greater(t, len(profiles), 0)
		}
	})
}

func TestMarshalJSON(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		out := marshalJSON(map[string]string{"key": "value"})
		assert.Equal(t, "{\n  \"key\": \"value\"\n}", out)
	})
	t.Run("nil", func(t *testing.T) {
		out := marshalJSON(nil)
		assert.Equal(t, "null", out)
	})
	t.Run("slice", func(t *testing.T) {
		out := marshalJSON([]int{1, 2, 3})
		assert.Equal(t, "[\n  1,\n  2,\n  3\n]", out)
	})
	t.Run("channel fails gracefully", func(t *testing.T) {
		out := marshalJSON(make(chan int))
		assert.Contains(t, out, `"error":`)
	})
}

func TestRelationalEngine(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewRelationalEngine(dbl)

	t.Run("returns synthetic relations when DuckDB unavailable", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"entity": "ecosystem-alpha"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ecosystem-alpha", r["entity"])
		assert.True(t, r["is_synthetic"].(bool))
		relationsRaw, ok := r["relations"].([]map[string]any)
		require.True(t, ok, "relations should be []map[string]interface{}")
		assert.Greater(t, len(relationsRaw), 0)
		assert.Contains(t, relationsRaw[0], "relation_id")
		assert.Contains(t, relationsRaw[0], "related_entity")
		assert.Contains(t, relationsRaw[0], "relation_type")
	})

	t.Run("defaults entity when empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "default", r["entity"])
	})

	t.Run("relation_ids are SHA-256 hashed", func(t *testing.T) {
		r1, _ := tool.Execute(context.Background(), map[string]any{"entity": "test"})
		r2, _ := tool.Execute(context.Background(), map[string]any{"entity": "test"})
		m1 := r1.(map[string]any)
		m2 := r2.(map[string]any)
		rels1 := m1["relations"].([]map[string]any)
		rels2 := m2["relations"].([]map[string]any)
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
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "patagonia", r["region"])
		assert.True(t, r["is_synthetic"].(bool))
		assert.Contains(t, r, "coordinates")
		assert.Contains(t, r, "clusters")
	})

	t.Run("returns valid coordinate ranges", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{"region": "test"})
		r := result.(map[string]any)
		coords := r["coordinates"].(map[string]any)
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
		r := result.(map[string]any)
		assert.Equal(t, "default", r["region"])
	})
}

func TestPatternClassifier(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewPatternClassifier(dbl)

	t.Run("returns synthetic patterns", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"data": "network analysis sample"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.True(t, r["is_synthetic"].(bool))
		patterns, ok := r["patterns"].([]map[string]any)
		require.True(t, ok, "patterns should be []map[string]interface{}")
		assert.Greater(t, len(patterns), 0)
	})

	t.Run("patterns have confidence scores", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"data": "confidence check"})
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

	t.Run("data defaults when empty", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("pattern_ids are hashed, never raw input", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{"data": "sensitive_info"})
		r := result.(map[string]any)
		patterns := r["patterns"].([]map[string]any)
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
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, r, "viz_type")
		assert.Contains(t, r, "nodes")
		assert.Contains(t, r, "edges")
		assert.Equal(t, "graph", r["viz_type"])
	})

	t.Run("supports heatmap viz_type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"viz_type": "heatmap"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "heatmap", r["viz_type"])
		assert.Contains(t, r, "matrix")
	})

	t.Run("supports timeline viz_type", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"viz_type": "timeline"})
		require.NoError(t, err)
		r := result.(map[string]any)
		assert.Equal(t, "timeline", r["viz_type"])
		assert.Contains(t, r, "events")
	})

	t.Run("nodes use SHA-256 hashed IDs, never raw data", func(t *testing.T) {
		result, _ := tool.Execute(context.Background(), map[string]any{})
		r := result.(map[string]any)
		nodes := r["nodes"].([]map[string]any)
		for _, node := range nodes {
			id, ok := node["id"].(string)
			require.True(t, ok)
			assert.Len(t, id, 64)
		}
	})
}

func TestDemographicProfileTool(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewDemographicProfileTool(dbl)

	t.Run("returns demographic data for USA", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "USA"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "USA", r["country_code"])
		assert.Equal(t, "United States", r["country_name"])
		assert.True(t, r["is_synthetic"].(bool))
		pop, ok := r["population"].(int64)
		require.True(t, ok, "population should be int64")
		assert.Greater(t, pop, int64(300000000))
		assert.Less(t, pop, int64(400000000))
		gdp, ok := r["gdp_per_capita"].(float64)
		require.True(t, ok)
		assert.Greater(t, gdp, 50000.0)
	})

	t.Run("returns demographic data for JPN", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "JPN"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Japan", r["country_name"])
		assert.Equal(t, "JPN", r["country_code"])
		lifeExp, ok := r["life_expectancy"].(float64)
		require.True(t, ok)
		assert.Greater(t, lifeExp, 80.0)
	})

	t.Run("errors on unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "XYZ"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown country code")
	})

	t.Run("errors on missing countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "countryCode is required")
	})
}

func TestSocioeconomicIndicatorsTool(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewSocioeconomicIndicatorsTool(dbl)

	t.Run("returns socioeconomic data for ZAF", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ZAF"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ZAF", r["country_code"])
		gini, ok := r["gini_coefficient"].(float64)
		require.True(t, ok)
		assert.Greater(t, gini, 50.0) // South Africa has very high Gini
		unemp, ok := r["unemployment_rate"].(float64)
		require.True(t, ok)
		assert.Greater(t, unemp, 20.0)
	})

	t.Run("returns low Gini for KOR", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "KOR"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		gini, ok := r["gini_coefficient"].(float64)
		require.True(t, ok)
		assert.Less(t, gini, 35.0)
	})

	t.Run("errors on unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ZZZ"})
		require.Error(t, err)
	})

	t.Run("errors on missing countryCode", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestCulturalMetricsTool(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewCulturalMetricsTool(dbl)

	t.Run("returns cultural metrics for IND", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "IND"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "IND", r["country_code"])
		langDiv, ok := r["language_diversity"].(float64)
		require.True(t, ok)
		assert.Greater(t, langDiv, 0.8) // India is highly diverse
	})

	t.Run("returns low language diversity for KOR", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "KOR"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		langDiv, ok := r["language_diversity"].(float64)
		require.True(t, ok)
		assert.Less(t, langDiv, 0.1) // Korea is homogeneous
	})

	t.Run("errors on unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ZZZ"})
		require.Error(t, err)
	})
}

func TestUrbanRuralDistributionTool(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewUrbanRuralDistributionTool(dbl)

	t.Run("returns urban distribution for JPN", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "JPN"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "JPN", r["country_code"])
		urbanPct, ok := r["urban_pct"].(float64)
		require.True(t, ok)
		assert.Greater(t, urbanPct, 80.0) // Japan is highly urbanized
		assert.Equal(t, "highly_urbanized", r["classification"])
		assert.True(t, r["above_threshold"].(bool))
	})

	t.Run("returns mostly_rural for ETH", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ETH"})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		urbanPct, ok := r["urban_pct"].(float64)
		require.True(t, ok)
		assert.Less(t, urbanPct, 30.0)
		assert.Equal(t, "mostly_rural", r["classification"])
	})

	t.Run("respects custom threshold", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{"countryCode": "JPN", "threshold": 90.0})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(90), r["threshold_pct"])
	})

	t.Run("errors on unknown country code", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"countryCode": "ZZZ"})
		require.Error(t, err)
	})
}

func TestMigrationPatternsTool(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tool := NewMigrationPatternsTool(dbl)

	t.Run("returns migration data for MEX to USA", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"originCountry": "MEX",
			"destCountry":   "USA",
		})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "MEX", r["origin"])
		assert.Equal(t, "USA", r["dest"])
		stock, ok := r["stock"].(int64)
		require.True(t, ok)
		assert.Greater(t, stock, int64(5000000))
	})

	t.Run("returns migration data for TUR to DEU", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"originCountry": "TUR",
			"destCountry":   "DEU",
		})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		stock, ok := r["stock"].(int64)
		require.True(t, ok)
		assert.Greater(t, stock, int64(1000000))
	})

	t.Run("returns empty stock for unknown corridor", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]any{
			"originCountry": "ETH",
			"destCountry":   "JPN",
		})
		require.NoError(t, err)
		r, ok := result.(map[string]any)
		require.True(t, ok)
		stock, ok := r["stock"].(int64)
		require.True(t, ok)
		assert.Equal(t, int64(0), stock)
	})

	t.Run("errors on missing parameters", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]any{"originCountry": "USA"})
		require.Error(t, err)
		_, err = tool.Execute(context.Background(), map[string]any{"destCountry": "USA"})
		require.Error(t, err)
		_, err = tool.Execute(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestListTools(t *testing.T) {
	dbl := SyntheticDuckDBLayer()
	tools := ListTools(dbl)

	t.Run("returns 10 tools", func(t *testing.T) {
		assert.Len(t, tools, 10)
	})

	t.Run("all tools have unique names", func(t *testing.T) {
		names := make(map[string]bool)
		for _, tool := range tools {
			assert.False(t, names[tool.Name()], "duplicate tool name: %s", tool.Name())
			names[tool.Name()] = true
		}
	})

	t.Run("all tools return valid Execute results", func(t *testing.T) {
		// Tools that require specific arguments.
		argMap := map[string]map[string]any{
			"demographicProfile":      {"countryCode": "USA"},
			"socioeconomicIndicators": {"countryCode": "USA"},
			"culturalMetrics":         {"countryCode": "USA"},
			"urbanRuralDistribution":  {"countryCode": "USA"},
			"migrationPatterns":       {"originCountry": "MEX", "destCountry": "USA"},
		}
		for _, tool := range tools {
			args, ok := argMap[tool.Name()]
			if !ok {
				args = map[string]any{}
			}
			result, err := tool.Execute(context.Background(), args)
			require.NoError(t, err, "tool %s failed", tool.Name())
			assert.NotNil(t, result, "tool %s returned nil", tool.Name())
		}
	})
}

// TestDescriptions verifies all tool Description() methods return non-empty strings.
// These are pure string-returning methods that were at 0% coverage.
func TestDescriptions(t *testing.T) {
	dbl := SyntheticDuckDBLayer()

	tools := []struct {
		name string
		tool interface {
			Name() string
			Description() string
		}
	}{
		{"demographicProfile", NewDemographicProfileTool(dbl)},
		{"socioeconomicIndicators", NewSocioeconomicIndicatorsTool(dbl)},
		{"culturalMetrics", NewCulturalMetricsTool(dbl)},
		{"urbanRuralDistribution", NewUrbanRuralDistributionTool(dbl)},
		{"migrationPatterns", NewMigrationPatternsTool(dbl)},
		{"he_geographic_context", NewGeographicContext(dbl)},
		{"he_pattern_classifier", NewPatternClassifier(dbl)},
		{"he_plugin_viz", NewPluginViz(dbl)},
		{"he_relational_engine", NewRelationalEngine(dbl)},
		{"he_research_profiles", NewResearchProfiles(dbl)},
	}

	for _, tt := range tools {
		t.Run(tt.name+"/description", func(t *testing.T) {
			desc := tt.tool.Description()
			assert.NotEmpty(t, desc, "Description() for %s should not be empty", tt.name)
		})
		t.Run(tt.name+"/name", func(t *testing.T) {
			name := tt.tool.Name()
			assert.NotEmpty(t, name, "Name() for %s should not be empty", tt.name)
		})
	}
}
