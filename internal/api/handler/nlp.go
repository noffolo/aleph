package handler

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	nlp "github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1"
	"github.com/ff3300/aleph-v2/internal/api/proto/aleph/nlp/v1/nlpconnect"
)

type NLPHandler struct {
	logger    *slog.Logger
	nlpClient nlpconnect.NLPServiceClient
}

func NewNLPHandler(logger *slog.Logger, rawClient nlpconnect.NLPServiceClient) *NLPHandler {
	// Wrap the client with the Circuit Breaker
	breakerClient := NewCircuitBreakerClient(rawClient)
	return &NLPHandler{logger: logger, nlpClient: breakerClient}
}

func (h *NLPHandler) AnalyzeSentiment(
	ctx context.Context,
	req *connect.Request[nlp.AnalyzeSentimentRequest],
) (*connect.Response[nlp.AnalyzeSentimentResponse], error) {
	h.logger.Info("Analyzing sentiment", "text", req.Msg.Text)
	return connect.NewResponse(&nlp.AnalyzeSentimentResponse{
		Score: 0.9,
		Label: "positive",
	}), nil
}

func (h *NLPHandler) StreamPredictions(
	ctx context.Context,
	req *connect.Request[nlp.StreamPredictionsRequest],
	stream *connect.ServerStream[nlp.StreamPredictionsResponse],
) error {
	h.logger.Info("Proxying prediction stream to Python sidecar", "context_id", req.Msg.ContextId)

	// Call the Python sidecar via gRPC
	pythonStream, err := h.nlpClient.StreamPredictions(ctx, req)
	if err != nil {
		h.logger.Error("Failed to connect to Python sidecar", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("sidecar connection failed: %w", err))
	}

	// Stream responses from Python sidecar back to the client
	for pythonStream.Receive() {
		if err := stream.Send(pythonStream.Msg()); err != nil {
			return err
		}
	}
	if err := pythonStream.Err(); err != nil {
		h.logger.Error("Stream error from Python sidecar", "error", err)
	}
	return nil
}

func (h *NLPHandler) RecordFeedback(
	ctx context.Context,
	req *connect.Request[nlp.RecordFeedbackRequest],
) (*connect.Response[nlp.RecordFeedbackResponse], error) {
	h.logger.Info("Recording feedback", "entity_id", req.Msg.EntityId)
	return connect.NewResponse(&nlp.RecordFeedbackResponse{
		Success: true,
	}), nil
}
