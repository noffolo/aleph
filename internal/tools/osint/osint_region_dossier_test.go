package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegionDossierTool(t *testing.T) {
	t.Run("happy: non-nil with nil broker", func(t *testing.T) {
		tool := NewRegionDossierTool(nil)
		require.NotNil(t, tool)
		assert.Nil(t, tool.broker)
	})

	t.Run("happy: with broker", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tool := NewRegionDossierTool(sb)
		assert.NotNil(t, tool)
		assert.Equal(t, sb, tool.broker)
	})

	t.Run("edge: nil broker accepted", func(t *testing.T) {
		tool := NewRegionDossierTool(nil)
		assert.NotNil(t, tool)
	})
}

func TestRegionDossierTool_Dossier(t *testing.T) {
	tool := NewRegionDossierTool(nil)
	ctx := context.Background()

	t.Run("happy: known region ID returns dossier", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "en_harbor")
		require.NoError(t, err)
		assert.Equal(t, "Eastern Harbor Region", result["region_name"])
		assert.Equal(t, true, result["is_synthetic"])
		assert.Greater(t, result["population"].(int), 0)
		assert.Greater(t, result["gdp"].(float64), 0.0)
		assert.GreaterOrEqual(t, result["stability"].(float64), 0.0)
		assert.LessOrEqual(t, result["stability"].(float64), 1.0)
		assert.NotEmpty(t, result["sources"])
	})

	t.Run("happy: another known region ID", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "northern_rise")
		require.NoError(t, err)
		assert.Equal(t, "Northern Rise Territory", result["region_name"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("happy: straits_of_orm", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "straits_of_orm")
		require.NoError(t, err)
		assert.Equal(t, "Straits of Orm", result["region_name"])
	})

	t.Run("happy: delta_9", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "delta_9")
		require.NoError(t, err)
		assert.Equal(t, "Delta-9 Economic Zone", result["region_name"])
	})

	t.Run("happy: meridian_arc", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "meridian_arc")
		require.NoError(t, err)
		assert.Equal(t, "Meridian Arc Corridor", result["region_name"])
	})

	t.Run("edge: unknown region ID derives name", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "custom_zone")
		require.NoError(t, err)
		assert.Equal(t, "Region custom_zone", result["region_name"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("edge: numeric region ID", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "zone_42")
		require.NoError(t, err)
		assert.Equal(t, "Region zone_42", result["region_name"])
	})

	t.Run("edge: empty-string region ID not allowed", func(t *testing.T) {
		_, err := tool.Dossier(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region_id is required")
	})

	t.Run("error: empty region ID", func(t *testing.T) {
		_, err := tool.Dossier(ctx, "")
		require.Error(t, err)
	})
}

