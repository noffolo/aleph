package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFProtection_GetPassesThrough(t *testing.T) {
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCSRFProtection_ValidOriginPasses(t *testing.T) {
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid origin, got %d", rec.Code)
	}
}

func TestCSRFProtection_InvalidOriginRejected(t *testing.T) {
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid origin, got %d", rec.Code)
	}
}

func TestCSRFProtection_ValidRefererPasses(t *testing.T) {
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Referer", "http://localhost:5173/some-page")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid referer, got %d", rec.Code)
	}
}

func TestCSRFProtection_NoOriginOrRefererPasses(t *testing.T) {
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	// No Origin, no Referer — should pass (CLI client)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for CLI client (no origin/referer), got %d", rec.Code)
	}
}
