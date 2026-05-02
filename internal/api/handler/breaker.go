package handler

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/errors"
)

const (
	cbClosed   int32 = 0
	cbHalfOpen int32 = 1
	cbOpen     int32 = 2
)

type CircuitBreakerClient struct {
	client     nlpconnect.NLPServiceClient
	failureCnt atomic.Int32
	mu         sync.Mutex
	state      atomic.Int32
	lastFail   atomic.Int64
	logger     *slog.Logger
}

func NewCircuitBreakerClient(c nlpconnect.NLPServiceClient, logger *slog.Logger) *CircuitBreakerClient {
	return &CircuitBreakerClient{client: c, logger: logger}
}

func (c *CircuitBreakerClient) currentState() int32 {
	s := c.state.Load()
	if s == cbOpen {
		elapsed := time.Since(time.Unix(c.lastFail.Load(), 0))
		if elapsed >= 30*time.Second {
			if c.state.CompareAndSwap(cbOpen, cbHalfOpen) {
				c.logger.Info("circuit breaker: open → half-open, probing")
			}
			return cbHalfOpen
		}
	}
	return s
}

func (c *CircuitBreakerClient) AnalyzeSentiment(ctx context.Context, req *connect.Request[nlp.AnalyzeSentimentRequest]) (*connect.Response[nlp.AnalyzeSentimentResponse], error) {
	state := c.currentState()
	if state == cbOpen || c.client == nil {
		c.recordFailure()
		return nil, connect.NewError(connect.CodeUnavailable, errors.NewAPIErrorWithMeta(
			errors.ErrUnavailable, "system in degraded mode: sidecar offline", nil,
			"nlp", "health", true, 30000,
		))
	}

	resp, err := c.client.AnalyzeSentiment(ctx, req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}
	c.recordSuccess()
	return resp, nil
}

func (c *CircuitBreakerClient) RecordFeedback(ctx context.Context, req *connect.Request[nlp.RecordFeedbackRequest]) (*connect.Response[nlp.RecordFeedbackResponse], error) {
	state := c.currentState()
	if state == cbOpen || c.client == nil {
		c.recordFailure()
		return nil, connect.NewError(connect.CodeUnavailable, errors.NewAPIErrorWithMeta(
			errors.ErrUnavailable, "system in degraded mode: sidecar offline", nil,
			"nlp", "health", true, 30000,
		))
	}

	resp, err := c.client.RecordFeedback(ctx, req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}
	c.recordSuccess()
	return resp, nil
}

func (c *CircuitBreakerClient) StreamPredictions(ctx context.Context, req *connect.Request[nlp.StreamPredictionsRequest]) (*connect.ServerStreamForClient[nlp.StreamPredictionsResponse], error) {
	state := c.currentState()
	if state == cbOpen || c.client == nil {
		c.recordFailure()
		return nil, connect.NewError(connect.CodeUnavailable, errors.NewAPIErrorWithMeta(
			errors.ErrUnavailable, "system in degraded mode: sidecar offline", nil,
			"nlp", "health", true, 30000,
		))
	}

	stream, err := c.client.StreamPredictions(ctx, req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}

	return stream, nil
}

func (c *CircuitBreakerClient) MarkHealthy() {
	c.state.Store(cbClosed)
	c.failureCnt.Store(0)
}

// MarkUnhealthy forces the circuit breaker to open (degraded mode).
func (c *CircuitBreakerClient) MarkUnhealthy() {
	c.lastFail.Store(time.Now().Unix())
	c.state.Store(cbOpen)
}

func (c *CircuitBreakerClient) recordFailure() {
	c.lastFail.Store(time.Now().Unix())
	count := c.failureCnt.Add(1)
	if count >= 3 {
		if c.state.CompareAndSwap(cbClosed, cbOpen) || c.state.CompareAndSwap(cbHalfOpen, cbOpen) {
			c.logger.Warn("circuit breaker: → open, too many failures", "failure_count", count)
		}
	}
}

func (c *CircuitBreakerClient) recordSuccess() {
	c.state.Store(cbClosed)
	c.failureCnt.Store(0)
}
