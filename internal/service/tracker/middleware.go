package tracker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/ff3300/aleph-v2/internal/middleware"
)

type trackingInterceptor struct {
	tracker Tracker
}

// NewTrackingInterceptor creates a ConnectRPC interceptor that records
// tool usage after each tool execution call.
func NewTrackingInterceptor(t Tracker) connect.Interceptor {
	return &trackingInterceptor{tracker: t}
}

// WrapUnary implements connect.Interceptor — records tool execution metrics.
func (i *trackingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		duration := time.Since(start)

		projectID := middleware.ProjectIDFromContext(ctx)
		if projectID == "" {
			return resp, err
		}

		toolName := extractToolName(req.Spec())
		if toolName == "" {
			return resp, err
		}

		usage := ToolUsage{
			ID:         generateID(),
			UserID:     projectID,
			ProjectID:  projectID,
			ToolName:   toolName,
			DurationMs: duration.Milliseconds(),
			Success:    err == nil,
			Timestamp:  start,
		}
		if err != nil {
			msg := err.Error()
			if len(msg) > 500 {
				msg = msg[:500]
			}
			usage.ErrorMsg = msg
		}

		go i.tracker.Record(context.Background(), usage)

		return resp, err
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for tracking).
func (i *trackingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor (no-op for tracking).
func (i *trackingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	}
}

// extractToolName derives a short tool name from a ConnectRPC procedure spec.
// e.g. "/aleph.v1.ToolService/ExecuteTool" -> "tool.execute"
// e.g. "/aleph.v1.QueryService/Chat" -> "query.chat"
func extractToolName(spec connect.Spec) string {
	procedure := spec.Procedure
	if procedure == "" {
		return ""
	}
	procedure = strings.TrimPrefix(procedure, "/")
	parts := strings.Split(procedure, "/")
	if len(parts) < 2 {
		return procedure
	}
	servicePart := parts[0]
	methodPart := parts[1]

	serviceParts := strings.Split(servicePart, ".")
	serviceName := serviceParts[len(serviceParts)-1]

	return fmt.Sprintf("%s.%s", strings.ToLower(serviceName), lowerFirst(methodPart))
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
