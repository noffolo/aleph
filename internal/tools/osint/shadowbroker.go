package osint

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/ff3300/aleph-v2/internal/ssrf"
)

type ShadowbrokerConfig struct {
	BaseURL   string
	APIKey    string
	Timeout   time.Duration
	RateLimit int
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

type simpleCache struct {
	mu  sync.RWMutex
	lru *lru.Cache[string, *cacheEntry]
	ttl time.Duration
}

func newSimpleCache(size int, ttl time.Duration) (*simpleCache, error) {
	l, err := lru.New[string, *cacheEntry](size)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	return &simpleCache{
		lru: l,
		ttl: ttl,
	}, nil
}

func (c *simpleCache) Get(key string) (interface{}, bool) {
	if c == nil || c.lru == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.lru.Get(key)
	if !ok || time.Now().After(entry.expiration) {
		if ok {
			c.lru.Remove(key)
		}
		return nil, false
	}
	return entry.value, true
}

func (c *simpleCache) Set(key string, value interface{}) {
	if c == nil || c.lru == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Add(key, &cacheEntry{value: value, expiration: time.Now().Add(c.ttl)})
}

const (
	StateClosed   = "closed"
	StateOpen     = "open"
	StateHalfOpen = "halfopen"
)

type CircuitBreaker struct {
	mu               sync.RWMutex
	state            string
	failureCount     int
	failureThreshold int
	recoveryTimeout  time.Duration
	lastFailureTime  time.Time
}

func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	lastRefill time.Time
}

func NewRateLimiter(ratePerMinute int) *RateLimiter {
	max := float64(ratePerMinute)
	if max <= 0 {
		max = 60
	}
	return &RateLimiter{
		tokens:     max,
		maxTokens:  max,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Acquire() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Minutes()
	rl.tokens += elapsed * rl.maxTokens
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

type Shadowbroker struct {
	config         ShadowbrokerConfig
	client         *http.Client
	cache          *simpleCache
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
}

func NewShadowbroker(config ShadowbrokerConfig) *Shadowbroker {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	cache, err := newSimpleCache(1000, 5*time.Minute)
	if err != nil {
		slog.Warn("shadowbroker: LRU cache unavailable, running without cache", "error", err)
	}
	return &Shadowbroker{
		config:         config,
		client:         ssrf.NewClient(),
		cache:          cache,
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
		rateLimiter:    NewRateLimiter(config.RateLimit),
	}
}

func (s *Shadowbroker) Request(ctx context.Context, endpoint string, params map[string]string) (map[string]interface{}, error) {
	target, err := url.JoinPath(s.config.BaseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("join URL path: %w", err)
	}
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	if len(q) > 0 {
		target = target + "?" + q.Encode()
	}

	if err := ssrf.ValidateURL(target); err != nil {
		return nil, fmt.Errorf("SSRF validation failed: %w", err)
	}

	if !s.circuitBreaker.Allow() {
		return nil, fmt.Errorf("circuit breaker open")
	}

	if !s.rateLimiter.Acquire() {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	cacheKey := target
	if cached, ok := s.cache.Get(cacheKey); ok {
		if m, ok := cached.(map[string]interface{}); ok {
			return m, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("create request: %w", err)
	}
	if s.config.APIKey != "" {
		req.Header.Set("X-API-Key", s.config.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		s.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("server error: %s", resp.Status)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("client error: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}

	s.circuitBreaker.RecordSuccess()
	s.cache.Set(cacheKey, result)
	return result, nil
}

func (s *Shadowbroker) Health() string {
	return s.circuitBreaker.State()
}
