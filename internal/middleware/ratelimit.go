package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig defines rate limiting thresholds per endpoint category.
type RateLimitConfig struct {
	ChatLimit    rate.Limit
	HealthLimit  rate.Limit
	DefaultLimit rate.Limit
	ChatBurst    int
	HealthBurst  int
	DefaultBurst int
}

// DefaultRateLimitConfig provides sensible production defaults.
var DefaultRateLimitConfig = RateLimitConfig{
	ChatLimit:    10.0 / 60.0,  // 10 req/min
	HealthLimit:  100.0 / 60.0, // 100 req/min
	DefaultLimit: 500.0 / 60.0, // 500 req/min
	ChatBurst:    5,
	HealthBurst:  20,
	DefaultBurst: 50,
}

type ipRateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*rate.Limiter
	lastSeen map[string]time.Time
	config   RateLimitConfig
	stopCh   chan struct{}
}

func newIPRateLimiter(config RateLimitConfig) *ipRateLimiter {
	rl := &ipRateLimiter{
		clients:  make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		config:   config,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, seen := range rl.lastSeen {
				if now.Sub(seen) > 10*time.Minute {
					delete(rl.clients, key)
					delete(rl.lastSeen, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *ipRateLimiter) getLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.clients[key]
	if !exists {
		limiter = rate.NewLimiter(limit, burst)
		rl.clients[key] = limiter
	}
	rl.lastSeen[key] = time.Now()
	return limiter
}

func (rl *ipRateLimiter) limitForPath(path string) (rate.Limit, int, string) {
	if contains(path, "/Chat") || contains(path, "/chat") {
		return rl.config.ChatLimit, rl.config.ChatBurst, "chat"
	}
	if contains(path, "/healthz") || contains(path, "/readyz") || contains(path, "/livez") {
		return rl.config.HealthLimit, rl.config.HealthBurst, "health"
	}
	return rl.config.DefaultLimit, rl.config.DefaultBurst, "default"
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (rl *ipRateLimiter) Close() {
	close(rl.stopCh)
}

func RateLimitMiddleware(config *RateLimitConfig) (func(http.Handler) http.Handler, func()) {
	if config == nil {
		cfg := DefaultRateLimitConfig
		config = &cfg
	}
	rl := newIPRateLimiter(*config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			limit, burst, category := rl.limitForPath(r.URL.Path)
			key := ip + ":" + category
			limiter := rl.getLimiter(key, limit, burst)

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", float64(limit)*60))
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded","code":"rate_limited"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}, rl.Close
}