package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_Generated(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rid := GetRequestID(r.Context()); rid == "" {
			t.Error("expected non-empty request ID")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id in response header")
	}
}

func TestRequestID_PropagatesFromHeader(t *testing.T) {
	expected := "test-request-id"
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rid := GetRequestID(r.Context()); rid != expected {
			t.Errorf("expected %s, got %s", expected, rid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", expected)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}