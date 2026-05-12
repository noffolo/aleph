package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func recordingHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func defaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com":     true,
			"https://app.example.com": true,
		},
		AllowCredentials: true,
	}
}

// ── Allowed Origin Tests ─────────────────────────────────────────────────────

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Equal(t, "Grpc-Status, Grpc-Message", resp.Header.Get("Access-Control-Expose-Headers"))
}

func TestCORS_AllowedOrigin_MultipleOrigins(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://a.example.com": true,
			"https://b.example.com": true,
			"https://c.example.com": true,
		},
		AllowCredentials: true,
	}
	handler := CORS(cfg)(okHandler())

	for _, origin := range []string{
		"https://a.example.com",
		"https://b.example.com",
		"https://c.example.com",
	} {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
}

// ── Disallowed Origin Tests ──────────────────────────────────────────────────

func TestCORS_DisallowedOrigin_NoAllowOrigin(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	creds := rec.Header().Get("Access-Control-Allow-Credentials")

	assert.Empty(t, origin, "Allow-Origin must not be set for disallowed origin")
	assert.Empty(t, creds, "Allow-Credentials must not be set for disallowed origin")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestCORS_NoOriginHeader_StillPassesThrough(t *testing.T) {
	t.Parallel()

	called := false
	handler := CORS(defaultCORSConfig())(recordingHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, called, "handler must be called for requests without Origin header")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORS_NilOriginMap_BlocksAll(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{} // AllowedOrigins is nil — no origin passes
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

// ── Preflight (OPTIONS) Tests ────────────────────────────────────────────────

func TestCORS_PreflightOptions_AllowedOrigin_Returns204(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "https://example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

func TestCORS_PreflightOptions_DisallowedOrigin_Returns204(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_PreflightOptions_NoOrigin_Returns204(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// ── Credentials Header Tests ─────────────────────────────────────────────────

func TestCORS_CredentialsDisabled(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		AllowCredentials: false,
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_CredentialsDefaultBehavior(t *testing.T) {
	t.Parallel()

	// AllowCredentials defaults to false (zero value of bool)
	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

// ── Max-Age Header Tests ─────────────────────────────────────────────────────

func TestCORS_MaxAgeHeader_Present(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		MaxAge: 86400,
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_MaxAgeHeader_Zero(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		MaxAge: 0,
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_MaxAgeHeader_Negative(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		MaxAge: -1,
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Max-Age"))
}

// ── Handler Chaining Tests ───────────────────────────────────────────────────

func TestCORS_HandlerChaining_NextIsCalledForNonOptions(t *testing.T) {
	t.Parallel()

	called := false
	handler := CORS(defaultCORSConfig())(recordingHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, called, "next handler must be called for non-OPTIONS requests")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS_HandlerChaining_NextIsSkippedForPreflight(t *testing.T) {
	t.Parallel()

	called := false
	handler := CORS(defaultCORSConfig())(recordingHandler(&called))
	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, called, "next handler must NOT be called for OPTIONS preflight")
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestCORS_HandlerChaining_VariousMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method       string
		expectCalled bool
	}{
		{http.MethodGet, true},
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodPatch, true},
		{http.MethodHead, true},
		{http.MethodOptions, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			called := false
			h := CORS(defaultCORSConfig())(recordingHandler(&called))
			req := httptest.NewRequest(tt.method, "/", nil)
			req.Header.Set("Origin", "https://example.com")
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if tt.expectCalled {
				assert.True(t, called, "next handler should be called for %s", tt.method)
			} else {
				assert.False(t, called, "next handler should NOT be called for %s", tt.method)
			}
		})
	}
}

// ── Expose Headers Tests ─────────────────────────────────────────────────────

func TestCORS_ExposeHeaders(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		ExposeHeaders: "X-Custom, X-Another",
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "X-Custom, X-Another", rec.Header().Get("Access-Control-Expose-Headers"))
}

func TestCORS_ExposeHeaders_Default(t *testing.T) {
	t.Parallel()

	handler := CORS(defaultCORSConfig())(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "Grpc-Status, Grpc-Message", rec.Header().Get("Access-Control-Expose-Headers"))
}

// ── Custom Methods / Headers Tests ───────────────────────────────────────────

func TestCORS_CustomAllowedMethods(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		AllowedMethods: "GET, POST, PATCH",
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "GET, POST, PATCH", rec.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORS_CustomAllowedHeaders(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://example.com": true,
		},
		AllowedHeaders: "X-Custom-Header, Authorization",
	}
	handler := CORS(cfg)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "X-Custom-Header, Authorization", rec.Header().Get("Access-Control-Allow-Headers"))
}

// ── Default Origins Helper Test ──────────────────────────────────────────────

func TestDefaultCORSAllowedOrigins(t *testing.T) {
	t.Parallel()

	origins := DefaultCORSAllowedOrigins()

	assert.True(t, origins["http://localhost:5173"])
	assert.True(t, origins["http://localhost:3000"])
	assert.True(t, origins["http://localhost:8081"])
	assert.True(t, origins["http://localhost:5174"])
	assert.Len(t, origins, 4)
}

// ── Full Integration Scenario ────────────────────────────────────────────────

func TestCORS_FullIntegration(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: map[string]bool{
			"https://trusted.com": true,
		},
		AllowedMethods:   "GET, POST",
		AllowedHeaders:   "Content-Type, X-Token",
		ExposeHeaders:    "X-Response-Id",
		AllowCredentials: true,
		MaxAge:           3600,
	}

	handler := CORS(cfg)(okHandler())

	t.Run("trusted GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://trusted.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "https://trusted.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "GET, POST", rec.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, X-Token", rec.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "X-Response-Id", rec.Header().Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "3600", rec.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("untrusted GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://untrusted.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("trusted OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://trusted.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "https://trusted.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
