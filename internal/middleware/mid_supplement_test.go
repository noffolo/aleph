package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
)

func TestSubsystemInterceptor_WrapUnary_Coverage(t *testing.T) {
	i := NewSubsystemInterceptor()
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}
	wrapped := i.WrapUnary(next)
	req := newMockRequest("/aleph.v1.QueryService/ExecuteQuery")
	_, _ = wrapped(context.Background(), req)
}

func TestSubsystemInterceptor_WrapUnary_EmptySubsystem(t *testing.T) {
	i := NewSubsystemInterceptor()
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}
	wrapped := i.WrapUnary(next)
	req := newMockRequest("/no.prefix.match/Method")
	_, _ = wrapped(context.Background(), req)
}

func TestSubsystemInterceptor_WrapStreamingHandler_Coverage(t *testing.T) {
	i := NewSubsystemInterceptor()
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := i.WrapStreamingHandler(next)
	_ = wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.NLPService/AnalyzeSentiment"})
}

func TestBulkheadInterceptor_WrapStreamingHandler(t *testing.T) {
	b := NewBulkheadInterceptor(&BulkheadConfig{
		QueryPoolSize:     5,
		IngestionPoolSize: 2,
		ChatPoolSize:      3,
	})
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := b.WrapStreamingHandler(next)
	err := wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.QueryService/ExecuteQuery"})
	assert.NoError(t, err)
}

func TestBulkheadInterceptor_WrapStreamingHandler_Overloaded(t *testing.T) {
	b := NewBulkheadInterceptor(&BulkheadConfig{
		QueryPoolSize: 1,
	})
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := b.WrapStreamingHandler(next)
	// Acquire the only slot
	_ = b.queryPool.TryAcquire(1)
	// Next request should fail
	err := wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.QueryService/ExecuteQuery"})
	assert.Error(t, err)
}

func TestBulkheadInterceptor_WrapStreamingHandler_NoPool(t *testing.T) {
	b := NewBulkheadInterceptor(nil)
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := b.WrapStreamingHandler(next)
	err := wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/no.bulkhead.Service/Method"})
	assert.NoError(t, err)
}

func TestBulkheadConfigForDomain_Default(t *testing.T) {
	b := NewBulkheadInterceptor(&BulkheadConfig{
		QueryPoolSize:     50,
		IngestionPoolSize: 10,
		ChatPoolSize:      20,
	})
	assert.Equal(t, int64(0), b.configForDomain("nonexistent"))
}

func TestCircuitBreaker_Reset_Coverage(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)
	failFn := func() error { return assert.AnError }
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)
	assert.Equal(t, StateOpen, cb.State())
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.failureCount)
}

func TestNewCircuitBreakerInterceptor(t *testing.T) {
	interceptor := NewCircuitBreakerInterceptor(3, 5*time.Second)
	assert.NotNil(t, interceptor)
	assert.NotNil(t, interceptor.breaker)
}

func TestCircuitBreakerInterceptor_WrapUnary(t *testing.T) {
	interceptor := NewCircuitBreakerInterceptor(5, 10*time.Second)
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}
	wrapped := interceptor.WrapUnary(next)
	req := newMockRequest("/aleph.v1.QueryService/ExecuteQuery")
	_, err := wrapped(context.Background(), req)
	assert.NoError(t, err)
}

func TestCircuitBreakerInterceptor_WrapStreamingHandler(t *testing.T) {
	interceptor := NewCircuitBreakerInterceptor(5, 10*time.Second)
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := interceptor.WrapStreamingHandler(next)
	err := wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/test"})
	assert.NoError(t, err)
}

func TestMultiKeyLimiter_Close(t *testing.T) {
	cfg := DefaultRateLimitConfig
	rl := newMultiKeyLimiter(cfg)
	rl.Close()
}

func TestAuthRateLimiter_Store(t *testing.T) {
	rl := NewAuthRateLimiter(nil, DefaultAuthRateLimitConfig)
	defer rl.Close()
	assert.NotNil(t, rl.Store())
}

func TestAuthRateLimiter_Close(t *testing.T) {
	rl := NewAuthRateLimiter(nil, DefaultAuthRateLimitConfig)
	rl.Close()
}

