package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery_PanicReturns500(t *testing.T) {
	t.Parallel()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic: something went wrong")
	})

	wrapped := Recovery(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}

	if body["error"] != "internal server error" {
		t.Errorf("expected error message 'internal server error', got %q", body["error"])
	}

	if body["code"] != "internal_error" {
		t.Errorf("expected code 'internal_error', got %q", body["code"])
	}

	// Verify panic details are NOT leaked in the response
	if body["stack"] != "" || body["panic"] != "" {
		t.Error("panic details leaked in response body")
	}
}

func TestRecovery_NormalHandlerPassesThrough(t *testing.T) {
	t.Parallel()

	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	})

	wrapped := Recovery(normalHandler)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRecovery_PanicAfterPartialWrite(t *testing.T) {
	t.Parallel()

	// Handler that writes partial response then panics
	panicAfterWriteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // partial write
		w.Write([]byte(`{"partial": true}`))
		panic("panic after partial write")
	})

	wrapped := Recovery(panicAfterWriteHandler)

	req := httptest.NewRequest(http.MethodGet, "/partial", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// If headers already sent, we can't change the status code anymore.
	// The key requirement: the server doesn't crash.
	// Accept whatever status we already sent (202 Accepted in this case).
	if resp.StatusCode == http.StatusInternalServerError {
		t.Log("recovery managed to override status to 500")
	} else {
		t.Logf("recovery could not override already-written status %d", resp.StatusCode)
		// This is acceptable behavior — once headers are flushed, we can't change them.
	}
}
