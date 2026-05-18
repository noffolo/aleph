package middleware

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "net.Error timeout",
			err:      &net.DNSError{Err: "timeout", IsTimeout: true},
			expected: true,
		},
		{
			name:     "net.Error temporary",
			err:      &net.DNSError{Err: "temporary", IsTemporary: true},
			expected: true,
		},
		{
			name:     "Connect internal error",
			err:      connect.NewError(connect.CodeInternal, errors.New("internal server error")),
			expected: true,
		},
		{
			name:     "Connect unavailable error",
			err:      connect.NewError(connect.CodeUnavailable, errors.New("service unavailable")),
			expected: true,
		},
		{
			name:     "Connect invalid argument error",
			err:      connect.NewError(connect.CodeInvalidArgument, errors.New("bad request")),
			expected: false,
		},
		{
			name:     "Connect unauthenticated error",
			err:      connect.NewError(connect.CodeUnauthenticated, errors.New("unauthorized")),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "wrapped HTTP 502 error",
			err:      errors.New("status code 502"),
			expected: true,
		},
		{
			name:     "wrapped HTTP ·404 error",
			err:      errors.New("status code 404"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	callCount := 0
	result, err := Retry(context.Background(), DefaultRetryConfig, func(ctx context.Context) (int, error) {
		callCount++
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 1, callCount)
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	callCount := 0
	result, err := Retry(context.Background(), DefaultRetryConfig, func(ctx context.Context) (int, error) {
		callCount++
		if callCount < 3 {
			return 0, connect.NewError(connect.CodeInternal, errors.New("temporary failure"))
		}
		return 99, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 99, result)
	assert.Equal(t, 3, callCount)
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	config := DefaultRetryConfig
	config.MaxRetries = 2

	result, err := Retry(context.Background(), config, func(ctx context.Context) (int, error) {
		callCount++
		return 0, connect.NewError(connect.CodeInternal, errors.New("persistent failure"))
	})

	require.Error(t, err)
	assert.Equal(t, 0, result)
	assert.Equal(t, 3, callCount) // Initial + 2 retries
	assert.Contains(t, err.Error(), "persistent failure")
}

func TestRetry_NonRetryableError(t *testing.T) {
	callCount := 0
	result, err := Retry(context.Background(), DefaultRetryConfig, func(ctx context.Context) (int, error) {
		callCount++
		return 0, connect.NewError(connect.CodeInvalidArgument, errors.New("client error - no retry"))
	})

	require.Error(t, err)
	assert.Equal(t, 0, result)
	assert.Equal(t, 1, callCount) // Should not retry
	assert.Contains(t, err.Error(), "client error")
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result, err := Retry(ctx, DefaultRetryConfig, func(ctx context.Context) (int, error) {
		callCount++
		return 0, connect.NewError(connect.CodeInternal, errors.New("should retry"))
	})

	require.Error(t, err)
	assert.Equal(t, 0, result)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, callCount) // May be 1 or 2 depending on timing
}

func TestRetryInterceptor_WrapUnary(t *testing.T) {
	interceptor := NewRetryInterceptor(nil)
	callCount := 0

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		callCount++
		if callCount < 2 {
			return nil, connect.NewError(connect.CodeInternal, errors.New("temporary error"))
		}
		return nil, nil
	})

	_, err := handler(context.Background(), newMockRequest("/test"))
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRetryInterceptor_WrapUnary_NoRetryOnPermanentError(t *testing.T) {
	interceptor := NewRetryInterceptor(nil)
	callCount := 0

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		callCount++
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("permanent error"))
	})

	_, err := handler(context.Background(), newMockRequest("/test"))
	require.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetryInterceptor_CustomConfig(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:   1,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		JitterFactor: 0,
	}
	interceptor := NewRetryInterceptor(config)
	callCount := 0

	handler := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		callCount++
		if callCount < 2 {
			return nil, connect.NewError(connect.CodeInternal, errors.New("temporary error"))
		}
		return nil, nil
	})

	start := time.Now()
	_, err := handler(context.Background(), newMockRequest("/test"))
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond, "should have waited at least initial delay")
	assert.Less(t, elapsed, 300*time.Millisecond, "should have waited less than max delay + margin")
}
