package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFProtection_GetPassesThrough(t *testing.T) {
	t.Parallel()
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

func TestCSRFProtection_OptionsPassesThrough(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS, got %d", rec.Code)
	}
}

func TestCSRFProtection_ValidOriginPasses(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestCSRFProtection_OriginPrefixRejectsPartialMatch(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Origin", "http://localhost:5173.evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for partial origin match, got %d", rec.Code)
	}
}

func TestCSRFProtection_ValidRefererPasses(t *testing.T) {
	t.Parallel()
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

func TestCSRFProtection_RefererDifferentPortRejected(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Referer", "http://localhost:8080/some-page")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for referer with wrong port, got %d", rec.Code)
	}
}

func TestCSRFProtection_NoOriginNoRefererRejected(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for POST with no Origin/Referer, got %d", rec.Code)
	}
}

func TestCSRFProtection_PutRequiresOrigin(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/resource", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for PUT with valid origin, got %d", rec.Code)
	}
}

func TestCSRFProtection_DeleteRequiresOrigin(t *testing.T) {
	t.Parallel()
	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/resource", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for DELETE with invalid origin, got %d", rec.Code)
	}
}