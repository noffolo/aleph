package predict

import (
	"log/slog"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
)

type BrierMonitor struct {
	logger *slog.Logger
}

func NewBrierMonitor(logger *slog.Logger) *BrierMonitor {
	return &BrierMonitor{logger: logger}
}

// Observe valuta la precisione di una predizione basandosi sul risultato reale (outcome)
func (bm *BrierMonitor) Observe(p *nlp.AlephPrediction, actual float32) {
	// Brier Score = (predizione - reale)^2
	diff := p.Probability - actual
	score := diff * diff
	bm.logger.Info("Brier Score Update", 
		"entity_id", p.EntityId, 
		"score", score, 
		"model", p.ModelSource)
}
