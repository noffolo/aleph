package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
	"github.com/ff3300/aleph-v2/internal/telemetry"
)

type BrierObserver interface {
	Observe(p *nlp.AlephPrediction, actual float32)
}

type NLPHandler struct {
	logger        *slog.Logger
	nlpClient     nlpconnect.NLPServiceClient
	breakerClient *CircuitBreakerClient
	httpClient    *http.Client
	brierMonitor  BrierObserver
}

func NewNLPHandler(logger *slog.Logger, rawClient nlpconnect.NLPServiceClient, httpClient *http.Client) *NLPHandler {
	breakerClient := NewCircuitBreakerClient(rawClient, logger)
	return &NLPHandler{logger: logger, nlpClient: breakerClient, breakerClient: breakerClient, httpClient: httpClient}
}

func (h *NLPHandler) SetBrierMonitor(bm BrierObserver) {
	h.brierMonitor = bm
}

func (h *NLPHandler) MarkHealthy() {
	h.breakerClient.MarkHealthy()
}

func (h *NLPHandler) MarkUnhealthy() {
	h.breakerClient.MarkUnhealthy()
}

func (h *NLPHandler) AnalyzeSentiment(
	ctx context.Context,
	req *connect.Request[nlp.AnalyzeSentimentRequest],
) (*connect.Response[nlp.AnalyzeSentimentResponse], error) {
	resp, err := h.nlpClient.AnalyzeSentiment(ctx, req)
	if err != nil {
		telemetry.RecordNLPRequest("sentiment", "error")
		h.logger.Warn("Sidecar sentiment analysis failed", "error", err)
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sentiment analysis unavailable: %w", err))
	}
	telemetry.RecordNLPRequest("sentiment", "success")
	return resp, nil
}

func (h *NLPHandler) StreamPredictions(
	ctx context.Context,
	req *connect.Request[nlp.StreamPredictionsRequest],
	stream *connect.ServerStream[nlp.StreamPredictionsResponse],
) error {
	pythonStream, err := h.nlpClient.StreamPredictions(ctx, req)
	if err != nil {
		telemetry.RecordNLPRequest("stream_predictions", "error")
		h.logger.Error("Failed to connect to Python sidecar", "error", err)
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("predictions unavailable: %w", err))
	}

		for pythonStream.Receive() {
		if err := stream.Send(pythonStream.Msg()); err != nil {
			telemetry.RecordNLPRequest("stream_predictions", "error")
			return fmt.Errorf("streamSend: %w", err)
		}
	}
	if err := pythonStream.Err(); err != nil {
		telemetry.RecordNLPRequest("stream_predictions", "error")
		h.logger.Error("Stream error from Python sidecar", "error", err)
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("prediction stream failed: %w", err))
	}
	telemetry.RecordNLPRequest("stream_predictions", "success")
	return nil
}

func (h *NLPHandler) RecordFeedback(
	ctx context.Context,
	req *connect.Request[nlp.RecordFeedbackRequest],
) (*connect.Response[nlp.RecordFeedbackResponse], error) {
	resp, err := h.nlpClient.RecordFeedback(ctx, req)
	if err != nil {
		telemetry.RecordNLPRequest("record_feedback", "error")
		h.logger.Warn("Sidecar feedback recording failed", "error", err)
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("feedback recording unavailable: %w", err))
	}
	if h.brierMonitor != nil && req.Msg != nil && req.Msg.IsCorrect {
		prediction := &nlp.AlephPrediction{
			EntityId:      req.Msg.EntityId,
			Probability:    0.5,
			ModelSource:    "feedback",
		}
		actual := float32(1.0)
		if !req.Msg.IsCorrect {
			actual = 0.0
		}
		h.brierMonitor.Observe(prediction, actual)
	}
	telemetry.RecordNLPRequest("record_feedback", "success")
	return resp, nil
}

// Close closes the underlying HTTP client connections.
func (h *NLPHandler) Close() error {
	if h.httpClient != nil {
		h.httpClient.CloseIdleConnections()
	}
	return nil
}
