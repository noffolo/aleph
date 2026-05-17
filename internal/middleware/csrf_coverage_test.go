package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFProtection_SessionCreateBypass(t *testing.T) {
	t.Parallel()

	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("post_to_session_no_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "session creation must bypass CSRF")
	})

	t.Run("post_to_other_endpoint_no_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "other POST must require Origin/Referer")
	})
}

func TestCSRFProtection_HeadPassesThrough(t *testing.T) {
	t.Parallel()

	csrf := CSRFProtection([]string{"http://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodHead, "/api/v1/resource", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFProtection_RefererDifferentHostRejected(t *testing.T) {
	t.Parallel()

	csrf := CSRFProtection([]string{"https://localhost:5173"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
	req.Header.Set("Referer", "http://evil.com/some-page")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFProtection_MultipleOrigins(t *testing.T) {
	t.Parallel()

	csrf := CSRFProtection([]string{"http://localhost:5173", "https://app.example.com"})
	handler := csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{"first origin", "http://localhost:5173", http.StatusOK},
		{"second origin", "https://app.example.com", http.StatusOK},
		{"unknown origin", "https://evil.com", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/query", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
