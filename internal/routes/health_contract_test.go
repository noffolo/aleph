package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints_Contract(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, RegisterConfig{})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	t.Run("/livez returns 200 and status=alive", func(t *testing.T) {
		resp, err := srv.Client().Get(srv.URL + "/livez")
		if err != nil {
			t.Fatalf("GET /livez failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if body["status"] != "alive" {
			t.Errorf("expected status=alive, got %q", body["status"])
		}
	})

	t.Run("/readyz returns 200 and status=ok when not draining", func(t *testing.T) {
		resp, err := srv.Client().Get(srv.URL + "/readyz")
		if err != nil {
			t.Fatalf("GET /readyz failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("expected status=ok, got %q", body["status"])
		}
	})

	t.Run("/readyz returns 503 when draining", func(t *testing.T) {
		SetDraining(true)
		t.Cleanup(func() { SetDraining(false) })

		resp, err := srv.Client().Get(srv.URL + "/readyz")
		if err != nil {
			t.Fatalf("GET /readyz failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if body["status"] != "not ready" {
			t.Errorf("expected status=\"not ready\", got %q", body["status"])
		}
		if body["reason"] != "draining" {
			t.Errorf("expected reason=draining, got %q", body["reason"])
		}
	})
}
