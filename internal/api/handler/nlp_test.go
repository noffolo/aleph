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
}

func TestNLPHandler_MarkUnhealthy(t *testing.T) {
	var client nlpconnect.NLPServiceClient
	h := NewNLPHandler(slog.Default(), client, nil)
	h.MarkUnhealthy()
}

func TestAlephPrediction_TypeExists(t *testing.T) {
	p := &nlp.AlephPrediction{
		EntityId:    "entity-1",
		Probability: 0.75,
	}
	assert.Equal(t, "entity-1", p.EntityId)
	assert.InDelta(t, float32(0.75), p.Probability, 0.001)
}
