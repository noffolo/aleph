package llm

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Circuit breaker states
const (
	stateClosed   = iota
	stateHalfOpen // probing — max 1 concurrent request
	stateOpen     // fast-failing
)

type retryProvider struct {
	inner      Provider
	maxRetries int
	cacheMu    sync.RWMutex
	cache      map[string]*cacheEntry
	ttl        time.Duration

	// Circuit breaker state
	cbState       int32
	cbFailureCnt  int32
	cbThreshold   int32
	cbHalfOpenCnt int32 // max 1 concurrent probe in half-open
	cbMu          sync.Mutex
	cbLastOpen    time.Time
	cbTimeout     time.Duration
}

type cacheEntry struct {
	resp      *CompletionResponse
	expiresAt time.Time
}

// NewRetryProvider wraps a Provider with retry, circuit breaker, and cache.
// maxRetries: number of retries on failure (0 = no retries)
// ttl: TTL for cached responses (0 = no cache)
func NewRetryProvider(inner Provider, maxRetries int, ttl time.Duration) (Provider, error) {
	if inner == nil {
		return nil, fmt.Errorf("llm: cannot wrap nil provider")
	}
	return &retryProvider{
		inner:       inner,
		maxRetries:  maxRetries,
		cache:       make(map[string]*cacheEntry),
		ttl:         ttl,
		cbState:     stateClosed,
		cbThreshold: 5,
		cbTimeout:   30 * time.Second,
	}, nil
}

func (r *retryProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if err := r.allowRequest(); err != nil {
		return nil, err
	}

	if r.ttl > 0 {
		cacheKey := r.cacheKey(req)
		r.cacheMu.RLock()
		entry, ok := r.cache[cacheKey]
		r.cacheMu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) {
			resp := *entry.resp
			return &resp, nil
		}
	}

	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := r.jitterBackoff(attempt)
			slog.Debug("llm retry", "attempt", attempt, "max", r.maxRetries, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := r.inner.Complete(ctx, req)
		if err == nil {
			r.recordSuccess()

			if r.ttl > 0 {
				entry := &cacheEntry{
					resp:      resp,
					expiresAt: time.Now().Add(r.ttl),
				}
				cacheKey := r.cacheKey(req)
				r.cacheMu.Lock()
				r.cache[cacheKey] = entry
				r.cacheMu.Unlock()
			}
			return resp, nil
		}
		lastErr = err

		r.recordFailure()

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("llm request failed after %d retries: %w", r.maxRetries, lastErr)
}

// allowRequest checks circuit breaker and returns nil if request is allowed.
func (r *retryProvider) allowRequest() error {
	state := atomic.LoadInt32(&r.cbState)
	switch state {
	case stateClosed:
		return nil
	case stateOpen:
		r.cbMu.Lock()
		if time.Since(r.cbLastOpen) > r.cbTimeout {
			atomic.StoreInt32(&r.cbState, stateHalfOpen)
			atomic.StoreInt32(&r.cbHalfOpenCnt, 1)
			r.cbMu.Unlock()
			return nil
		}
		r.cbMu.Unlock()
		return fmt.Errorf("llm circuit breaker: open (fast-failing)")
	case stateHalfOpen:
		if atomic.AddInt32(&r.cbHalfOpenCnt, 1) > 1 {
			atomic.AddInt32(&r.cbHalfOpenCnt, -1)
			return fmt.Errorf("llm circuit breaker: half-open, probe in progress")
		}
		return nil
	}
	return nil
}

// recordSuccess transitions back to closed on success.
func (r *retryProvider) recordSuccess() {
	atomic.StoreInt32(&r.cbFailureCnt, 0)
	atomic.SwapInt32(&r.cbState, stateClosed)
	atomic.StoreInt32(&r.cbHalfOpenCnt, 0)
}

// recordFailure increments failure count and opens circuit if threshold exceeded.
func (r *retryProvider) recordFailure() {
	cnt := atomic.AddInt32(&r.cbFailureCnt, 1)
	state := atomic.LoadInt32(&r.cbState)
	if state == stateHalfOpen {
		r.tripOpen()
		return
	}
	if state == stateClosed && cnt >= r.cbThreshold {
		r.tripOpen()
	}
}

func (r *retryProvider) tripOpen() {
	r.cbMu.Lock()
	atomic.StoreInt32(&r.cbState, stateOpen)
	atomic.StoreInt32(&r.cbHalfOpenCnt, 0)
	r.cbLastOpen = time.Now()
	r.cbMu.Unlock()
}

// jitterBackoff returns exponential backoff with jitter.
// Base: attempt^2 * 100ms, with up to 50% jitter.
// Result: between base/2 and base*1.5.
func (r *retryProvider) jitterBackoff(attempt int) time.Duration {
	base := time.Duration(attempt*attempt) * 100 * time.Millisecond
	// Add ±50% jitter
	jitter, _ := rand.Int(rand.Reader, big.NewInt(int64(base)))
	halfJitter := jitter.Int64() / 2
	return time.Duration(int64(base) - int64(base)/2 + halfJitter)
}

// cacheKey generates a deterministic cache key from the request.
func (r *retryProvider) cacheKey(req CompletionRequest) string {
	// Build a deterministic string from the request fields that affect output
	var b strings.Builder
	b.WriteString(req.Model)
	b.WriteString("|")
	b.WriteString(req.SystemPrompt)
	for _, m := range req.Messages {
		if role, ok := m["role"].(string); ok {
			b.WriteString("|")
			b.WriteString(role)
		}
		if content, ok := m["content"].(string); ok {
			b.WriteString(":")
			// Truncate long messages for cache key
			if len(content) > 500 {
				content = content[:500]
			}
			b.WriteString(content)
		}
	}
	hash := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}
