package osint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewThreatLevelTool(t *testing.T) {
	t.Run("happy: non-nil with nil broker", func(t *testing.T) {
		tool := NewThreatLevelTool(nil)
		require.NotNil(t, tool)
		assert.Nil(t, tool.broker)
	})

	t.Run("happy: with broker", func(t *testing.T) {
		sb := NewShadowbroker(ShadowbrokerConfig{BaseURL: ""})
		tool := NewThreatLevelTool(sb)
		assert.NotNil(t, tool)
		assert.Equal(t, sb, tool.broker)
	})

	t.Run("edge: nil broker accepted", func(t *testing.T) {
		tool := NewThreatLevelTool(nil)
		assert.NotNil(t, tool)
	})
}

func TestThreatLevelTool_Assess(t *testing.T) {
	tool := NewThreatLevelTool(nil)
	ctx := context.Background()

	t.Run("happy: valid target returns assessment", func(t *testing.T) {
		result, err := tool.Assess(ctx, "harbor_district")
		require.NoError(t, err)
		assert.Equal(t, "harbor_district", result["target"])
		assert.Equal(t, true, result["is_synthetic"])
		assert.Contains(t, []string{"low", "medium", "high", "critical"}, result["level"])
		assert.Greater(t, result["confidence"].(float64), 0.0)
		assert.LessOrEqual(t, result["confidence"].(float64), 1.0)
	})

	t.Run("edge: numeric target works", func(t *testing.T) {
		result, err := tool.Assess(ctx, "192.168.1.1")
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.1", result["target"])
		assert.Contains(t, []string{"low", "medium", "high", "critical"}, result["level"])
	})

	t.Run("error: empty target", func(t *testing.T) {
		_, err := tool.Assess(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target is required")
	})
}

func TestThreatLevelTool_Execute(t *testing.T) {
	tool := NewThreatLevelTool(nil)
	ctx := context.Background()

	t.Run("happy: valid JSON returns JSON string", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"target":"airport_zone"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "airport_zone", parsed["target"])
		assert.Equal(t, true, parsed["is_synthetic"])
	})

	t.Run("edge: extra JSON fields ignored", func(t *testing.T) {
		raw, err := tool.Execute(ctx, `{"target":"test_area","severity":"high"}`)
		require.NoError(t, err)
		var parsed map[string]any
		err = json.Unmarshal([]byte(raw), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "test_area", parsed["target"])
	})

	t.Run("error: invalid JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `not json`)
		assert.Error(t, err)
	})

	t.Run("error: empty target in JSON", func(t *testing.T) {
		_, err := tool.Execute(ctx, `{"target":""}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target is required")
	})
}

func TestThreatLevelTool_Register(t *testing.T) {
	t.Run("happy: registers successfully", func(t *testing.T) {
		tool := NewThreatLevelTool(nil)
		repo := newMetadataRepo(t)
		err := tool.Register(repo)
		assert.NoError(t, err)
	})

	t.Run("edge: re-registration attempt", func(t *testing.T) {
		tool := NewThreatLevelTool(nil)
		repo := newMetadataRepo(t)
		require.NoError(t, tool.Register(repo))
		_ = tool.Register(repo)
	})
}

func TestThreatLevelTool_OutputStructure(t *testing.T) {
	tool := NewThreatLevelTool(nil)
	ctx := context.Background()

	t.Run("happy: output has all required fields", func(t *testing.T) {
		result, err := tool.Assess(ctx, "critical_infra")
		require.NoError(t, err)
		assert.NotEmpty(t, result["target"])
		assert.NotEmpty(t, result["level"])
		assert.NotEmpty(t, result["description"])
		assert.NotEmpty(t, result["vector"])
		assert.Contains(t, []string{"cyber", "physical", "economic", "social", "environmental"}, result["vector"])
		assert.Equal(t, true, result["is_synthetic"])
		assert.NotEmpty(t, result["generated_at"])
	})

	t.Run("edge: deterministic for same target", func(t *testing.T) {
		result1, _ := tool.Assess(ctx, "deterministic")
		result2, _ := tool.Assess(ctx, "deterministic")
		assert.Equal(t, result1["level"], result2["level"])
		assert.Equal(t, result1["vector"], result2["vector"])
		assert.Equal(t, result1["confidence"], result2["confidence"])
	})

	t.Run("edge: different targets differ", func(t *testing.T) {
		result1, _ := tool.Assess(ctx, "alpha")
		result2, _ := tool.Assess(ctx, "beta")
		assert.NotEqual(t, result1["level"], result2["level"])
	})
}
