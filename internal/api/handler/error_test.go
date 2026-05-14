package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, "something bad happened", http.StatusBadRequest)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	body := rec.Body.String()
	if body != "{\"error\":\"something bad happened\"}\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestWriteError_InternalServerError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, "boom", http.StatusInternalServerError)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestWriteError_NotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, "not found", http.StatusNotFound)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]interface{}{"id": 42, "name": "test"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	body := rec.Body.String()
	if body != "{\"id\":42,\"name\":\"test\"}\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestWriteJSON_Created(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"status": "created"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}
