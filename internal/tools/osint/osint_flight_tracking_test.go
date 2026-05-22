package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlightTrackingTool(t *testing.T) {
	t.Run("happy: non-nil with nil broker", func(t *testing.T) {
		tool := NewFlightTrackingTool(nil)
		require.NotNil(t, tool)
		assert.Nil(t, tool.broker)
	})

	t.Run("happy: with broker", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tool := NewFlightTrackingTool(sb)
		assert.NotNil(t, tool)
		assert.Equal(t, sb, tool.broker)
	})

	t.Run("edge: nil broker accepted", func(t *testing.T) {
		tool := NewFlightTrackingTool(nil)
		assert.NotNil(t, tool)
	})
}

func TestFlightTrackingTool_Track(t *testing.T) {
	tool := NewFlightTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: valid flight number returns data", func(t *testing.T) {
		result, err := tool.Track(ctx, "UA123")
		require.NoError(t, err)
		assert.Equal(t, "UA123", result["flight_number"])
		assert.Equal(t, true, result["is_synthetic"])
		assert.NotEmpty(t, result["flight_number"])
		assert.NotEmpty(t, result["airline"])
		assert.NotEmpty(t, result["origin"])
		assert.NotEmpty(t, result["destination"])
		assert.NotEqual(t, result["origin"], result["destination"])
	})

	t.Run("edge: numeric flight number works", func(t *testing.T) {
		result, err := tool.Track(ctx, "1234")
		require.NoError(t, err)
		assert.Equal(t, "1234", result["flight_number"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("error: empty flight number", func(t *testing.T) {
		_, err := tool.Track(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flight_number is required")
	})
}

func TestFlightTrackingTool_Execute(t *testing.T) {
	tool := NewFlightTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: valid JSON returns JSON string", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"flight_number":"DL555"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "DL555", parsed["flight_number"])
		assert.Equal(t, true, parsed["is_synthetic"])
	})

	t.Run("edge: extra JSON fields ignored", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"flight_number":"AA100","extra":"ignored"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "AA100", parsed["flight_number"])
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{broken`)
		assert.Error(t, err)
	})

	t.Run("error: empty flight_number in JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{"flight_number":""}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flight_number is required")
	})
}

func TestFlightTrackingTool_Register(t *testing.T) {
	t.Run("happy: registers successfully", func(t *testing.T) {
		tool := NewFlightTrackingTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		assert.NoError(t, err)
	})

	t.Run("edge: re-registration attempt", func(t *testing.T) {
		tool := NewFlightTrackingTool(nil)
		repo := newMetadataRepo(t)
		require.NoError(t, tool.Register(repo))
		_ = tool.Register(repo)
	})
}

func TestFlightTrackingTool_OutputStructure(t *testing.T) {
	tool := NewFlightTrackingTool(nil)
	ctx := context.Background()

	t.Run("happy: flight data has expected fields", func(t *testing.T) {
		result, err := tool.Track(ctx, "BA2490")
		require.NoError(t, err)
		assert.NotEmpty(t, result["flight_number"])
		assert.NotEmpty(t, result["airline"])
		assert.NotEmpty(t, result["origin"])
		assert.NotEmpty(t, result["destination"])
		assert.Greater(t, result["latitude"].(float64), -90.0)
		assert.Less(t, result["latitude"].(float64), 90.0)
		assert.Greater(t, result["longitude"].(float64), -180.0)
		assert.Less(t, result["longitude"].(float64), 180.0)
		assert.Greater(t, result["altitude"].(float64), 0.0)
		assert.Greater(t, result["speed"].(float64), 0.0)
		assert.Contains(t, []string{"scheduled", "en_route", "landed", "delayed", "cancelled"}, result["status"])
		assert.Equal(t, true, result["is_synthetic"])
	})

	t.Run("edge: deterministic for same flight number", func(t *testing.T) {
		result1, _ := tool.Track(ctx, "BA2490")
		result2, _ := tool.Track(ctx, "BA2490")
		assert.Equal(t, result1["latitude"], result2["latitude"])
		assert.Equal(t, result1["longitude"], result2["longitude"])
		assert.Equal(t, result1["status"], result2["status"])
	})

	t.Run("edge: different flight numbers differ", func(t *testing.T) {
		result1, _ := tool.Track(ctx, "AA100")
		result2, _ := tool.Track(ctx, "UA200")
		assert.NotEqual(t, result1["airline"], result2["airline"])
	})
}
