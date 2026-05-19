package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ff3300/aleph-v2/internal/api/handler"
	"github.com/ff3300/aleph-v2/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes_LivezUnhealthy(t *testing.T) {
	mux := http.NewServeMux()
	cfg := testConfig()
	cfg.HealthCheckFunc = func(ctx context.Context) error {
		return errors.New("db connection refused")
	}
	RegisterRoutes(mux, cfg)

	req := httptest.NewRequest("GET", "/livez", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), `"status":"unhealthy"`)
	assert.Contains(t, rr.Body.String(), `"reason":"db connection refused"`)
}

func TestRegisterRoutes_HealthzUnhealthy(t *testing.T) {
	mux := http.NewServeMux()
	cfg := testConfig()
	cfg.HealthCheckFunc = func(ctx context.Context) error {
		return errors.New("database timeout")
	}
	RegisterRoutes(mux, cfg)

	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), `"status":"unhealthy"`)
	assert.Contains(t, rr.Body.String(), `"reason":"database timeout"`)
}

func testConfigWithSession() RegisterConfig {
	cfg := testConfig()
	cfg.SessionHandler = &handler.SessionHandler{}
	cfg.AuthRateLimiter = middleware.NewAuthRateLimiter(nil, middleware.AuthRateLimitConfig{
		SessionCreateLimit:  5,
		SessionCreateWindow: time.Minute,
	})
	return cfg
}

func TestRegisterRoutes_SessionDeleteNoCookie(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testConfigWithSession())

	req := httptest.NewRequest("DELETE", "/api/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}

func TestRegisterRoutes_SessionGetNoCookie(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testConfigWithSession())

	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "no session")
}

func TestRegisterRoutes_SessionDefaultMethod(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testConfigWithSession())

	req := httptest.NewRequest("PUT", "/api/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Contains(t, rr.Body.String(), "method not allowed")
}

func TestRegisterRoutes_SessionOptionsMethod(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, testConfigWithSession())

	req := httptest.NewRequest("OPTIONS", "/api/v1/auth/session", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}


