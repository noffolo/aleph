package predict

import (
	"testing"

	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/stretchr/testify/assert"
	"log/slog"
)

func TestNewBrierMonitor(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())
	assert.NotNil(t, bm)
	assert.Equal(t, float64(0), bm.GetAvgBrierScore())
}

func TestBrierMonitor_Observe(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())

	p := &nlp.AlephPrediction{
		EntityId:    "tool_1",
		Probability: 0.8,
		ModelSource: "test_model",
	}

	// Observe with actual=1.0 (correct prediction, diff = 0.2)
	// Score = 0.2^2 = 0.04
	bm.Observe(p, 1.0)

	score, ok := bm.GetBrierScore("tool_1")
	assert.True(t, ok)
	assert.InDelta(t, 0.04, score, 0.0001)
}

func TestBrierMonitor_ObserveWrongPrediction(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())

	p := &nlp.AlephPrediction{
		EntityId:    "tool_2",
		Probability: 0.9,
		ModelSource: "model_a",
	}

	// Observe with actual=0.0 (completely wrong, diff = 0.9)
	// Score = 0.9^2 = 0.81
	bm.Observe(p, 0.0)

	score, ok := bm.GetBrierScore("tool_2")
	assert.True(t, ok)
	assert.InDelta(t, 0.81, score, 0.0001)
}

func TestBrierMonitor_AvgScore(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())

	bm.Observe(&nlp.AlephPrediction{EntityId: "a", Probability: 0.8}, 1.0) // 0.04
	bm.Observe(&nlp.AlephPrediction{EntityId: "b", Probability: 0.6}, 0.0) // 0.36

	avg := bm.GetAvgBrierScore()
	assert.InDelta(t, 0.20, avg, 0.0001) // (0.04 + 0.36) / 2
}

func TestBrierMonitor_GetBrierScoreNotFound(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())
	_, ok := bm.GetBrierScore("nonexistent")
	assert.False(t, ok)
}

func TestBrierMonitor_EmptyAvg(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())
	assert.Equal(t, float64(0), bm.GetAvgBrierScore())
}

func TestBrierMonitor_MultipleObservations(t *testing.T) {
	bm := NewBrierMonitor(slog.Default())

	// Observe same entity twice - last score wins
	bm.Observe(&nlp.AlephPrediction{EntityId: "x", Probability: 0.5}, 1.0) // 0.25
	bm.Observe(&nlp.AlephPrediction{EntityId: "x", Probability: 0.9}, 1.0) // 0.01

	score, ok := bm.GetBrierScore("x")
	assert.True(t, ok)
	assert.InDelta(t, 0.01, score, 0.0001)
}

func TestNewFactorManager(t *testing.T) {
	fm := NewFactorManager()
	assert.NotNil(t, fm)
}

func TestFactorManager_UpdatePrediction(t *testing.T) {
	fm := NewFactorManager()
	p := &nlp.AlephPrediction{
		EntityId:    "tool_1",
		Probability: 0.75,
		ModelSource: "model_v1",
	}
	fm.UpdatePrediction(p)
	// No return value, just ensure no panic
}

func TestFactorManager_UpdatePredictionOverride(t *testing.T) {
	fm := NewFactorManager()
	fm.UpdatePrediction(&nlp.AlephPrediction{EntityId: "tool_1", Probability: 0.5})
	fm.UpdatePrediction(&nlp.AlephPrediction{EntityId: "tool_1", Probability: 0.9})
	// No panic on override
}
