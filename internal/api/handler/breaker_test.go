package handler

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNLPServiceClient struct {
	sentimentResponse *connect.Response[nlp.AnalyzeSentimentResponse]
	sentimentErr      error
	feedbackResponse  *connect.Response[nlp.RecordFeedbackResponse]
	feedbackErr       error
	streamResponse    *connect.ServerStreamForClient[nlp.StreamPredictionsResponse]
	streamErr         error
	callCount         atomic.Int32
}

func (m *mockNLPServiceClient) AnalyzeSentiment(ctx context.Context, req *connect.Request[nlp.AnalyzeSentimentRequest]) (*connect.Response[nlp.AnalyzeSentimentResponse], error) {
	m.callCount.Add(1)
	return m.sentimentResponse, m.sentimentErr
}

func (m *mockNLPServiceClient) RecordFeedback(ctx context.Context, req *connect.Request[nlp.RecordFeedbackRequest]) (*connect.Response[nlp.RecordFeedbackResponse], error) {
	m.callCount.Add(1)
	return m.feedbackResponse, m.feedbackErr
}

func (m *mockNLPServiceClient) StreamPredictions(ctx context.Context, req *connect.Request[nlp.StreamPredictionsRequest]) (*connect.ServerStreamForClient[nlp.StreamPredictionsResponse], error) {
	m.callCount.Add(1)
	return m.streamResponse, m.streamErr
}

func TestCircuitBreakerClient_StateTransitions(t *testing.T) {
	logger := slog.Default()
	mockClient := &mockNLPServiceClient{
		sentimentResponse: connect.NewResponse(&nlp.AnalyzeSentimentResponse{Score: 0.5, Label: "positive"}),
	}
	cb := NewCircuitBreakerClient(mockClient, logger)

	t.Run("initial state is closed", func(t *testing.T) {
		assert.Equal(t, int32(cbClosed), cb.state.Load())
		assert.Equal(t, int32(0), cb.failureCnt.Load())
	})

	t.Run("single failure increments counter", func(t *testing.T) {
		mockClient.sentimentErr = connect.NewError(connect.CodeUnavailable, nil)
		_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
		assert.Error(t, err)
		assert.Equal(t, int32(1), cb.failureCnt.Load())
		assert.Equal(t, int32(cbClosed), cb.state.Load())
	})

	t.Run("three failures transitions to open", func(t *testing.T) {
		cb.failureCnt.Store(0)
		for i := 0; i < 3; i++ {
			_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
			assert.Error(t, err)
		}
		assert.Equal(t, int32(3), cb.failureCnt.Load())
		state := cb.state.Load()
		assert.Equal(t, int32(cbOpen), state)
	})

	t.Run("open state rejects requests", func(t *testing.T) {
		mockClient.sentimentErr = nil
		mockClient.callCount.Store(0)
		_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
		assert.Error(t, err)
		assert.Equal(t, int32(0), mockClient.callCount.Load())
	})

	t.Run("open transitions to half-open after 30 seconds", func(t *testing.T) {
		cb.state.Store(cbOpen)
		cb.lastFail.Store(time.Now().Add(-35 * time.Second).Unix())
		state := cb.currentState()
		assert.Equal(t, int32(cbHalfOpen), state)
	})

	t.Run("half-open allows probing request", func(t *testing.T) {
		cb.state.Store(cbHalfOpen)
		mockClient.sentimentErr = nil
		mockClient.callCount.Store(0)
		resp, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
		require.NoError(t, err)
		assert.Equal(t, float32(0.5), resp.Msg.Score)
		assert.Equal(t, int32(1), mockClient.callCount.Load())
		assert.Equal(t, int32(cbClosed), cb.state.Load())
		assert.Equal(t, int32(0), cb.failureCnt.Load())
	})

	t.Run("half-open failure increments counter", func(t *testing.T) {
		cb.state.Store(cbHalfOpen)
		cb.failureCnt.Store(0)
		mockClient.sentimentErr = connect.NewError(connect.CodeUnavailable, nil)
		_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
		assert.Error(t, err)
		state := cb.state.Load()
		assert.Equal(t, int32(cbHalfOpen), state)
		assert.Equal(t, int32(1), cb.failureCnt.Load())
	})

	t.Run("MarkHealthy resets state", func(t *testing.T) {
		cb.state.Store(cbOpen)
		cb.failureCnt.Store(5)
		cb.MarkHealthy()
		assert.Equal(t, int32(cbClosed), cb.state.Load())
		assert.Equal(t, int32(0), cb.failureCnt.Load())
	})
}

