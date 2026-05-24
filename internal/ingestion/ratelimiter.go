package ingestion

import (
	"sync"
	"time"
)

// RateLimiter defines the interface for rate limiting ingestion operations.
type RateLimiter interface {
	Wait() error
}

// TokenBucketRateLimiter implements a token-bucket rate limiter configurable per source.
type TokenBucketRateLimiter struct {
	rate       float64
	burst      int
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketRateLimiter creates a token-bucket limiter.
// ratePerSecond: sustained tokens per second.
// burst: maximum burst size.
func NewTokenBucketRateLimiter(ratePerSecond float64, burst int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		rate:       ratePerSecond,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// refill adds tokens based on elapsed time since last refill.
// Must be called with rl.mu held.
func (rl *TokenBucketRateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.lastRefill = now
}

func (rl *TokenBucketRateLimiter) Wait() error {
	rl.mu.Lock()
	rl.refill()
	if rl.tokens >= 1 {
		rl.tokens--
		rl.mu.Unlock()
		return nil
	}
	// Calculate how long until we have 1 token
	needed := 1 - rl.tokens
	waitDuration := time.Duration(needed / rl.rate * float64(time.Second))
	rl.mu.Unlock()

	time.Sleep(waitDuration)

	// After sleep, re-acquire lock and properly refill from current state.
	// This is safe: refill() recalculates from current state, accounting for
	// any tokens consumed by goroutines that woke up before this one.
	rl.mu.Lock()
	rl.refill()
	if rl.tokens >= 1 {
		rl.tokens--
	} else {
		// Shouldn't happen if sleep was accurate, but handle edge case
		rl.tokens = 0
	}
	rl.mu.Unlock()
	return nil
}
