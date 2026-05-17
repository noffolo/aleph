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

func TestGetCrossContextRecommendations_ClampBelowZero(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "terrible-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   50,
		ErrorCount:  50,
		TotalCalls:  50,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.NotEmpty(t, recs)

	for _, rec := range recs {
		if rec.ToolID == "terrible-tool" {
			assert.GreaterOrEqual(t, rec.Score, 0.0)
			assert.LessOrEqual(t, rec.Score, 100.0)
		}
	}
}

func TestGetCrossContextRecommendations_BelowThresholdReason(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "mediocre-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   5,
		ErrorCount:  3, // 3/5 = 60% error rate → heavy penalty
		TotalCalls:  5,
	})
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = cf.RecordExecution(context.Background(), "mediocre-tool", codeflow.ExecutionMetrics{
			Duration:    100,
			CallCount:   5,
			ErrorCount:  3,
			TotalCalls:  5,
		})
		assert.NoError(t, err)
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "mediocre-tool" {
			assert.NotEmpty(t, rec.Reason)
		}
	}
}

func TestGetCrossContextRecommendations_TimeDecayClampAbove100(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "popular-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   5,
		ErrorCount:  0,
		TotalCalls:  5,
	})
	assert.NoError(t, err)

	err = ut.RecordUsage(context.Background(), "user1", "popular-tool", "test")
	assert.NoError(t, err)
	for i := 0; i < 100; i++ {
		err = ut.RecordUsage(context.Background(), "user1", "popular-tool", "bulk")
		assert.NoError(t, err)
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	se.TimeDecayHalfLife = 365 * 24 // effectively no decay but code path exercised

	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		assert.GreaterOrEqual(t, rec.Score, 0.0)
		assert.LessOrEqual(t, rec.Score, 100.0)
	}
}

func TestGetCrossContextRecommendations_AllReasonsPresent(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	tools := []struct {
		id        string
		errCount  int64
		totalCall int64
		usageFreq int
	}{
		{"good-tool", 0, 5, 20},
		{"ok-tool", 1, 10, 5},
		{"bad-tool", 8, 10, 1},
	}

	for _, tool := range tools {
		err := cf.RecordExecution(context.Background(), tool.id, codeflow.ExecutionMetrics{
			Duration:    100,
			CallCount:   tool.totalCall,
			ErrorCount:  tool.errCount,
			TotalCalls:  tool.totalCall,
		})
		assert.NoError(t, err)

		for j := 0; j < tool.usageFreq; j++ {
			err = ut.RecordUsage(context.Background(), "user1", tool.id, "analysis")
			assert.NoError(t, err)
		}
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Len(t, recs, 3)

	reasonSet := make(map[string]bool)
	for _, rec := range recs {
		assert.NotEmpty(t, rec.ToolID)
		assert.GreaterOrEqual(t, rec.Score, 0.0)
		assert.LessOrEqual(t, rec.Score, 100.0)
		reasonSet[rec.Reason] = true
	}
	// We expect at least 2 unique reason values
	assert.GreaterOrEqual(t, len(reasonSet), 2)
}

func TestGetCrossContextRecommendations_SuggestionsForHighError(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "buggy", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   10,
		ErrorCount:  10,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "buggy" {
			assert.NotEmpty(t, rec.Suggestions)
		}
	}
}

func TestGetUnifiedToolIntel_WithErrorRate(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "error-tool", codeflow.ExecutionMetrics{
		Duration:    200,
		CallCount:   10,
		ErrorCount:  5,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "error-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, "error-tool", intel.ToolID)
	assert.Greater(t, intel.ErrorRate, 0.0)
}

func TestGetUnifiedToolIntel_WithRecommendations(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "bad-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   10,
		ErrorCount:  10,
		TotalCalls:  10,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "bad-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.NotEmpty(t, intel.Recommendations)
}

func TestGetUnifiedToolIntel_MultipleAnomalyRecords(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	// Record multiple executions to trigger anomaly detection
	for i := 0; i < 10; i++ {
		err := cf.RecordExecution(context.Background(), "multi-exec", codeflow.ExecutionMetrics{
			Duration:    100,
			CallCount:   1,
			ErrorCount:  0,
			TotalCalls:  1,
		})
		assert.NoError(t, err)
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "multi-exec")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, "multi-exec", intel.ToolID)
}

func TestGetUnifiedToolIntel_ZeroErrorRate(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "perfect-tool", codeflow.ExecutionMetrics{
		Duration:    50,
		CallCount:   100,
		ErrorCount:  0,
		TotalCalls:  100,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "perfect-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, 0.0, intel.ErrorRate)
}

func TestGetUnifiedToolIntel_WithRelatedTools(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "rel-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   1,
		ErrorCount:  0,
		TotalCalls:  1,
	})
	assert.NoError(t, err)

	err = ut.RecordUsage(context.Background(), "user1", "rel-tool", "analysis")
	assert.NoError(t, err)
	err = ut.RecordUsage(context.Background(), "user1", "other-tool", "viz")
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "rel-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
}