func TestAuthRateLimiter_CheckHTTP(t *testing.T) {
	rl := NewAuthRateLimiter(nil, DefaultAuthRateLimitConfig)
	defer rl.Close()

	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	allowed, _ := rl.CheckHTTP(req, "session_create")
	assert.True(t, allowed)
}

func TestRoleFromEnv(t *testing.T) {
	origFn := roleFromEnvFn
	defer func() { roleFromEnvFn = origFn }()

	roleFromEnvFn = func(apiKey string) Role {
		return RoleUser
	}
	assert.Equal(t, RoleUser, roleFromEnv("any-key"))
}

func TestJWTFromCookie_Coverage(t *testing.T) {
	h := make(map[string][]string)
	assert.Empty(t, jwtFromCookie(h))

	h["Cookie"] = []string{"aleph_jwt=my-jwt-token"}
	assert.Equal(t, "my-jwt-token", jwtFromCookie(h))

	h["Cookie"] = []string{"other_cookie=value; aleph_jwt=another-token"}
	assert.Equal(t, "another-token", jwtFromCookie(h))
}

func TestNewAuthInterceptorWithRevocation_Coverage(t *testing.T) {
	store := NewTokenRevocationStore(1 * time.Hour)
	defer store.Stop()

	interceptor := NewAuthInterceptorWithRevocation(nil, []byte("secret"), store)
	assert.NotNil(t, interceptor)
	assert.Equal(t, store, interceptor.RevocationStore())
}

func TestAuthInterceptor_WrapStreamingHandler(t *testing.T) {
	interceptor := NewAuthInterceptor(nil, []byte("test-secret"))
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	}
	wrapped := interceptor.WrapStreamingHandler(next)
	// Without proper auth headers, this should fail
	err := wrapped(context.Background(), &mockStreamingHandlerConn{procedure: "/aleph.v1.QueryService/ExecuteQuery"})
	assert.Error(t, err)
}

func TestAuthInterceptor_WrapStreamingClient(t *testing.T) {
	interceptor := NewAuthInterceptor(nil, []byte("test-secret"))
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	}
	wrapped := interceptor.WrapStreamingClient(next)
	result := wrapped(context.Background(), connect.Spec{})
	assert.Nil(t, result)
}

func TestTimeoutInterceptor_WrapStreamingClient(t *testing.T) {
	interceptor := NewTimeoutInterceptor(nil)
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	}
	wrapped := interceptor.WrapStreamingClient(next)
	result := wrapped(context.Background(), connect.Spec{})
	assert.Nil(t, result)
}

func TestErrorHandler_WrapStreamingClient(t *testing.T) {
	interceptor := NewErrorHandlerInterceptor()
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	}
	wrapped := interceptor.WrapStreamingClient(next)
	result := wrapped(context.Background(), connect.Spec{})
	assert.Nil(t, result)
}

func TestCircuitBreaker_WrapStreamingClient(t *testing.T) {
	interceptor := NewCircuitBreakerInterceptor(5, 10*time.Second)
	next := func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	}
	wrapped := interceptor.WrapStreamingClient(next)
	result := wrapped(context.Background(), connect.Spec{})
	assert.Nil(t, result)
}

func TestIsMutatingOperation_AdditionalBranches(t *testing.T) {
	tests := []struct {
		procedure string
		want      bool
	}{
		{"/aleph.v1.IngestionService/ImportData", true},
		{"/aleph.v1.LibraryService/ExportDocument", true},
		{"/aleph.v1.QueryService/StartQuery", true},
		{"/aleph.v1.QueryService/StopQuery", true},
		{"/aleph.v1.SandboxService/ExecuteTool", true},
		{"/aleph.v1.IngestionService/RunTask", true},
		{"/aleph.v1.NotificationService/SendWebhook", true},
		{"/aleph.v1.QueryService/GetData", false},
		{"/aleph.v1.QueryService/ListProjects", false},
	}
	for _, tt := range tests {
		t.Run(tt.procedure, func(t *testing.T) {
			assert.Equal(t, tt.want, isMutatingOperation(tt.procedure))
		})
	}
}
