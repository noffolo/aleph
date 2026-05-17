package handler

import (
	"log/slog"
	"testing"

	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/stretchr/testify/assert"
)

func TestNewNLPHandler(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	assert.NotNil(t, h)
	assert.NotNil(t, h.logger)
	assert.NotNil(t, h.breakerClient)
}

func TestNLPHandler_SetBrierMonitor(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	assert.Nil(t, h.brierMonitor)
}

func TestNLPHandler_MarkHealthy(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	h.MarkHealthy()
	assert.NotNil(t, h.breakerClient)
}

func TestNLPHandler_MarkUnhealthy(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	h.MarkUnhealthy()
	assert.NotNil(t, h.breakerClient)
}

func TestAlephPrediction_TypeExists(t *testing.T) {
	p := &nlp.AlephPrediction{
		EntityId:    "entity-1",
		Probability: 0.75,
	}
	assert.Equal(t, "entity-1", p.EntityId)
	assert.InDelta(t, float32(0.75), p.Probability, 0.001)
}

type brierObserverSpy struct {
	observed []*nlp.AlephPrediction
	actuals  []float32
}

func (b *brierObserverSpy) Observe(p *nlp.AlephPrediction, actual float32) {
	b.observed = append(b.observed, p)
	b.actuals = append(b.actuals, actual)
}

func TestNLPHandler_SetBrierMonitor_WithObserver(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	spy := &brierObserverSpy{}
	h.SetBrierMonitor(spy)
	assert.Equal(t, spy, h.brierMonitor)
	h.SetBrierMonitor(nil)
	assert.Nil(t, h.brierMonitor)
}

func TestNLPHandler_Close_Extended(t *testing.T) {
	h := NewNLPHandler(slog.Default(), nil, nil)
	assert.NoError(t, h.Close())
}

func TestBrierObserverSpy_Observes(t *testing.T) {
	spy := &brierObserverSpy{}
	pred := &nlp.AlephPrediction{EntityId: "e1", Probability: 0.5, ModelSource: "test"}
	spy.Observe(pred, 1.0)
	assert.Len(t, spy.observed, 1)
	assert.Equal(t, "e1", spy.observed[0].EntityId)
	assert.Equal(t, float32(1.0), spy.actuals[0])

	spy.Observe(pred, 0.0)
	assert.Len(t, spy.observed, 2)
	assert.Equal(t, float32(0.0), spy.actuals[1])
}
