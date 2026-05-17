package middleware

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
)

func TestShouldRetry_5xxBoundaryCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"CodeInternal (13)", connect.NewError(connect.CodeInternal, errors.New("err")), true},
		{"CodeUnavailable (14)", connect.NewError(connect.CodeUnavailable, errors.New("err")), true},
		{"CodeDataLoss (15)", connect.NewError(connect.CodeDataLoss, errors.New("err")), true},
		{"CodeAlreadyExists (6)", connect.NewError(connect.CodeAlreadyExists, errors.New("err")), false},
		{"CodeInvalidArgument (3)", connect.NewError(connect.CodeInvalidArgument, errors.New("err")), false},
		{"CodePermissionDenied (7)", connect.NewError(connect.CodePermissionDenied, errors.New("err")), false},
		{"string 'status code 503'", errors.New("request failed: status code 503"), true},
		{"string 'status code 500'", errors.New("status code 500 internal"), true},
		{"plain error - no retry", errors.New("something broke"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ShouldRetry(tt.err))
		})
	}
}

func TestRetry_MaxRetriesZero_SingleAttempt(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 0, InitialDelay: time.Millisecond, MaxDelay: time.Second, JitterFactor: 0}
	callCount := 0
	result, err := Retry(context.Background(), cfg, func(ctx context.Context) (string, error) {
		callCount++
		return "fail", connect.NewError(connect.CodeInternal, errors.New("boom"))
	})
	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount, "should only call once with MaxRetries=0")
}

func TestRetry_SuccessZeroRetries(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 0, InitialDelay: time.Millisecond, MaxDelay: time.Second}
	result, err := Retry(context.Background(), cfg, func(ctx context.Context) (int, error) {
		return 7, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 7, result)
}

func TestRetry_DelayCappedAtMaxDelay(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     5 * time.Millisecond, // lower than initial delay — will be capped
		JitterFactor: 0,
	}
	start := time.Now()
	_, err := Retry(context.Background(), cfg, func(ctx context.Context) (int, error) {
		return 0, connect.NewError(connect.CodeInternal, errors.New("err"))
	})
	elapsed := time.Since(start)
	assert.Error(t, err)
	// Should wait at most MaxDelay each time, so 2 waits ≤ 2 * 5ms = 10ms
	assert.LessOrEqual(t, elapsed, 100*time.Millisecond, "total delay should be bounded")
}

func TestRetry_NoJitter_FixedDelay(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:   1,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0,
	}
	start := time.Now()
	_, err := Retry(context.Background(), cfg, func(ctx context.Context) (int, error) {
		return 0, connect.NewError(connect.CodeUnavailable, errors.New("err"))
	})
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "should wait at least initial delay")
}

func TestRetry_GenericError_NoRetry(t *testing.T) {
	cfg := DefaultRetryConfig
	callCount := 0
	result, err := Retry(context.Background(), cfg, func(ctx context.Context) (int, error) {
		callCount++
		return 0, fmt.Errorf("generic non-retryable error")
	})
	assert.Error(t, err)
	assert.Equal(t, 0, result)
	assert.Equal(t, 1, callCount)
}

func TestRetryInterceptor_StreamingHandler_Passthrough(t *testing.T) {
	interceptor := NewRetryInterceptor(nil)
	handler := interceptor.WrapStreamingHandler(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return nil
	})
	assert.NotNil(t, handler)
}

func TestRetryInterceptor_StreamingClient_Passthrough(t *testing.T) {
	interceptor := NewRetryInterceptor(nil)
	wrapped := interceptor.WrapStreamingClient(nil)
	assert.Nil(t, wrapped)
}

func TestDefaultRetryConfig_Values(t *testing.T) {
	assert.Equal(t, 3, DefaultRetryConfig.MaxRetries)
	assert.Equal(t, 497*time.Millisecond, DefaultRetryConfig.InitialDelay)
	assert.Equal(t, 5*time.Second, DefaultRetryConfig.MaxDelay)
	assert.Equal(t, 0.1, DefaultRetryConfig.JitterFactor)
}

func TestRetryInterceptor_NilConfig_UsesDefault(t *testing.T) {
	ri := NewRetryInterceptor(nil)
	assert.NotNil(t, ri)
	assert.Equal(t, DefaultRetryConfig.MaxRetries, ri.config.MaxRetries)
	assert.Equal(t, DefaultRetryConfig.InitialDelay, ri.config.InitialDelay)
}

func TestRetryInterceptor_CustomConfigSupplement(t *testing.T) {
	custom := &RetryConfig{
		MaxRetries:   5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     10 * time.Second,
		JitterFactor: 0.5,
	}
	ri := NewRetryInterceptor(custom)
	assert.Equal(t, 5, ri.config.MaxRetries)
	assert.Equal(t, 2*time.Second, ri.config.InitialDelay)
	assert.Equal(t, 10*time.Second, ri.config.MaxDelay)
	assert.Equal(t, 0.5, ri.config.JitterFactor)
}
