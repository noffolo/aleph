package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVesselTrackingTool(t *testing.T) {
	t.Run("happy: non-nil with nil broker", func(t *testing.T) {
		tool := NewVesselTrackingTool(nil)
		require.NotNil(t, tool)
		assert.Nil(t, tool.broker)
	})

	t.Run("happy: with broker", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tool := NewVesselTrackingTool(sb)
		assert.NotNil(t, tool)
		assert.Equal(t, sb, tool.broker)
	})

	t.Run("edge: nil broker accepted", func(t *testing.T) {
		tool := NewVesselTrackingTool(nil)
		assert.NotNil(t, tool)
	})
}

func TestVesselTrackingTool_Track(t *testing.T) {
	tool := NewVesselTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: valid MMSI returns vessel data", func(t *testing.T) {
		result, err := tool.Track(ctx, "123456789")
		require.NoError(t, err)
		assert.Equal(t, "123456789", result["mmsi"])
		assert.Equal(t, true, result["is_synthetic"])
		assert.NotEmpty(t, result["vessel_name"])
	})

	t.Run("edge: short MMSI works", func(t *testing.T) {
		result, err := tool.Track(ctx, "111")
		require.NoError(t, err)
		assert.Equal(t, "111", result["mmsi"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("error: empty MMSI", func(t *testing.T) {
		_, err := tool.Track(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mmsi is required")
	})
}

func TestVesselTrackingTool_Execute(t *testing.T) {
	tool := NewVesselTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: valid JSON returns JSON string", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"mmsi":"987654321"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "987654321", parsed["mmsi"])
		assert.Equal(t, true, parsed["is_synthetic"])
	})

	t.Run("edge: extra JSON fields ignored", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"mmsi":"555666777","extra":"value"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "555666777", parsed["mmsi"])
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `[not json]`)
		assert.Error(t, err)
	})

	t.Run("error: empty MMSI in JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{"mmsi":""}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mmsi is required")
	})
}

func TestVesselTrackingTool_Register(t *testing.T) {
	t.Run("happy: registers successfully", func(t *testing.T) {
		tool := NewVesselTrackingTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		assert.NoError(t, err)
	})

	t.Run("edge: re-registration attempt", func(t *testing.T) {
		tool := NewVesselTrackingTool(nil)
		repo := newMetadataRepo(t)
		require.NoError(t, tool.Register(repo))
		_ = tool.Register(repo)
	})
}

func TestVesselTrackingTool_OutputStructure(t *testing.T) {
	tool := NewVesselTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: vessel data has expected fields", func(t *testing.T) {
		result, err := tool.Track(ctx, "999888777")
		require.NoError(t, err)
		assert.NotEmpty(t, result["mmsi"])
		assert.NotEmpty(t, result["vessel_name"])
		assert.Greater(t, result["latitude"].(float64), -90.0)
		assert.Less(t, result["latitude"].(float64), 90.0)
		assert.Greater(t, result["longitude"].(float64), -180.0)
		assert.Less(t, result["longitude"].(float64), 180.0)
		assert.GreaterOrEqual(t, result["speed"].(float64), 0.0)
		assert.GreaterOrEqual(t, result["course"].(float64), 0.0)
		assert.Less(t, result["course"].(float64), 360.0)
		assert.Contains(t, []string{"underway", "anchored", "moored", "drifting"}, result["status"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("edge: deterministic for same MMSI", func(t *testing.T) {
		result1, _ := tool.Track(ctx, "VESSEL123")
		result2, _ := tool.Track(ctx, "VESSEL123")
		assert.Equal(t, result1["latitude"], result2["latitude"])
		assert.Equal(t, result1["longitude"], result2["longitude"])
		assert.Equal(t, result1["vessel_name"], result2["vessel_name"])
	})

	t.Run("edge: different MMSI yield different vessels", func(t *testing.T) {
		result1, _ := tool.Track(ctx, "SHIP_A")
		result2, _ := tool.Track(ctx, "SHIP_B")
		assert.NotEqual(t, result1["vessel_name"], result2["vessel_name"])
	})
}

func TestRoundFloat(t *testing.T) {
	t.Run("happy: zero decimals", func(t *testing.T) {
		assert.Equal(t, 42.0, roundFloat(42.3, 0))
		assert.Equal(t, 43.0, roundFloat(42.6, 0))
	})

	t.Run("edge: two decimals", func(t *testing.T) {
		assert.Equal(t, 3.14, roundFloat(3.14159, 2))
		assert.Equal(t, 2.72, roundFloat(2.71828, 2))
	})

	t.Run("edge: four decimals", func(t *testing.T) {
		assert.Equal(t, 1.2346, roundFloat(1.234567, 4))
	})
}