func TestRegionDossierTool_Execute(t *testing.T) {
	tool := NewRegionDossierTool(nil)
	ctx := context.Background()

	t.Run("happy: valid JSON returns JSON string", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"region_id":"en_harbor"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "Eastern Harbor Region", parsed["region_name"])
		assert.Equal(t, true, parsed["is_synthetic"])
	})

	t.Run("edge: extra JSON fields ignored", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"region_id":"delta_9","priority":5}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "Delta-9 Economic Zone", parsed["region_name"])
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{broken`)
		assert.Error(t, err)
	})

	t.Run("error: empty region_id in JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{"region_id":""}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region_id is required")
	})
}

func TestRegionDossierTool_Register(t *testing.T) {
	t.Run("happy: registers successfully", func(t *testing.T) {
		tool := NewRegionDossierTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		assert.NoError(t, err)
	})

	t.Run("edge: re-registration attempt", func(t *testing.T) {
		tool := NewRegionDossierTool(nil)
		repo := newMetadataRepo(t)
		require.NoError(t, tool.Register(repo))
		_ = tool.Register(repo)
	})
}

func TestRegionDossierTool_OutputFields(t *testing.T) {
	tool := NewRegionDossierTool(nil)
	ctx := context.Background()

	t.Run("happy: dossier has all expected keys", func(t *testing.T) {
		result, err := tool.Dossier(ctx, "en_harbor")
		require.NoError(t, err)
		expectedKeys := []string{"region_name", "population", "gdp", "stability", "sources", "is_synthetic", "generated_at"}
		for _, k := range expectedKeys {
			assert.Contains(t, result, k, "missing key: %s", k)
		}
	})

	t.Run("edge: deterministic for same region ID", func(t *testing.T) {
		result1, _ := tool.Dossier(ctx, "en_harbor")
		result2, _ := tool.Dossier(ctx, "en_harbor")
		assert.Equal(t, result1["region_name"], result2["region_name"])
		assert.Equal(t, result1["population"], result2["population"])
		assert.Equal(t, result1["gdp"], result2["gdp"])
		assert.Equal(t, result1["stability"], result2["stability"])
	})

	t.Run("edge: different regions differ", func(t *testing.T) {
		result1, _ := tool.Dossier(ctx, "en_harbor")
		result2, _ := tool.Dossier(ctx, "northern_rise")
		assert.NotEqual(t, result1["region_name"], result2["region_name"])
	})
}

func TestDeriveRegionName_Extended(t *testing.T) {
	t.Run("happy: known en_harbor", func(t *testing.T) {
		assert.Equal(t, "Eastern Harbor Region", deriveRegionName("en_harbor"))
	})

	t.Run("happy: known northern_rise", func(t *testing.T) {
		assert.Equal(t, "Northern Rise Territory", deriveRegionName("northern_rise"))
	})

	t.Run("happy: known straits_of_orm", func(t *testing.T) {
		assert.Equal(t, "Straits of Orm", deriveRegionName("straits_of_orm"))
	})

	t.Run("happy: known delta_9", func(t *testing.T) {
		assert.Equal(t, "Delta-9 Economic Zone", deriveRegionName("delta_9"))
	})

	t.Run("happy: known meridian_arc", func(t *testing.T) {
		assert.Equal(t, "Meridian Arc Corridor", deriveRegionName("meridian_arc"))
	})

	t.Run("edge: unknown region derives pattern", func(t *testing.T) {
		assert.Equal(t, "Region custom_zone", deriveRegionName("custom_zone"))
	})

	t.Run("edge: empty region", func(t *testing.T) {
		assert.Equal(t, "Region ", deriveRegionName(""))
	})

	t.Run("edge: single character", func(t *testing.T) {
		assert.Equal(t, "Region x", deriveRegionName("x"))
	})

	t.Run("edge: with special chars", func(t *testing.T) {
		assert.Equal(t, "Region zone-1", deriveRegionName("zone-1"))
	})
}

func TestHashString(t *testing.T) {
	t.Run("happy: non-empty produces non-zero", func(t *testing.T) {
		assert.NotZero(t, hashString("hello"))
	})

	t.Run("happy: different inputs differ", func(t *testing.T) {
		assert.NotEqual(t, hashString("abc"), hashString("abd"))
	})

	t.Run("edge: empty string", func(t *testing.T) {
		h := hashString("")
		assert.GreaterOrEqual(t, h, uint32(0))
	})
}

func TestClampFloat(t *testing.T) {
	t.Run("happy: value within range", func(t *testing.T) {
		assert.Equal(t, 0.5, clampFloat(0.5, 0, 1))
	})

	t.Run("edge: below min", func(t *testing.T) {
		assert.Equal(t, 0.0, clampFloat(-0.5, 0, 1))
	})

	t.Run("edge: above max", func(t *testing.T) {
		assert.Equal(t, 1.0, clampFloat(1.5, 0, 1))
	})

	t.Run("edge: at min boundary", func(t *testing.T) {
		assert.Equal(t, 0.0, clampFloat(0.0, 0, 1))
	})

	t.Run("edge: at max boundary", func(t *testing.T) {
		assert.Equal(t, 1.0, clampFloat(1.0, 0, 1))
	})
}
