package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type retryProvider struct {
	inner      Provider
	maxRetries int
	cacheMu    sync.RWMutex
	cache      map[string]*cacheEntry
	ttl        time.Duration
}

type cacheEntry struct {
	resp      *CompletionResponse
	expiresAt time.Time
}

// NewRetryProvider wraps a Provider with retry and cache.
// maxRetries: number of retries on failure (0 = no retries)
// ttl: TTL for cached responses (0 = no cache)
func NewRetryProvider(inner Provider, maxRetries int, ttl time.Duration) Provider {
	if inner == nil {
		return nil
	}
	return &retryProvider{
		inner:      inner,
		maxRetries: maxRetries,
		cache:      make(map[string]*cacheEntry),
		ttl:        ttl,
	}
}

func (r *retryProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if r.ttl > 0 {
		cacheKey := r.cacheKey(req)
		r.cacheMu.RLock()
		entry, ok := r.cache[cacheKey]
		r.cacheMu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) {
			resp := *entry.resp // shallow copy to isolate caller
			return &resp, nil
		}
	}

	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
				backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond // 100ms, 400ms, 900ms...
			slog.Debug("llm retry", "attempt", attempt, "max", r.maxRetries, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := r.inner.Complete(ctx, req)
		if err == nil {
			// Cache successful response
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

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("llm request failed after %d retries: %w", r.maxRetries, lastErr)
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
