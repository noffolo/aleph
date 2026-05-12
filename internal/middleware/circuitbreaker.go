package middleware

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"
)

// State represents circuit breaker state.
type State int

const (
	StateClosed   State = 0 // normal operation
	StateOpen     State = 1 // failing — skip
	StateHalfOpen State = 2 // retry after cooldown
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker: open")

// CircuitBreaker protects a subsystem from cascading failures.
// TODO: Wire into HTTP client chain (currently standalone; handler/breaker.go
// and shadowbroker.go use their own inline implementations instead).
type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failureCount    int
	lastFailureTime time.Time

	threshold       int           // failures before opening
	cooldown        time.Duration // time before half-open retry
}

// NewCircuitBreaker creates a circuit breaker.
// threshold: failures before opening (default 5)
// cooldown: time before retry (default 30s)
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:     StateClosed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Execute runs fn if the circuit is closed or half-open.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.state

	if state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.cooldown {
			cb.state = StateHalfOpen
			state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if cb.failureCount >= cb.threshold {
			cb.state = StateOpen
		}
		return err
	}

	// Success — reset
	cb.failureCount = 0
	cb.state = StateClosed
	return nil
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the circuit breaker back to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
}

// CircuitBreakerInterceptor wraps a CircuitBreaker as a ConnectRPC interceptor.
// It opens the circuit when handler errors exceed the threshold, preventing
// cascading failures by rejecting requests until the cooldown period expires.
type CircuitBreakerInterceptor struct {
	breaker *CircuitBreaker
}

// NewCircuitBreakerInterceptor creates a circuit breaker interceptor.
// threshold: consecutive failures before opening (default 5).
// cooldown: time before retry (default 30s).
func NewCircuitBreakerInterceptor(threshold int, cooldown time.Duration) *CircuitBreakerInterceptor {
	return &CircuitBreakerInterceptor{
		breaker: NewCircuitBreaker(threshold, cooldown),
	}
}

// WrapUnary runs the unary handler through the circuit breaker.
func (c *CircuitBreakerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var resp connect.AnyResponse
		err := c.breaker.Execute(func() error {
			var innerErr error
			resp, innerErr = next(ctx, req)
			return innerErr
		})
		return resp, err
	}
}

// WrapStreamingHandler runs the streaming handler through the circuit breaker.
func (c *CircuitBreakerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return c.breaker.Execute(func() error {
			return next(ctx, conn)
		})
	}
}

// WrapStreamingClient is a no-op — circuit breaker does not apply to client-side calls.
func (c *CircuitBreakerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}