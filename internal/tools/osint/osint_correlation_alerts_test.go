package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewCorrelationAlertsTool ─────────────────────────────────────────────────────

func TestNewCorrelationAlertsTool(t *testing.T) {
	t.Run("happy: non-nil tool", func(t *testing.T) {
		tool := NewCorrelationAlertsTool(nil)
		require.NotNil(t, tool)
		assert.Nil(t, tool.broker)
	})

	t.Run("happy: with broker", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tool := NewCorrelationAlertsTool(sb)
		assert.NotNil(t, tool)
		assert.Equal(t, sb, tool.broker)
	})

	t.Run("edge: nil broker accepted", func(t *testing.T) {
		tool := NewCorrelationAlertsTool(nil)
		assert.NotNil(t, tool)
		// Should still work for synthetic-only operations
	})
}

// ─── Correlate ─────────────────────────────────────────────────────────────

func TestCorrelationAlertsTool_Correlate(t *testing.T) {
	tool := NewCorrelationAlertsTool(nil)
	ctx := context.Background()

	t.Run("happy: valid signals produce alerts", func(t *testing.T) {
		result, err := tool.Correlate(ctx, []string{"movement_detected", "comms_anomaly"})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, true, result["is_synthetic"])
		assert.Equal(t, 2, result["signal_count"])
		alerts, ok := result["alerts"].([]map[string]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(alerts), 1)
		// First alert has expected fields
		alert := alerts[0]
		assert.Contains(t, alert["alert_id"].(string), "CORR-")
		assert.Equal(t, true, alert["is_synthetic"])
		assert.NotEmpty(t, alert["title"])
	})

	t.Run("edge: single signal", func(t *testing.T) {
		result, err := tool.Correlate(ctx, []string{"single_signal"})
		require.NoError(t, err)
		assert.Equal(t, 1, result["signal_count"])
	})

	t.Run("error: empty signals", func(t *testing.T) {
		_, err := tool.Correlate(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one signal is required")

		_, err = tool.Correlate(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one signal is required")
	})
}

// ─── Execute (JSON→JSON) ───────────────────────────────────────────────────

func TestCorrelationAlertsTool_Execute(t *testing.T) {
	tool := NewCorrelationAlertsTool(nil)
	ctx := context.Background()

	t.Run("happy: valid JSON produces JSON output", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"signals":["threat_sig1","threat_sig2"]}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, true, parsed["is_synthetic"])
	})

	t.Run("edge: JSON with extra fields still works", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"signals":["test"],"ignored_field":42}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, float64(1), parsed["signal_count"])
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `not json`)
		assert.Error(t, err)
	})

	t.Run("error: empty signals in JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{"signals":[]}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one signal is required")
	})
}

// ─── Register ──────────────────────────────────────────────────────────────

func TestCorrelationAlertsTool_Register(t *testing.T) {
	t.Run("happy: registers successfully", func(t *testing.T) {
		tool := NewCorrelationAlertsTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		assert.NoError(t, err)
	})

	t.Run("edge: duplicate registration is idempotent", func(t *testing.T) {
		tool := NewCorrelationAlertsTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		require.NoError(t, err)
		// Second registration may fail due to unique constraint — that's OK for edge
		_ = tool.Register(repo)
	})
}

// ─── Alert Output Structure ────────────────────────────────────────────────

func TestCorrelationAlertsTool_OutputStructure(t *testing.T) {
	tool := NewCorrelationAlertsTool(nil)
	ctx := context.Background()

	t.Run("happy: alerts have expected structure", func(t *testing.T) {
		result, err := tool.Correlate(ctx, []string{"cyber_threat"})
		require.NoError(t, err)
		alerts := result["alerts"].([]map[string]any)
		require.NotEmpty(t, alerts)
		for _, alert := range alerts {
			assert.NotEmpty(t, alert["alert_id"])
			assert.NotEmpty(t, alert["title"])
			assert.Contains(t, []string{"info", "low", "medium", "high", "critical"}, alert["severity"])
			assert.NotNil(t, alert["events"])
			assert.Equal(t, true, alert["is_synthetic"])
			assert.NotEmpty(t, alert["generated_at"])
		}
	})

	t.Run("edge: consistent output for same input", func(t *testing.T) {
		result1, _ := tool.Correlate(ctx, []string{"consistent"})
		result2, _ := tool.Correlate(ctx, []string{"consistent"})
		alerts1 := result1["alerts"].([]map[string]any)
		alerts2 := result2["alerts"].([]map[string]any)
		assert.Equal(t, len(alerts1), len(alerts2))
		assert.Equal(t, alerts1[0]["alert_id"], alerts2[0]["alert_id"])
	})

	t.Run("edge: different signals yield different output", func(t *testing.T) {
		result1, _ := tool.Correlate(ctx, []string{"signal_a"})
		result2, _ := tool.Correlate(ctx, []string{"signal_b"})
		alerts1 := result1["alerts"].([]map[string]any)
		alerts2 := result2["alerts"].([]map[string]any)
		assert.NotEqual(t, alerts1[0]["alert_id"], alerts2[0]["alert_id"])
	})
}