func TestCircuitBreakerClient_NilClient(t *testing.T) {
	logger := slog.Default()
	cb := NewCircuitBreakerClient(nil, logger)
	cb.state.Store(cbClosed)

	_, err := cb.AnalyzeSentiment(context.Background(), connect.NewRequest(&nlp.AnalyzeSentimentRequest{Text: "test"}))
	assert.Error(t, err)
	// Single failure with nil client increments counter but doesn't transition to open
	assert.Equal(t, int32(1), cb.failureCnt.Load())
	// State remains closed until 3 failures
	assert.Equal(t, int32(cbClosed), cb.state.Load())
}

func TestCircuitBreakerClient_CurrentState(t *testing.T) {
	logger := slog.Default()
	mockClient := &mockNLPServiceClient{}
	cb := NewCircuitBreakerClient(mockClient, logger)

	t.Run("open state with recent failure stays open", func(t *testing.T) {
		cb.state.Store(cbOpen)
		cb.lastFail.Store(time.Now().Unix())
		state := cb.currentState()
		assert.Equal(t, int32(cbOpen), state)
	})

	t.Run("closed state remains closed", func(t *testing.T) {
		cb.state.Store(cbClosed)
		state := cb.currentState()
		assert.Equal(t, int32(cbClosed), state)
	})

	t.Run("half-open state remains half-open", func(t *testing.T) {
		cb.state.Store(cbHalfOpen)
		state := cb.currentState()
		assert.Equal(t, int32(cbHalfOpen), state)
	})
}

func TestCircuitBreakerClient_RecordFeedback(t *testing.T) {
	logger := slog.Default()
	mockClient := &mockNLPServiceClient{
		feedbackResponse: connect.NewResponse(&nlp.RecordFeedbackResponse{Success: true}),
	}
	cb := NewCircuitBreakerClient(mockClient, logger)

	t.Run("successful feedback resets failure count", func(t *testing.T) {
		cb.failureCnt.Store(2)
		resp, err := cb.RecordFeedback(context.Background(), connect.NewRequest(&nlp.RecordFeedbackRequest{}))
		require.NoError(t, err)
		assert.True(t, resp.Msg.Success)
		assert.Equal(t, int32(0), cb.failureCnt.Load())
		assert.Equal(t, int32(cbClosed), cb.state.Load())
	})

	t.Run("failed feedback increments failure count", func(t *testing.T) {
		mockClient.feedbackErr = connect.NewError(connect.CodeUnavailable, nil)
		_, err := cb.RecordFeedback(context.Background(), connect.NewRequest(&nlp.RecordFeedbackRequest{}))
		assert.Error(t, err)
		assert.Equal(t, int32(1), cb.failureCnt.Load())
	})
}

func TestCircuitBreakerClient_StreamPredictions(t *testing.T) {
	logger := slog.Default()
	mockClient := &mockNLPServiceClient{}
	cb := NewCircuitBreakerClient(mockClient, logger)

	t.Run("stream creation with nil client fails", func(t *testing.T) {
		cb := NewCircuitBreakerClient(nil, logger)
		_, err := cb.StreamPredictions(context.Background(), connect.NewRequest(&nlp.StreamPredictionsRequest{}))
		assert.Error(t, err)
	})

	t.Run("open state rejects stream creation", func(t *testing.T) {
		cb.state.Store(cbOpen)
		cb.lastFail.Store(time.Now().Unix()) // Recent failure to keep state open
		_, err := cb.StreamPredictions(context.Background(), connect.NewRequest(&nlp.StreamPredictionsRequest{}))
		assert.Error(t, err)
	})
}