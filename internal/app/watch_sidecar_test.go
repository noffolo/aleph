package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ff3300/aleph-v2/internal/api/handler"
)

type mockHealthClient struct {
	grpc_health_v1.HealthClient
	status grpc_health_v1.HealthCheckResponse_ServingStatus
	err    error
}

func (m *mockHealthClient) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &grpc_health_v1.HealthCheckResponse{Status: m.status}, nil
}

func newTestNLPHandler() *handler.NLPHandler {
	return handler.NewNLPHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
}

func TestCheckSidecarOnce(t *testing.T) {
	tests := []struct {
		name               string
		mockErr            error
		mockStatus         grpc_health_v1.HealthCheckResponse_ServingStatus
		nlpHandler         *handler.NLPHandler
		consecutiveErr     bool
		restartCount       int
		restartStart       time.Time
		wantContinue       bool
		wantConsecutiveErr bool
		wantRestartCount   int
	}{
		{
			name:               "health OK — SERVING status",
			mockErr:            nil,
			mockStatus:         grpc_health_v1.HealthCheckResponse_SERVING,
			nlpHandler:         newTestNLPHandler(),
			consecutiveErr:     false,
			restartCount:       0,
			wantContinue:       true,
			wantConsecutiveErr: false,
			wantRestartCount:   0,
		},
		{
			name:               "error — first failure starts counter",
			mockErr:            errors.New("connection refused"),
			mockStatus:         grpc_health_v1.HealthCheckResponse_SERVING,
			nlpHandler:         newTestNLPHandler(),
			consecutiveErr:     false,
			restartCount:       0,
			wantContinue:       true,
			wantConsecutiveErr: true,
			wantRestartCount:   1,
		},
		{
			name:               "error — consecutive failure increments counter",
			mockErr:            errors.New("timeout"),
			mockStatus:         grpc_health_v1.HealthCheckResponse_SERVING,
			nlpHandler:         newTestNLPHandler(),
			consecutiveErr:     true,
			restartCount:       2,
			wantContinue:       true,
			wantConsecutiveErr: true,
			wantRestartCount:   3,
		},
		{
			name:               "max restarts exceeded — stops loop",
			mockErr:            errors.New("connection refused"),
			mockStatus:         grpc_health_v1.HealthCheckResponse_SERVING,
			nlpHandler:         newTestNLPHandler(),
			consecutiveErr:     true,
			restartCount:       4, // > sidecarMaxRestarts (3)
			restartStart:       time.Now(),
			wantContinue:       false,
			wantConsecutiveErr: true,
			wantRestartCount:   5, // incremented before the guard check
		},
		{
			name:               "nil NLPHandler — no panic, resets counters",
			mockErr:            nil,
			mockStatus:         grpc_health_v1.HealthCheckResponse_SERVING,
			nlpHandler:         nil,
			consecutiveErr:     true,
			restartCount:       5,
			wantContinue:       true,
			wantConsecutiveErr: false,
			wantRestartCount:   0,
		},
		{
			name:               "non-SERVING status treated as healthy",
			mockErr:            nil,
			mockStatus:         grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			nlpHandler:         newTestNLPHandler(),
			consecutiveErr:     true,
			restartCount:       2,
			wantContinue:       true,
			wantConsecutiveErr: false,
			wantRestartCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockHealthClient{status: tt.mockStatus, err: tt.mockErr}
			a := &AlephApp{ctx: context.Background()}

			consecutiveErr := tt.consecutiveErr
			restartCount := tt.restartCount
			restartStart := tt.restartStart

			got := a.checkSidecarOnce(mock, tt.nlpHandler, &consecutiveErr, &restartCount, &restartStart)

			assert.Equal(t, tt.wantContinue, got, "continue")
			assert.Equal(t, tt.wantConsecutiveErr, consecutiveErr, "consecutiveErr")
			assert.Equal(t, tt.wantRestartCount, restartCount, "restartCount")
		})
	}
}

func TestCheckSidecarOnce_NilNLPHandler(t *testing.T) {
	mock := &mockHealthClient{status: grpc_health_v1.HealthCheckResponse_SERVING}
	a := &AlephApp{ctx: context.Background()}

	consecutiveErr := true
	restartCount := 5
	var restartStart time.Time

	got := a.checkSidecarOnce(mock, nil, &consecutiveErr, &restartCount, &restartStart)

	require.True(t, got, "should continue when nlpHandler is nil")
	assert.False(t, consecutiveErr, "consecutiveErr should be reset")
	assert.Equal(t, 0, restartCount, "restartCount should be reset")
}

func TestCheckSidecarOnce_MaxRestartsExceeded(t *testing.T) {
	mock := &mockHealthClient{err: errors.New("connection refused")}
	a := &AlephApp{ctx: context.Background()}
	nlpHandler := newTestNLPHandler()

	consecutiveErr := true
	restartCount := 4 // > sidecarMaxRestarts (3)
	restartStart := time.Now()

	got := a.checkSidecarOnce(mock, nlpHandler, &consecutiveErr, &restartCount, &restartStart)

	assert.False(t, got, "should stop loop when max restarts exceeded")
	assert.True(t, consecutiveErr)
	assert.Equal(t, 5, restartCount, "restartCount should be incremented before guard check")
}
