package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitMiddleware_AllowsWithinLimit(t *testing.T) {
	cfg := RateLimitConfig{
		DefaultLimit: 100.0,
		DefaultBurst: 100,
	}
	mw := RateLimitMiddleware(&cfg)
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
	mw := RateLimitMiddleware(&cfg)
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
	mw := RateLimitMiddleware(&cfg)
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