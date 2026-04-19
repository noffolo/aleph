package handler

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
)

func (c *CircuitBreakerClient) AnalyzeSentiment(ctx context.Context, req *connect.Request[nlp.AnalyzeSentimentRequest]) (*connect.Response[nlp.AnalyzeSentimentResponse], error) {
	return c.client.AnalyzeSentiment(ctx, req)
}

func (c *CircuitBreakerClient) RecordFeedback(ctx context.Context, req *connect.Request[nlp.RecordFeedbackRequest]) (*connect.Response[nlp.RecordFeedbackResponse], error) {
	return c.client.RecordFeedback(ctx, req)
}

type CircuitBreakerClient struct {
	client     nlpconnect.NLPServiceClient
	failureCnt int
	mu         sync.Mutex
	isDegraded bool
}

func NewCircuitBreakerClient(c nlpconnect.NLPServiceClient) *CircuitBreakerClient {
	return &CircuitBreakerClient{client: c}
}

func (c *CircuitBreakerClient) StreamPredictions(ctx context.Context, req *connect.Request[nlp.StreamPredictionsRequest]) (*connect.ServerStreamForClient[nlp.StreamPredictionsResponse], error) {
	c.mu.Lock()
	if c.isDegraded {
		c.mu.Unlock()
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("system in degraded mode: sidecar offline"))
	}
	c.mu.Unlock()

	stream, err := c.client.StreamPredictions(ctx, req)
	if err != nil {
		c.mu.Lock()
		c.failureCnt++
		if c.failureCnt >= 3 {
			c.isDegraded = true
		}
		c.mu.Unlock()
		return nil, err
	}

	c.mu.Lock()
	c.failureCnt = 0 // Reset su successo
	c.mu.Unlock()
	return stream, nil
}
// Metodi AnalyzeSentiment e RecordFeedback omessi per brevità (implementazione analoga)