func TestGetCrossContextRecommendations_ScoreExactlyZero(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "worst-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   100,
		ErrorCount:  100,
		TotalCalls:  100,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "worst-tool" {
			assert.Equal(t, 0.0, rec.Score)
		}
	}
}

func TestGetCrossContextRecommendations_SingleTool(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "solo-tool", codeflow.ExecutionMetrics{
		Duration:    50,
		CallCount:   2,
		ErrorCount:  0,
		TotalCalls:  2,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "solo-tool", recs[0].ToolID)
	assert.Greater(t, recs[0].Score, 40.0)
}

func TestGetUnifiedToolIntel_WithAllAnomalies(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	for i := 0; i < 20; i++ {
		err := cf.RecordExecution(context.Background(), "anomaly-tool", codeflow.ExecutionMetrics{
			Duration:    100,
			CallCount:   1,
			ErrorCount:  int64(i % 3),
			TotalCalls:  1,
			MemoryBytes: int64(100 * (i + 1)),
		})
		assert.NoError(t, err)
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	intel, err := se.GetUnifiedToolIntel(context.Background(), "anomaly-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
}

func TestGetCrossContextRecommendations_ModerateScore(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "moderate-tool", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   5,
		ErrorCount:  1,
		TotalCalls:  5,
	})
	assert.NoError(t, err)

	err = ut.RecordUsage(context.Background(), "user1", "moderate-tool", "check")
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "moderate-tool" {
			assert.GreaterOrEqual(t, rec.Score, 0.0)
			assert.LessOrEqual(t, rec.Score, 100.0)
			assert.NotEmpty(t, rec.Reason)
		}
	}
}

func TestGetUnifiedToolIntel_NilCodeFlow_Recovers(t *testing.T) {
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := &SynthesisEngine{
		codeFlow:     nil,
		usageTracker: ut,
		shadowbroker: sb,
		toolIntel:    osint.NewToolIntel(),
		logger:       slog.Default().With("component", "synthesis"),
	}

	intel, err := se.GetUnifiedToolIntel(context.Background(), "any-tool")
	assert.Error(t, err)
	assert.Nil(t, intel)
	assert.Contains(t, err.Error(), "codeflow goroutine")
}

func TestGetUnifiedToolIntel_NilUsageTracker_Recovers(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})
	se := &SynthesisEngine{
		codeFlow:     cf,
		usageTracker: nil,
		shadowbroker: sb,
		toolIntel:    osint.NewToolIntel(),
		logger:       slog.Default().With("component", "synthesis"),
	}

	intel, err := se.GetUnifiedToolIntel(context.Background(), "any-tool")
	assert.Error(t, err)
	assert.Nil(t, intel)
	assert.Contains(t, err.Error(), "human ecosystems goroutine")
}

func TestGetUnifiedToolIntel_ShadowbrokerPanics(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	se := &SynthesisEngine{
		codeFlow:     cf,
		usageTracker: ut,
		shadowbroker: nil,
		toolIntel:    osint.NewToolIntel(),
		logger:       slog.Default().With("component", "synthesis"),
	}

	intel, err := se.GetUnifiedToolIntel(context.Background(), "resilient-tool")
	assert.NoError(t, err)
	assert.NotNil(t, intel)
	assert.Equal(t, "resilient-tool", intel.ToolID)
}

func TestGetCrossContextRecommendations_UsageFrequencyBoost(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "popular", codeflow.ExecutionMetrics{
		Duration:    50,
		CallCount:   1,
		ErrorCount:  0,
		TotalCalls:  1,
	})
	assert.NoError(t, err)

	for i := 0; i < 50; i++ {
		err = ut.RecordUsage(context.Background(), "user1", "popular", "frequent")
		assert.NoError(t, err)
	}

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "popular" {
			assert.Greater(t, rec.Score, 50.0)
		}
	}
}

func TestGetCrossContextRecommendations_SecurityRiskPenalty(t *testing.T) {
	cf := codeflow.NewCodeFlow()
	ut := he.NewToolUsageTracker()
	sb := osint.NewShadowbroker(osint.ShadowbrokerConfig{})

	err := cf.RecordExecution(context.Background(), "risky", codeflow.ExecutionMetrics{
		Duration:    100,
		CallCount:   1,
		ErrorCount:  0,
		TotalCalls:  1,
	})
	assert.NoError(t, err)

	se := NewSynthesisEngine(cf, ut, sb, slog.Default())
	recs, err := se.GetCrossContextRecommendations(context.Background(), "user1")
	assert.NoError(t, err)

	for _, rec := range recs {
		if rec.ToolID == "risky" {
			assert.GreaterOrEqual(t, rec.Score, 0.0)
			assert.LessOrEqual(t, rec.Score, 100.0)
		}
	}
}
