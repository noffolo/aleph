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

// TestSynthesisWithShadowbroker_Empty verifies empty shadowbroker works
func TestSynthesisWithShadowbroker_Empty(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := &osint.Shadowbroker{}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	assert.NotNil(t, se)
	assert.Equal(t, sb, se.shadowbroker)
}

// TestGetUnifiedToolIntel_WithMultipleDataSources tests combining CodeFlow and HE data
func TestGetUnifiedToolIntel_WithMultipleDataSources(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()

	// Setup CodeFlow data
	err := cf.RecordExecution(context.Background(), "multi-source-tool", codeflow.ExecutionMetrics{
		Duration:      200,
		MemoryBytes:   1024,
		CPUMillicores: 500,
		CallCount:     20,
		ErrorCount:    2,
		TotalCalls:    20,
	})
	assert.NoError(t, err)

	// Setup usage data
	err = ut.RecordUsage(context.Background(), "user1", "multi-source-tool", "analysis")
	assert.NoError(t, err)
	err = ut.RecordUsage(context.Background(), "user2", "multi-source-tool", "research")
	assert.NoError(t, err)

	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "multi-source-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)

	assert.Equal(t, "multi-source-tool", intel.ToolID)
	assert.Equal(t, int64(1), intel.ExecutionCount) // CallCount=1 after first RecordExecution
	assert.Equal(t, 2, intel.UsageFrequency)
	assert.Len(t, intel.TopUsers, 2)

	// With ErrorCount=2 and TotalCalls=1 (first call), error rate is 200%
	foundErrorRec := false
	for _, r := range intel.Recommendations {
		if contains(r, "Error rate") {
			foundErrorRec = true
		}
	}
	assert.True(t, foundErrorRec, "high error rate should trigger recommendation")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGetCrossContextRecommendations_ScoreBounds verifies scores are clamped [0, 100]
func TestGetCrossContextRecommendations_ScoreBounds(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	err := cf.RecordExecution(context.Background(), "bad-tool", codeflow.ExecutionMetrics{
		Duration:   100,
		CallCount:  100,
		ErrorCount: 99,
		TotalCalls: 100,
	})
	assert.NoError(t, err)

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		assert.GreaterOrEqual(t, rec.Score, 0.0, "score should be >= 0")
		assert.LessOrEqual(t, rec.Score, 100.0, "score should be <= 100")
	}
}

// TestToolRecommendation_ReasonValues verifies reasons are set correctly
func TestToolRecommendation_ReasonValues(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	err := cf.RecordExecution(context.Background(), "good-tool", codeflow.ExecutionMetrics{
		Duration:   50,
		CallCount:  5,
		ErrorCount: 0,
		TotalCalls: 5,
	})
	assert.NoError(t, err)

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		assert.NotEmpty(t, rec.Reason)
	}
}

// TestMultipleRecs verifies multiple tools produce multiple recommendations
func TestMultipleRecs(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := NewSynthesisEngine(cf, ut, sb, slog.Default())

	for _, toolID := range []string{"tool-a", "tool-b", "tool-c"} {
		err := cf.RecordExecution(context.Background(), toolID, codeflow.ExecutionMetrics{
			Duration:   100,
			CallCount:  5,
			ErrorCount: 0,
			TotalCalls: 5,
		})
		assert.NoError(t, err)
	}

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Len(t, recs, 3)
}
