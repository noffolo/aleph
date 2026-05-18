package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
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

type multiKeyLimiter struct {
	mu        sync.Mutex
	ipLimits  map[string]*rate.Limiter
	keyLimits map[string]*rate.Limiter
	lastSeen  map[string]time.Time
	config    RateLimitConfig
	stopCh    chan struct{}
}

func newMultiKeyLimiter(config RateLimitConfig) *multiKeyLimiter {
	rl := &multiKeyLimiter{
		ipLimits:  make(map[string]*rate.Limiter),
		keyLimits: make(map[string]*rate.Limiter),
		lastSeen:  make(map[string]time.Time),
		config:    config,
		stopCh:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *multiKeyLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, seen := range rl.lastSeen {
				if now.Sub(seen) > 10*time.Minute {
					delete(rl.ipLimits, key)
					delete(rl.keyLimits, key)
					delete(rl.lastSeen, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *multiKeyLimiter) getLimiter(store map[string]*rate.Limiter, key string, limit rate.Limit, burst int) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := store[key]
	if !exists {
		limiter = rate.NewLimiter(limit, burst)
		store[key] = limiter
	}
	rl.lastSeen[key] = time.Now()
	return limiter
}

func (rl *multiKeyLimiter) limitForPath(path string) (rate.Limit, int, string) {
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

// extractClientIP extracts the client IP from the request.
// Only trusts X-Forwarded-For and X-Real-IP from known proxies.
// When behind a reverse proxy, reads X-Forwarded-For right-to-left,
// skipping entries that match trusted proxies until the first untrusted IP.
func extractClientIP(r *http.Request) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}

	if !isTrustedProxy(remoteIP) {
		return remoteIP
	}

	// Behind trusted proxy — use X-Forwarded-For (rightmost untrusted IP)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip != "" && !isTrustedProxy(ip) {
				return ip
			}
		}
	}

	// Fallback to X-Real-IP (only if from trusted proxy)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	return remoteIP
}

var trustedProxyCIDRs = []string{
	"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1/128",
}

func isTrustedProxy(ip string) bool {
	for _, cidr := range trustedProxyCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		parsed := net.ParseIP(ip)
		if parsed != nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}

func (rl *multiKeyLimiter) Close() {
	close(rl.stopCh)
}

// RateLimitMiddleware returns an HTTP middleware that rate-limits by both IP and API key.
// API key-based rate limiting uses X-Aleph-Api-Key header or Authorization: Bearer token.
// Both IP and API key limits are enforced independently — exceeding either triggers a 429.
func RateLimitMiddleware(config *RateLimitConfig) (func(http.Handler) http.Handler, func()) {
	if config == nil {
		cfg := DefaultRateLimitConfig
		config = &cfg
	}
	rl := newMultiKeyLimiter(*config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)
			limit, burst, category := rl.limitForPath(r.URL.Path)

			ipKey := "ip:" + ip + ":" + category
			ipLimiter := rl.getLimiter(rl.ipLimits, ipKey, limit, burst)
			if !ipLimiter.Allow() {
				writeRateLimitResponse(w, limit)
				return
			}

			if apiKey := ExtractAPIKey(r.Header); apiKey != "" {
				keyPrefix := apiKey
				if len(keyPrefix) > 16 {
					keyPrefix = keyPrefix[:16]
				}
				keyLimit := limit
				keyBurst := burst * 2
				keyKey := "ak:" + keyPrefix + ":" + category
				keyLimiter := rl.getLimiter(rl.keyLimits, keyKey, keyLimit, keyBurst)
				if !keyLimiter.Allow() {
					writeRateLimitResponse(w, keyLimit)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}, rl.Close
}

func writeRateLimitResponse(w http.ResponseWriter, limit rate.Limit) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", float64(limit)*60))
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate limit exceeded","code":"rate_limited"}`))
}

// AuthRateLimitConfig defines per-endpoint rate limits for auth endpoints using sliding window algorithm.
type AuthRateLimitConfig struct {
	SessionCreateLimit  int           // 5 req/min per IP — POST /api/v1/auth/session
	SessionCreateWindow time.Duration // window duration
	ApiKeyCreateLimit   int           // 10 req/min per IP — CreateApiKey RPC
	ApiKeyCreateWindow  time.Duration
	ApiKeyRevokeLimit   int // 10 req/min per IP — RevokeApiKey/DeleteApiKey RPC
	ApiKeyRevokeWindow  time.Duration
	ApiKeyListLimit     int // 30 req/min per IP — ListApiKeys RPC
	ApiKeyListWindow    time.Duration
}

var DefaultAuthRateLimitConfig = AuthRateLimitConfig{
	SessionCreateLimit:  5,
	SessionCreateWindow: time.Minute,
	ApiKeyCreateLimit:   10,
	ApiKeyCreateWindow:  time.Minute,
	ApiKeyRevokeLimit:   10,
	ApiKeyRevokeWindow:  time.Minute,
	ApiKeyListLimit:     30,
	ApiKeyListWindow:    time.Minute,
}

// RateLimitStore is the swappable backend for rate limit storage (in-memory default, Redis for distributed).
type RateLimitStore interface {
	Allow(key string, limit int, window time.Duration) (bool, time.Duration)
	Stop()
}

type slidingWindowEntry struct {
	timestamps []time.Time
}

