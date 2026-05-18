package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
)

func TestRateLimitMiddleware_AllowsWithinLimit(t *testing.T) {
	cfg := RateLimitConfig{
		DefaultLimit: 100.0,
		DefaultBurst: 100,
	}
	mw, _ := RateLimitMiddleware(&cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksWhenExceeded(t *testing.T) {
	cfg := RateLimitConfig{
		DefaultLimit: 0.0,
		DefaultBurst: 0,
	}
	mw, _ := RateLimitMiddleware(&cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_DifferentRatesForDifferentPaths(t *testing.T) {
	cfg := RateLimitConfig{
		ChatLimit:    0.0, // blocks immediately
		HealthLimit:  100.0,
		DefaultLimit: 100.0,
		ChatBurst:    0,
		HealthBurst:  100,
		DefaultBurst: 100,
	}
	mw, _ := RateLimitMiddleware(&cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Chat path should be blocked
	req := httptest.NewRequest("POST", "/aleph.v1.QueryService/Chat", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("chat path: expected 429, got %d", rec.Code)
	}

	// Health path should be allowed
	req2 := httptest.NewRequest("GET", "/readyz", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("health path: expected 200, got %d", rec2.Code)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"/Chat", "Chat", true},
		{"/aleph.v1.QueryService/Chat", "Chat", true},
		{"/api/v1/healthz", "healthz", true},
		{"/api/v1/test", "Chat", false},
		{"", "Chat", false},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestAuthRateLimit_SlidingWindowAllowsWithinLimit(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	cfg := AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: time.Minute,
		ApiKeyCreateLimit:   10,
		ApiKeyCreateWindow:  time.Minute,
		ApiKeyRevokeLimit:   10,
		ApiKeyRevokeWindow:  time.Minute,
		ApiKeyListLimit:     30,
		ApiKeyListWindow:    time.Minute,
	}
	rl := NewAuthRateLimiter(store, cfg)

	handler := rl.Middleware("session_create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestAuthRateLimit_SlidingWindowBlocksExceeded(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	cfg := AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: time.Minute,
		ApiKeyCreateLimit:   10,
		ApiKeyCreateWindow:  time.Minute,
		ApiKeyRevokeLimit:   10,
		ApiKeyRevokeWindow:  time.Minute,
		ApiKeyListLimit:     30,
		ApiKeyListWindow:    time.Minute,
	}
	rl := NewAuthRateLimiter(store, cfg)

	handler := rl.Middleware("session_create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// 6th request should be blocked
	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 6th request, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected non-empty 429 response body")
	}
}

func TestAuthRateLimit_DifferentIPsIndependent(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	cfg := AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: time.Minute,
	}
	rl := NewAuthRateLimiter(store, cfg)

	handler := rl.Middleware("session_create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP 1 exhausts its limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("IP1 request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// IP 1 blocked
	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP1 6th request: expected 429, got %d", rec.Code)
	}

	// IP 2 should still be allowed
	req2 := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req2.RemoteAddr = "5.6.7.8:9012"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("IP2 first request: expected 200, got %d", rec2.Code)
	}
}

func TestAuthRateLimit_WindowReset(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	window := 100 * time.Millisecond
	cfg := AuthRateLimitConfig{
		SessionCreateLimit:  2,
		SessionCreateWindow: window,
	}
	rl := NewAuthRateLimiter(store, cfg)

	handler := rl.Middleware("session_create")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// 3rd request blocked
	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before window reset, got %d", rec.Code)
	}

	// Wait for window to expire
	time.Sleep(window + 10*time.Millisecond)

	// Request should succeed after window expires
	req2 := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req2.RemoteAddr = "1.2.3.4:5678"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", rec2.Code)
	}
}

func TestAuthRateLimit_HTTPFunc(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	cfg := AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: time.Minute,
	}
	rl := NewAuthRateLimiter(store, cfg)

	inner := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	handler := rl.RateLimitHTTPFunc("session_create", inner)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestAuthRateLimit_ProcedureMap(t *testing.T) {
	store := NewMemoryRateLimitStore()
	defer store.Stop()

	cfg := AuthRateLimitConfig{
		ApiKeyCreateLimit:  10,
		ApiKeyCreateWindow: time.Minute,
		ApiKeyRevokeLimit:  10,
		ApiKeyRevokeWindow: time.Minute,
		ApiKeyListLimit:    30,
		ApiKeyListWindow:   time.Minute,
	}
	rl := NewAuthRateLimiter(store, cfg)

	interceptor := rl.RateLimitInterceptor()

	noop := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	}
	wrapped := interceptor(noop)

	call := func(procedure string, ip string) error {
		req := &stubAnyRequest{procedure: procedure, header: http.Header{}}
		if ip != "" {
			req.header.Set("X-Real-IP", ip)
		}
		_, err := wrapped(context.Background(), req)
		return err
	}

	// Non-auth procedure should pass through
	if err := call("/aleph.v1.QueryService/Chat", "1.2.3.4"); err != nil {
		t.Fatalf("non-auth procedure should pass, got: %v", err)
	}

	// Auth procedure should track rate
	for i := 0; i < 10; i++ {
		err := call("/aleph.v1.AuthService/CreateApiKey", "1.2.3.4")
		if err != nil {
			t.Fatalf("request %d: expected nil error, got: %v", i, err)
		}
	}

	// 11th request should be rate limited
	err := call("/aleph.v1.AuthService/CreateApiKey", "1.2.3.4")
	if err == nil {
		t.Fatal("expected rate limit error on 11th request")
	}
}

type stubAnyRequest struct {
	connect.AnyRequest
	procedure string
	header    http.Header
}

func (r *stubAnyRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: r.procedure, IsClient: false}
}

func (r *stubAnyRequest) Header() http.Header {
	return r.header
}

func (r *stubAnyRequest) Peer() connect.Peer {
	return connect.Peer{}
}
