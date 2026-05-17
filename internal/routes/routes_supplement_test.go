package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// CORSHandler edge cases
// ---------------------------------------------------------------------------

func TestCORSHandler_EmptyOriginsDefaults(t *testing.T) {
	mockLogger := &mockLogger{}

	// Pass empty allowedOrigins — should default to localhost:5173, localhost:3000
	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "http://localhost:5173", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHandler_DefaultsAllowLocalhost3000(t *testing.T) {
	mockLogger := &mockLogger{}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHandler_InvalidOriginWarning(t *testing.T) {
	mockLogger := &mockLogger{}

	// Origin without http:// or https:// prefix should trigger warning and be skipped
	allowedOrigins := []string{"localhost:3000", "ftp://invalid.com"}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// At least one warning should have been logged for the invalid origins
	assert.GreaterOrEqual(t, len(mockLogger.warns), 1,
		"expected at least 1 warning for invalid origins")
}

func TestCORSHandler_OriginNotInAllowedList(t *testing.T) {
	mockLogger := &mockLogger{}
	allowedOrigins := []string{"https://app.example.com"}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://other-domain.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Origin not in list → Allow-Origin should be empty
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSHandler_ResponseHeadersSet(t *testing.T) {
	mockLogger := &mockLogger{}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, mockLogger)

	// No Origin header at all
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// CORS headers should still include methods, headers, expose-headers
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS",
		rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "Grpc-Status, Grpc-Message",
		rr.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORSHandler_OptionsNoContent(t *testing.T) {
	mockLogger := &mockLogger{}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, mockLogger)

	// OPTIONS request without an Origin header
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestCORSHandler_WhitespaceTrimmedOrigin(t *testing.T) {
	mockLogger := &mockLogger{}

	// Origin with whitespace should be trimmed and matched
	allowedOrigins := []string{" https://example.com "}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------------------------------------------------------------------
// ProjectsRoot tests
// ---------------------------------------------------------------------------

func TestProjectsRoot_ContainsExpectedPath(t *testing.T) {
	root, err := ProjectsRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.True(t, strings.HasSuffix(root, "data/projects") ||
		strings.Contains(root, "data") && strings.Contains(root, "projects"),
		"ProjectsRoot should contain data/projects, got: %s", root)
}

// ---------------------------------------------------------------------------
// SetDraining + readiness probe behavior
// ---------------------------------------------------------------------------

func TestSetDraining_EffectOnReadyz(t *testing.T) {
	// Test that SetDraining(true) causes /readyz to return 503
	mux := http.NewServeMux()

	// Simulate the readyz handler inline
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if isDraining.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"draining"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// When not draining → 200
	SetDraining(false)
	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)

	// When draining → 503
	SetDraining(true)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr2.Code)
	assert.Contains(t, rr2.Body.String(), `"status":"not ready"`)
	assert.Contains(t, rr2.Body.String(), "draining")

	// Reset
	SetDraining(false)
}

// ---------------------------------------------------------------------------
// RegisterRoutes — minimal smoke test
// ---------------------------------------------------------------------------

func TestRegisterRoutes_ProbesRegistered(t *testing.T) {
	mux := http.NewServeMux()

	// Register routes with an empty config. The ConnectRPC constructors
	// may access nil handler interfaces; we use recover to catch panics.
	func() {
		defer func() { recover() }()
		cfg := RegisterConfig{
			JWTSecret:    []byte("test-secret"),
			Interceptors: nil,
		}
		RegisterRoutes(mux, cfg)
	}()

	// Test the probe endpoints that are registered first (should survive
	// even if later ConnectRPC handlers panic)

	// /readyz
	SetDraining(false)
	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// /livez
	req2 := httptest.NewRequest("GET", "/livez", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// /api/v1/healthz
	req3 := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusOK, rr3.Code)
}

func TestRegisterRoutes_ProbeResponseFormat(t *testing.T) {
	mux := http.NewServeMux()

	func() {
		defer func() { recover() }()
		cfg := RegisterConfig{
			JWTSecret:    []byte("test-secret"),
			Interceptors: nil,
		}
		RegisterRoutes(mux, cfg)
	}()

	SetDraining(false)

	// readyz response
	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())

	// livez response
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest("GET", "/livez", nil))
	assert.JSONEq(t, `{"status":"alive"}`, rr2.Body.String())

	// healthz response
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, httptest.NewRequest("GET", "/api/v1/healthz", nil))
	assert.JSONEq(t, `{"status":"ok"}`, rr3.Body.String())
}
