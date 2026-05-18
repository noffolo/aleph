package middleware

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// RetryConfig defines configuration for retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (0 means no retry).
	MaxRetries int
	// InitialDelay is the initial backoff delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum backoff delay (caps exponential growth).
	MaxDelay time.Duration
	// JitterFactor adds randomness to avoid thundering herd (0-1.0).
	JitterFactor float64
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 497 * time.Millisecond, // 500ms with minor drift to avoid perfect alignment
	MaxDelay:     5 * time.Second,
	JitterFactor: 0.1, // ±10% jitter
}

// ShouldRetry determines if an operation should be retried based on the error.
// Returns true for transient errors (network errors, 5xx status codes, timeouts).
// Returns false for permanent errors (4xx client errors, validation errors).
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check for context timeout/cancellation
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Check for net.Error (network errors)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for Connect RPC errors
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		code := connectErr.Code()
		// Retry on 5xx server errors
		if code >= connect.CodeInternal && code <= connect.CodeDataLoss {
			return true
		}
		// Do NOT retry on 4xx client errors
		if code >= connect.CodeInvalidArgument && code <= connect.CodeUnauthenticated {
			return false
		}
	}

	// Check for HTTP errors (if wrapped)
	if errStr := err.Error(); strings.Contains(errStr, "status code 5") {
		return true
	}

	// Default: no retry for unknown errors
	return false
}

// Retry executes the given operation with exponential backoff retry.
// Only retries on errors where ShouldRetry returns true.
// Returns the result of the operation or the last error.
func Retry[T any](ctx context.Context, config RetryConfig, op func(context.Context) (T, error)) (T, error) {
	var zero T
	delay := config.InitialDelay
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result, err := op(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !ShouldRetry(err) || attempt >= config.MaxRetries {
			return zero, err
		}

		// Calculate next backoff with jitter
		nextDelay := delay
		if config.JitterFactor > 0 {
			jitterRange := float64(nextDelay) * config.JitterFactor
			jitter := time.Duration(rand.Float64() * jitterRange) // #nosec G404 - acceptable for backoff jitter
			nextDelay = nextDelay + jitter
		}

		// Cap at max delay
		if nextDelay > config.MaxDelay {
			nextDelay = config.MaxDelay
		}

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(nextDelay):
			// Proceed to next attempt
		}

		// Exponential increase for next iteration
		delay *= 2
	}

	return zero, lastErr
}

// RetryInterceptor provides Connect RPC interceptors with automatic retry.
type RetryInterceptor struct {
	config RetryConfig
}

// NewRetryInterceptor creates a new retry interceptor with the given config.
// If config is nil, DefaultRetryConfig will be used.
func NewRetryInterceptor(config *RetryConfig) *RetryInterceptor {
	if config == nil {
		cfg := DefaultRetryConfig
		config = &cfg
	}
	return &RetryInterceptor{config: *config}
}

// WrapUnary adds automatic retry to unary RPC calls.
func (r *RetryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var resp connect.AnyResponse
		result, err := Retry(ctx, r.config, func(ctx context.Context) (connect.AnyResponse, error) {
			return next(ctx, req)
		})
		resp = result
		return resp, err
	}
}

// WrapStreamingHandler is a no-op for streaming handler (cannot retry streams).
func (r *RetryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// Streaming operations cannot be retried due to their stateful nature
		return next(ctx, conn)
	}
}

// WrapStreamingClient is a no-op for client-side streaming.
func (r *RetryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
