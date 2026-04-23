package predict

import (
	"log/slog"
	"sync"

	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
)

type BrierMonitor struct {
	logger *slog.Logger
	mu     sync.Mutex
	scores map[string]float64
}

func NewBrierMonitor(logger *slog.Logger) *BrierMonitor {
	return &BrierMonitor{
		logger: logger,
		scores: make(map[string]float64),
	}
}

// Observe evaluates prediction accuracy based on actual outcome.
// Score = (probability - actual)² per Brier score definition.
func (bm *BrierMonitor) Observe(p *nlp.AlephPrediction, actual float32) {
	diff := p.Probability - actual
	score := diff * diff
	bm.mu.Lock()
	bm.scores[p.EntityId] = float64(score)
	bm.mu.Unlock()
	bm.logger.Info("Brier Score Update",
		"entity_id", p.EntityId,
		"score", score,
		"model", p.ModelSource)
}

// GetAvgBrierScore returns the average Brier score across all observed entities.
func (bm *BrierMonitor) GetAvgBrierScore() float64 {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	if len(bm.scores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range bm.scores {
		sum += s
	}
	return sum / float64(len(bm.scores))
}

// GetBrierScore returns the Brier score for a specific entity.
func (bm *BrierMonitor) GetBrierScore(entityID string) (float64, bool) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	s, ok := bm.scores[entityID]
	return s, ok
}