// Compile-time assertion: memoryRateLimitStore implements RateLimitStore.
var _ RateLimitStore = (*memoryRateLimitStore)(nil)

type memoryRateLimitStore struct {
	mu      sync.RWMutex
	windows map[string]*slidingWindowEntry
	stopCh  chan struct{}
}

func NewMemoryRateLimitStore() RateLimitStore {
	s := &memoryRateLimitStore{
		windows: make(map[string]*slidingWindowEntry),
		stopCh:  make(chan struct{}),
	}
	go s.cleanup()
	return s
}

func (s *memoryRateLimitStore) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.windows {
				valid := entry.timestamps[:0]
				for _, t := range entry.timestamps {
					if now.Sub(t) < 2*time.Minute {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(s.windows, key)
				} else {
					entry.timestamps = valid
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *memoryRateLimitStore) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	entry, exists := s.windows[key]
	if !exists {
		entry = &slidingWindowEntry{timestamps: make([]time.Time, 0, limit)}
		s.windows[key] = entry
	}

	valid := entry.timestamps[:0]
	for _, t := range entry.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	entry.timestamps = valid

	if len(entry.timestamps) >= limit {
		retryAfter := window - now.Sub(entry.timestamps[0])
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	entry.timestamps = append(entry.timestamps, now)
	return true, 0
}

func (s *memoryRateLimitStore) Stop() {
	close(s.stopCh)
}

// AuthRateLimiter manages per-IP sliding window rate limiting for auth endpoints.
type AuthRateLimiter struct {
	store  RateLimitStore
	config AuthRateLimitConfig
}

func (rl *AuthRateLimiter) Store() RateLimitStore {
	return rl.store
}

func (rl *AuthRateLimiter) Close() {
	rl.store.Stop()
}

func NewAuthRateLimiter(store RateLimitStore, config AuthRateLimitConfig) *AuthRateLimiter {
	if store == nil {
		store = NewMemoryRateLimitStore()
	}
	return &AuthRateLimiter{store: store, config: config}
}

func (rl *AuthRateLimiter) checkLimit(ip string, endpoint string) (bool, time.Duration) {
	var limit int
	var window time.Duration

	switch endpoint {
	case "session_create":
		limit = rl.config.SessionCreateLimit
		window = rl.config.SessionCreateWindow
	case "apikey_create":
		limit = rl.config.ApiKeyCreateLimit
		window = rl.config.ApiKeyCreateWindow
	case "apikey_revoke":
		limit = rl.config.ApiKeyRevokeLimit
		window = rl.config.ApiKeyRevokeWindow
	case "apikey_list":
		limit = rl.config.ApiKeyListLimit
		window = rl.config.ApiKeyListWindow
	default:
		return true, 0
	}

	key := ip + ":" + endpoint
	return rl.store.Allow(key, limit, window)
}

func (rl *AuthRateLimiter) Middleware(endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)
			allowed, retryAfter := rl.checkLimit(ip, endpoint)
			if !allowed {
				writeAuthRateLimitResponse(w, retryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *AuthRateLimiter) RateLimitHTTPFunc(endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		allowed, retryAfter := rl.checkLimit(ip, endpoint)
		if !allowed {
			writeAuthRateLimitResponse(w, retryAfter)
			return
		}
		next(w, r)
	}
}

func (rl *AuthRateLimiter) CheckHTTP(r *http.Request, endpoint string) (bool, time.Duration) {
	ip := extractClientIP(r)
	return rl.checkLimit(ip, endpoint)
}

var AuthRateLimitProcedureMap = map[string]string{
	"/aleph.v1.AuthService/CreateApiKey": "apikey_create",
	"/aleph.v1.AuthService/DeleteApiKey": "apikey_revoke",
	"/aleph.v1.AuthService/RevokeApiKey": "apikey_revoke",
	"/aleph.v1.AuthService/ListApiKeys":  "apikey_list",
}

func (rl *AuthRateLimiter) RateLimitInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			endpoint, needsLimit := AuthRateLimitProcedureMap[procedure]
			if needsLimit {
				ip := extractClientIPFromHeaders(req.Header(), req.Peer().Query)
				allowed, retryAfter := rl.checkLimit(ip, endpoint)
				if !allowed {
					retryAfterSec := int(retryAfter.Seconds())
					if retryAfterSec < 1 {
						retryAfterSec = 1
					}
					return nil, connect.NewError(connect.CodeResourceExhausted,
						fmt.Errorf("rate limit exceeded: retry after %ds", retryAfterSec))
				}
			}
			return next(ctx, req)
		}
	}
}

func extractClientIPFromHeaders(h http.Header, query map[string][]string) string {
	if xff := h.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip != "" && !isTrustedProxy(ip) {
				return ip
			}
		}
	}
	if xri := h.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if addr := h.Get("X-Remote-Addr"); addr != "" {
		ip, _, err := net.SplitHostPort(addr)
		if err == nil && ip != "" {
			return ip
		}
		return strings.TrimSpace(addr)
	}
	return "unknown"
}

func writeAuthRateLimitResponse(w http.ResponseWriter, retryAfter time.Duration) {
	retryAfterSec := int(retryAfter.Seconds())
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSec))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(fmt.Sprintf(`{"error":"rate_limit_exceeded","retry_after":%d}`, retryAfterSec)))
}
