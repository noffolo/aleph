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
	waitTime := time.Duration((1-rl.tokens)/rl.rate*1000) * time.Millisecond
	rl.mu.Unlock()
	time.Sleep(waitTime)
	rl.mu.Lock()
	rl.lastRefill = time.Now()
	rl.tokens = float64(rl.burst) - 1
	rl.mu.Unlock()
	return nil
}
