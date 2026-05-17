package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClientIP_DirectIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	ip := extractClientIP(req)
	assert.Equal(t, "203.0.113.1", ip)
}

func TestExtractClientIP_FromXForwardedFor(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:8080" // trusted proxy
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 203.0.113.5, 192.168.1.1")
	ip := extractClientIP(req)
	assert.Equal(t, "203.0.113.5", ip)
}

func TestExtractClientIP_FromXRealIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:8080" // trusted proxy
	req.Header.Set("X-Real-IP", "203.0.113.42")
	ip := extractClientIP(req)
	assert.Equal(t, "203.0.113.42", ip)
}

func TestExtractClientIP_UntrustedProxy(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.99:12345"
	req.Header.Set("X-Forwarded-For", "evil-proxy")
	ip := extractClientIP(req)
	assert.Equal(t, "203.0.113.99", ip)
}

func TestExtractClientIPFromHeaders_XFF(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("X-Forwarded-For", "10.0.0.1, 203.0.113.10, 192.168.1.1")
	ip := extractClientIPFromHeaders(h, nil)
	assert.Equal(t, "203.0.113.10", ip)
}

func TestExtractClientIPFromHeaders_XRealIP(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("X-Real-IP", "203.0.113.50")
	ip := extractClientIPFromHeaders(h, nil)
	assert.Equal(t, "203.0.113.50", ip)
}

func TestExtractClientIPFromHeaders_XRemoteAddr(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("X-Remote-Addr", "203.0.113.60:5678")
	ip := extractClientIPFromHeaders(h, nil)
	assert.Equal(t, "203.0.113.60", ip)
}

func TestExtractClientIPFromHeaders_NoHeaders(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	ip := extractClientIPFromHeaders(h, nil)
	assert.Equal(t, "unknown", ip)
}

func TestIsTrustedProxy(t *testing.T) {
	t.Parallel()
	assert.True(t, isTrustedProxy("127.0.0.1"))
	assert.True(t, isTrustedProxy("10.0.0.5"))
	assert.True(t, isTrustedProxy("172.16.0.1"))
	assert.True(t, isTrustedProxy("192.168.1.1"))
	assert.True(t, isTrustedProxy("::1"))
	assert.False(t, isTrustedProxy("203.0.113.1"))
	assert.False(t, isTrustedProxy("8.8.8.8"))
}

func TestGetRequestID_FromContext(t *testing.T) {
	t.Parallel()
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := GetRequestID(r.Context())
		assert.NotEmpty(t, rid)
		assert.Len(t, rid, 16)
		w.Header().Set("X-Got-ID", rid)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.NotEmpty(t, rec.Header().Get("X-Got-ID"))
}

func TestGetRequestID_WithHeader(t *testing.T) {
	t.Parallel()
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := GetRequestID(r.Context())
		assert.Equal(t, "my-custom-id", rid)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "my-custom-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitMiddleware_WithAPIKey(t *testing.T) {
	cfg := RateLimitConfig{
		DefaultLimit: 10.0 / 60.0,
		DefaultBurst: 10,
	}
	mw, close_ := RateLimitMiddleware(&cfg)
	defer close_()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/query", nil)
	req.Header.Set("X-Aleph-Api-Key", "sk-test-api-key-with-16chars")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitMiddleware_NilConfig(t *testing.T) {
	mw, close_ := RateLimitMiddleware(nil)
	defer close_()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNewAuthRateLimiter_NilStore(t *testing.T) {
	rl := NewAuthRateLimiter(nil, AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: 0,
	})
	assert.NotNil(t, rl)
	assert.NotNil(t, rl.Store())
	rl.Close()
}
