package predict

import (
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"sync"
)

type FactorManager struct {
	mu          sync.RWMutex
	predictions map[string]*nlp.AlephPrediction
}

func NewFactorManager() *FactorManager {
	return &FactorManager{
		predictions: make(map[string]*nlp.AlephPrediction),
	}
}

func (fm *FactorManager) UpdatePrediction(p *nlp.AlephPrediction) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.predictions[p.EntityId] = p
}
