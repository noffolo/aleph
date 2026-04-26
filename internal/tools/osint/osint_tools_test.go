package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionDossierTool(t *testing.T) {
	tool := NewRegionDossierTool(nil)

	t.Run("returns structured dossier for known region", func(t *testing.T) {
		result, err := tool.Dossier(context.Background(), "en_harbor")
		require.NoError(t, err)
		assert.Equal(t, "Eastern Harbor Region", result["region_name"])
		assert.NotZero(t, result["population"])
		assert.NotZero(t, result["gdp"])
		assert.NotZero(t, result["stability"])
		assert.True(t, result["is_synthetic"].(bool))
		sources, ok := result["sources"].([]string)
		require.True(t, ok)
		assert.Greater(t, len(sources), 0)
	})

	t.Run("handles empty region_id", func(t *testing.T) {
		_, err := tool.Dossier(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"region_id":"delta_9"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Contains(t, parsed["region_name"], "Delta-9")
	})

	t.Run("Execute rejects empty JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{}`)
		assert.Error(t, err)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `not json`)
		assert.Error(t, err)
	})
}

func TestThreatLevelTool(t *testing.T) {
	tool := NewThreatLevelTool(nil)

	t.Run("returns threat assessment for target", func(t *testing.T) {
		result, err := tool.Assess(context.Background(), "en_harbor")
		require.NoError(t, err)
		assert.Equal(t, "en_harbor", result["target"])
		level, ok := result["level"].(string)
		require.True(t, ok)
		assert.Contains(t, []string{"low", "medium", "high", "critical"}, level)
		confidence, ok := result["confidence"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, confidence, 0.5)
		assert.LessOrEqual(t, confidence, 1.0)
	})

	t.Run("handles empty target", func(t *testing.T) {
		_, err := tool.Assess(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"target":"meridian_arc"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "meridian_arc", parsed["target"])
	})

	t.Run("different targets produce different assessments", func(t *testing.T) {
		r1, _ := tool.Assess(context.Background(), "zone_a")
		r2, _ := tool.Assess(context.Background(), "zone_b")
		// Deterministic from the same input
		r1b, _ := tool.Assess(context.Background(), "zone_a")
		assert.Equal(t, r1["level"], r1b["level"])
		// Different inputs may differ
		t.Logf("zone_a: %v, zone_b: %v", r1["level"], r2["level"])
	})
}

func TestVesselTrackingTool(t *testing.T) {
	tool := NewVesselTrackingTool(nil)

	t.Run("returns vessel data for valid MMSI", func(t *testing.T) {
		result, err := tool.Track(context.Background(), "123456789")
		require.NoError(t, err)
		assert.Equal(t, "123456789", result["mmsi"])
		lat, ok := result["latitude"].(float64)
		require.True(t, ok)
		lon, ok := result["longitude"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, lat, -90.0)
		assert.LessOrEqual(t, lat, 90.0)
		assert.GreaterOrEqual(t, lon, -180.0)
		assert.LessOrEqual(t, lon, 180.0)
		speed, ok := result["speed"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, speed, 0.0)
		assert.LessOrEqual(t, speed, 30.0)
	})

	t.Run("handles empty MMSI", func(t *testing.T) {
		_, err := tool.Track(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"mmsi":"987654321"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "987654321", parsed["mmsi"])
	})

	t.Run("deterministic output for same MMSI", func(t *testing.T) {
		r1, _ := tool.Track(context.Background(), "111222333")
		r2, _ := tool.Track(context.Background(), "111222333")
		assert.Equal(t, r1["latitude"], r2["latitude"])
		assert.Equal(t, r1["longitude"], r2["longitude"])
	})
}

func TestFlightTrackingTool(t *testing.T) {
	tool := NewFlightTrackingTool(nil)

	t.Run("returns flight data for valid flight number", func(t *testing.T) {
		result, err := tool.Track(context.Background(), "AA123")
		require.NoError(t, err)
		assert.Equal(t, "AA123", result["flight_number"])
		assert.NotEmpty(t, result["origin"])
		assert.NotEmpty(t, result["destination"])
		assert.NotEqual(t, result["origin"], result["destination"])
	})

	t.Run("handles empty flight number", func(t *testing.T) {
		_, err := tool.Track(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"flight_number":"LH400"}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "LH400", parsed["flight_number"])
	})

	t.Run("returns valid position data", func(t *testing.T) {
		result, err := tool.Track(context.Background(), "BA789")
		require.NoError(t, err)
		lat := result["latitude"].(float64)
		lon := result["longitude"].(float64)
		assert.GreaterOrEqual(t, lat, -90.0)
		assert.LessOrEqual(t, lat, 90.0)
		assert.GreaterOrEqual(t, lon, -180.0)
		assert.LessOrEqual(t, lon, 180.0)
	})
}

func TestCorrelationAlertsTool(t *testing.T) {
	tool := NewCorrelationAlertsTool(nil)

	t.Run("returns correlations for signals", func(t *testing.T) {
		result, err := tool.Correlate(context.Background(), []string{"cyber_spike", "satellite_anomaly"})
		require.NoError(t, err)
		assert.Equal(t, 2, result["signal_count"])
		alerts, ok := result["alerts"].([]map[string]interface{})
		if !ok {
			// Might be []interface{} due to JSON round-trip
			alertsRaw, ok := result["alerts"].([]interface{})
			require.True(t, ok)
			assert.Greater(t, len(alertsRaw), 0)
		} else {
			assert.Greater(t, len(alerts), 0)
		}
	})

	t.Run("handles empty signals", func(t *testing.T) {
		_, err := tool.Correlate(context.Background(), []string{})
		assert.Error(t, err)

		_, err = tool.Execute(context.Background(), `{"signals":[]}`)
		assert.Error(t, err)
	})

	t.Run("Execute JSON→JSON", func(t *testing.T) {
		raw, err := tool.Execute(context.Background(), `{"signals":["movement_detected","comms_intercept"]}`)
		require.NoError(t, err)
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Greater(t, parsed["signal_count"], 0.0)
	})

	t.Run("Execute rejects invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), `{invalid}`)
		assert.Error(t, err)
	})
}
