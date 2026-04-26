package synthesis

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ff3300/aleph-v2/internal/tools/codeflow"
	he "github.com/ff3300/aleph-v2/internal/tools/humanecosystems"
	"github.com/ff3300/aleph-v2/internal/tools/osint"
	"github.com/stretchr/testify/assert"
)

func TestNewSynthesisEngine(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()

	se := NewSynthesisEngine(cf, ut, nil, slog.Default())
	assert.NotNil(t, se)
	assert.Equal(t, cf, se.codeFlow)
	assert.Equal(t, ut, se.usageTracker)
	assert.Nil(t, se.shadowbroker)
	assert.NotNil(t, se.toolIntel)
	assert.NotNil(t, se.logger)
	assert.Zero(t, se.TimeDecayHalfLife)
	assert.False(t, se.startupTime.IsZero())
}

// TestNewSynthesisEngine_NilLogger is intentionally omitted because
// NewSynthesisEngine calls logger.With() which panics on nil.

func TestNewSynthesisEngine_WithShadowbroker(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := &osint.Shadowbroker{}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	assert.NotNil(t, se)
	assert.Equal(t, sb, se.shadowbroker)
}

func TestGetUnifiedToolIntel_EmptyToolID(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	se := NewSynthesisEngine(cf, ut, nil, slog.Default())

	intel, err := se.GetUnifiedToolIntel(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, intel)
	assert.Contains(t, err.Error(), "toolID cannot be empty")
}

func TestGetUnifiedToolIntel_NewTool(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	intel, err := se.GetUnifiedToolIntel(context.Background(), "new-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, "new-tool", intel.ToolID)
	assert.Zero(t, intel.ExecutionCount)
	assert.Zero(t, intel.UsageFrequency)
	assert.Empty(t, intel.Warnings)
	assert.Empty(t, intel.Recommendations)
}

func TestGetUnifiedToolIntel_WithCodeFlowData(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()

	// Record some executions
	err := cf.RecordExecution(context.Background(), "my-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   5,
		ErrorCount:  1,
		TotalCalls:  5,
	})
	assert.NoError(t, err)

	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "my-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	// ExecutionCount maps to CallCount which is 1 after first RecordExecution
	assert.Equal(t, int64(1), intel.ExecutionCount)
}

func TestGetUnifiedToolIntel_WithUsageData(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()

	// Record usage
	err := ut.RecordUsage(context.Background(), "user1", "my-tool", "test context")
	assert.NoError(t, err)
	err = ut.RecordUsage(context.Background(), "user2", "my-tool", "another context")
	assert.NoError(t, err)

	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "my-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, 2, intel.UsageFrequency)
	assert.Len(t, intel.TopUsers, 2)
}

func TestGetCrossContextRecommendations_EmptyUserID(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	se := NewSynthesisEngine(cf, ut, nil, slog.Default())

	recs, err := se.GetCrossContextRecommendations(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, recs)
	assert.Contains(t, err.Error(), "userID cannot be empty")
}

func TestGetCrossContextRecommendations_NoData(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Empty(t, recs)
}

func TestGetCrossContextRecommendations_WithData(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)

	// Add some execution data
	err := cf.RecordExecution(context.Background(), "tool-a", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   10,
		ErrorCount:  0,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.NotEmpty(t, recs)

	for _, rec := range recs {
		assert.NotEmpty(t, rec.ToolID)
		assert.GreaterOrEqual(t, rec.Score, 0.0)
		assert.LessOrEqual(t, rec.Score, 100.0)
		assert.NotEmpty(t, rec.Reason)
	}
}

func TestGetCrossContextRecommendations_HighErrorRate(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)

	err := cf.RecordExecution(context.Background(), "buggy-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   10,
		ErrorCount:  8,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.NotEmpty(t, recs)

	// High error rate should produce suggestions
	for _, rec := range recs {
		if rec.ToolID == "buggy-tool" {
			assert.NotEmpty(t, rec.Suggestions)
		}
	}
}

func TestNewToolIntel(t *testing.T) {
	intel := osint.NewToolIntel()
	assert.NotNil(t, intel)
}

func TestUnifiedToolIntel_Fields(t *testing.T) {
	intel := &UnifiedToolIntel{
		ToolID:           "test-tool",
		Name:             "Test Tool",
		Category:         "analysis",
		HealthStatus:     "healthy",
		ExecutionCount:   100,
		SecurityRiskScore: 25.0,
		Warnings:         []string{"warning 1"},
		Recommendations:  []string{"rec 1"},
	}

	assert.Equal(t, "test-tool", intel.ToolID)
	assert.Equal(t, "Test Tool", intel.Name)
	assert.Equal(t, "analysis", intel.Category)
	assert.Equal(t, "healthy", intel.HealthStatus)
	assert.Equal(t, int64(100), intel.ExecutionCount)
	assert.Equal(t, 25.0, intel.SecurityRiskScore)
	assert.Len(t, intel.Warnings, 1)
	assert.Len(t, intel.Recommendations, 1)
}

func TestToolRecommendation_Fields(t *testing.T) {
	rec := &ToolRecommendation{
		ToolID:     "tool-1",
		Score:      85.5,
		Reason:     "Well-performing tool with good metrics",
		Suggestions: []string{"Monitor performance"},
	}

	assert.Equal(t, "tool-1", rec.ToolID)
	assert.Equal(t, 85.5, rec.Score)
	assert.Equal(t, "Well-performing tool with good metrics", rec.Reason)
	assert.Len(t, rec.Suggestions, 1)
}

func TestGetCrossContextRecommendations_WithTimeDecay(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)

	err := cf.RecordExecution(context.Background(), "tool-a", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   10,
		ErrorCount:  0,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	se.TimeDecayHalfLife = 24 * 7 // 7 hours (unrealistic but tests the code path)

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		assert.GreaterOrEqual(t, rec.Score, 0.0)
		assert.LessOrEqual(t, rec.Score, 100.0)
	}
}

func TestGetCrossContextRecommendations_EmptyAfterNoExecutions(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	// No executions recorded but usage tracked
	err := ut.RecordUsage(context.Background(), "user1", "tool-without-exec", "test")
	assert.NoError(t, err)

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Empty(t, recs) // No execution records, so no tools to recommend
}

func TestGetUnifiedToolIntel_MultipleCalls(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{}, nil)
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	// Multiple calls for the same tool should work
	intel1, err := se.GetUnifiedToolIntel(context.Background(), "multi-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel1)

	intel2, err := se.GetUnifiedToolIntel(context.Background(), "multi-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel2)
}


