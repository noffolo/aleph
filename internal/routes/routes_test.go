package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	warns []string
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.warns = append(m.warns, msg)
}

func TestCORSHandler_AllowedOrigin(t *testing.T) {
	mockLogger := &mockLogger{}
	allowedOrigins := []string{"https://example.com", "https://app.example.com"}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSHandler_DisallowedOrigin(t *testing.T) {
	mockLogger := &mockLogger{}
	allowedOrigins := []string{"https://example.com"}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		assert.NotEqual(t, "https://evil.com", origin)
	}
}

func TestCORSHandler_OptionsRequest(t *testing.T) {
	mockLogger := &mockLogger{}
	allowedOrigins := []string{"https://example.com"}

	handler := CORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), allowedOrigins, mockLogger)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestSetDraining(t *testing.T) {
	SetDraining(true)
	SetDraining(false)
}

func TestProjectsRoot(t *testing.T) {
	root, err := ProjectsRoot()
	assert.NoError(t, err, "ProjectsRoot should not error")
	assert.NotEmpty(t, root, "ProjectsRoot should return non-empty path")
	assert.True(t, strings.Contains(root, "data") || strings.Contains(root, "projects"), 
		"ProjectsRoot should contain expected path components")
}
