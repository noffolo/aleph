package tracker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRequest implements connect.AnyRequest by embedding *connect.Request.
type testRequest struct {
	*connect.Request[struct{}]
	procedure string
}

func newTestRequest(procedure string) connect.AnyRequest {
	return &testRequest{
		Request:   connect.NewRequest(&struct{}{}),
		procedure: procedure,
	}
}

func (m *testRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: m.procedure}
}

// mockTracker captures calls to Record for verification.
type mockTracker struct {
	recorded chan ToolUsage
	err      error
}

func newMockTracker() *mockTracker {
	return &mockTracker{recorded: make(chan ToolUsage, 10)}
}

func (m *mockTracker) Record(ctx context.Context, usage ToolUsage) error {
	select {
	case m.recorded <- usage:
	default:
	}
	return m.err
}

func (m *mockTracker) MostUsedTools(ctx context.Context, userID string, limit int, since time.Time) ([]ToolUsageStat, error) {
	return nil, nil
}

func (m *mockTracker) ToolSequences(ctx context.Context, userID string, limit int) ([][]string, error) {
	return nil, nil
}

// ── NewTrackingInterceptor ──────────────────────────────────────────────────

func TestNewTrackingInterceptor_Happy(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)
	assert.NotNil(t, interceptor, "NewTrackingInterceptor should return non-nil interceptor")
}

func TestNewTrackingInterceptor_EdgeNilTracker(t *testing.T) {
	interceptor := NewTrackingInterceptor(nil)
	assert.NotNil(t, interceptor, "should still return a non-nil interceptor even with nil tracker")

	ti, ok := interceptor.(*trackingInterceptor)
	require.True(t, ok, "returned value should be *trackingInterceptor")
	assert.Nil(t, ti.tracker, "tracker field should be nil when passed nil")
}

func TestNewTrackingInterceptor_ErrorImplementsInterface(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)
	assert.Implements(t, (*connect.Interceptor)(nil), interceptor,
		"must implement connect.Interceptor interface")
}

// ── WrapUnary ──────────────────────────────────────────────────────────────

func TestWrapUnary_HappyNoProjectIDPassthrough(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	called := false
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return connect.NewResponse(&struct{}{}), nil
	}

	wrapped := interceptor.WrapUnary(next)
	req := newTestRequest("/aleph.v1.ToolService/ExecuteTool")
	resp, err := wrapped(context.Background(), req)

	assert.NoError(t, err, "handler should not error")
	assert.True(t, called, "next handler must be invoked")
	assert.NotNil(t, resp, "response should be non-nil")
}

func TestWrapUnary_EdgeHandlerReturnsError(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	expectedErr := errors.New("handler failure")
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, expectedErr
	}

	wrapped := interceptor.WrapUnary(next)
	req := newTestRequest("/aleph.v1.ToolService/ExecuteTool")
	resp, err := wrapped(context.Background(), req)

	assert.Error(t, err, "error from handler must propagate")
	assert.Equal(t, expectedErr, err, "error must be the exact error from next handler")
	assert.Nil(t, resp, "response should be nil when handler errors")
}

func TestWrapUnary_ErrorPropagatesCorrectly(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, fmt.Errorf("wrapped: %w", errors.New("inner"))
	}

	wrapped := interceptor.WrapUnary(next)
	req := newTestRequest("/aleph.v1.ToolService/ExecuteTool")
	_, err := wrapped(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inner", "original error message must be preserved")
}

// ── WrapStreamingClient ────────────────────────────────────────────────────

func TestWrapStreamingClient_HappyPassthrough(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	called := false
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		called = true
		return nil
	}

	wrapped := interceptor.WrapStreamingClient(next)
	conn := wrapped(context.Background(), connect.Spec{Procedure: "test"})
	assert.True(t, called, "next streaming client func must be called")
	assert.Nil(t, conn, "conn result should pass through from next")
}

func TestWrapStreamingClient_EdgeIdentityFunction(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	}

	wrapped := interceptor.WrapStreamingClient(next)
	assert.NotNil(t, wrapped, "wrapped function must be non-nil")
}

func TestWrapStreamingClient_ErrorNilNextReturnsNil(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	wrapped := interceptor.WrapStreamingClient(nil)
	assert.Nil(t, wrapped, "WrapStreamingClient with nil next must return nil")
}

// ── WrapStreamingHandler ───────────────────────────────────────────────────

func TestWrapStreamingHandler_HappyPassthrough(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	called := false
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		called = true
		return nil
	}

	wrapped := interceptor.WrapStreamingHandler(next)
	err := wrapped(context.Background(), nil)

	assert.NoError(t, err, "handler should return no error")
	assert.True(t, called, "next handler must be invoked")
}

func TestWrapStreamingHandler_EdgeNilConnPassthrough(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	connReceived := false
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		connReceived = conn == nil
		return nil
	}

	wrapped := interceptor.WrapStreamingHandler(next)
	err := wrapped(context.Background(), nil)

	assert.NoError(t, err)
	assert.True(t, connReceived, "nil conn must be passed through to next handler")
}

func TestWrapStreamingHandler_ErrorPropagation(t *testing.T) {
	mock := newMockTracker()
	interceptor := NewTrackingInterceptor(mock)

	expectedErr := errors.New("streaming handler failure")
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return expectedErr
	}

	wrapped := interceptor.WrapStreamingHandler(next)
	err := wrapped(context.Background(), nil)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err, "error from handler must propagate unchanged")
}

// ── extractToolName ────────────────────────────────────────────────────────

func TestExtractToolName_HappyMultiDotService(t *testing.T) {
	spec := connect.Spec{Procedure: "/com.example.internal.v2.AnalyticsService/GenerateReport"}
	got := extractToolName(spec)
	assert.Equal(t, "analyticsservice.generateReport", got)
}

func TestExtractToolName_EdgeTrailingSlash(t *testing.T) {
	spec := connect.Spec{Procedure: "/aleph.v1.ToolService/trailing/"}
	got := extractToolName(spec)
	assert.Equal(t, "toolservice.trailing", got)
}

func TestExtractToolName_ErrorEmptySpec(t *testing.T) {
	spec := connect.Spec{Procedure: ""}
	got := extractToolName(spec)
	assert.Equal(t, "", got)
}

// ── lowerFirst ─────────────────────────────────────────────────────────────

func TestLowerFirst_HappyStandard(t *testing.T) {
	assert.Equal(t, "hello", lowerFirst("Hello"))
	assert.Equal(t, "world", lowerFirst("World"))
	assert.Equal(t, "chat", lowerFirst("Chat"))
}

func TestLowerFirst_EdgeSingleUpperChar(t *testing.T) {
	assert.Equal(t, "a", lowerFirst("A"))
	assert.Equal(t, "z", lowerFirst("Z"))
}

func TestLowerFirst_ErrorNumericStarts(t *testing.T) {
	assert.Equal(t, "123abc", lowerFirst("123abc"))
	assert.Equal(t, "9thing", lowerFirst("9thing"))
}
