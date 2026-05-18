package nlp_adapter

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/api/handler"
	nlpv1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/ingestion"
	"github.com/stretchr/testify/assert"
)

// mockNLPServiceClient implements nlpconnect.NLPServiceClient for testing
type mockNLPServiceClient struct {
	sentimentFn func(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error)
}

func (m *mockNLPServiceClient) AnalyzeSentiment(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error) {
	if m.sentimentFn != nil {
		return m.sentimentFn(ctx, req)
	}
	return &connect.Response[nlpv1.AnalyzeSentimentResponse]{
		Msg: &nlpv1.AnalyzeSentimentResponse{
			Score: 0.5,
			Label: "neutral",
		},
	}, nil
}

func (m *mockNLPServiceClient) StreamPredictions(ctx context.Context, req *connect.Request[nlpv1.StreamPredictionsRequest]) (*connect.ServerStreamForClient[nlpv1.StreamPredictionsResponse], error) {
	return nil, nil
}

func (m *mockNLPServiceClient) RecordFeedback(ctx context.Context, req *connect.Request[nlpv1.RecordFeedbackRequest]) (*connect.Response[nlpv1.RecordFeedbackResponse], error) {
	return &connect.Response[nlpv1.RecordFeedbackResponse]{Msg: &nlpv1.RecordFeedbackResponse{}}, nil
}

func TestNewAdapter(t *testing.T) {
	handler := handler.NewNLPHandler(nil, &mockNLPServiceClient{}, nil)
	adapter := &Adapter{NLPHandler: handler}
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.NLPHandler)
}

func TestNewAdapter_Nil(t *testing.T) {
	adapter := &Adapter{NLPHandler: nil}
	assert.NotNil(t, adapter)
	assert.Nil(t, adapter.NLPHandler)
}

func TestAdapter_InterfaceCompliance(t *testing.T) {
	// Compile-time check
	var _ ingestion.NLPAnalyzer = (*Adapter)(nil)
}

func TestAnalyzeSentiment(t *testing.T) {
	mockClient := &mockNLPServiceClient{
		sentimentFn: func(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error) {
			return &connect.Response[nlpv1.AnalyzeSentimentResponse]{
				Msg: &nlpv1.AnalyzeSentimentResponse{
					Score: 0.8,
					Label: "positive",
				},
			}, nil
		},
	}

	h := handler.NewNLPHandler(nil, mockClient, nil)
	adapter := &Adapter{NLPHandler: h}

	score, label, err := adapter.AnalyzeSentiment(context.Background(), "This is great!")
	assert.NoError(t, err)
	assert.Equal(t, float32(0.8), score)
	assert.Equal(t, "positive", label)
}

func TestAnalyzeSentiment_Negative(t *testing.T) {
	mockClient := &mockNLPServiceClient{
		sentimentFn: func(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error) {
			return &connect.Response[nlpv1.AnalyzeSentimentResponse]{
				Msg: &nlpv1.AnalyzeSentimentResponse{
					Score: 0.1,
					Label: "negative",
				},
			}, nil
		},
	}

	h := handler.NewNLPHandler(nil, mockClient, nil)
	adapter := &Adapter{NLPHandler: h}

	score, label, err := adapter.AnalyzeSentiment(context.Background(), "This is terrible!")
	assert.NoError(t, err)
	assert.Equal(t, float32(0.1), score)
	assert.Equal(t, "negative", label)
}

func TestAnalyzeSentiment_EmptyText(t *testing.T) {
	mockClient := &mockNLPServiceClient{
		sentimentFn: func(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error) {
			return &connect.Response[nlpv1.AnalyzeSentimentResponse]{
				Msg: &nlpv1.AnalyzeSentimentResponse{
					Score: 0.5,
					Label: "neutral",
				},
			}, nil
		},
	}

	h := handler.NewNLPHandler(nil, mockClient, nil)
	adapter := &Adapter{NLPHandler: h}

	score, _, err := adapter.AnalyzeSentiment(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, float32(0.5), score)
}

func TestAnalyzeSentiment_HandlerError(t *testing.T) {
	mockClient := &mockNLPServiceClient{
		sentimentFn: func(ctx context.Context, req *connect.Request[nlpv1.AnalyzeSentimentRequest]) (*connect.Response[nlpv1.AnalyzeSentimentResponse], error) {
			return nil, connect.NewError(connect.CodeUnavailable, nil)
		},
	}

	h := handler.NewNLPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), mockClient, nil)
	adapter := &Adapter{NLPHandler: h}

	score, label, err := adapter.AnalyzeSentiment(context.Background(), "test")
	assert.Error(t, err)
	assert.Equal(t, float32(0.0), score)
	assert.Equal(t, "", label)
}
