package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders_Production(t *testing.T) {
	t.Parallel()

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(false)(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Production mode: strict CSP, no localhost exceptions.
	tests := []struct {
		name        string
		header      string
		want        string
		contains    bool
		notContains string
	}{
		{"Strict-Transport-Security", "Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload", false, ""},
		{"X-Content-Type-Options", "X-Content-Type-Options", "nosniff", false, ""},
		{"X-Frame-Options", "X-Frame-Options", "DENY", false, ""},
		{"X-XSS-Protection", "X-XSS-Protection", "1; mode=block", false, ""},
		{"Referrer-Policy", "Referrer-Policy", "strict-origin-when-cross-origin", false, ""},
		{"Permissions-Policy", "Permissions-Policy", "geolocation=(), microphone=(), camera=()", false, ""},
		{"Cross-Origin-Opener-Policy", "Cross-Origin-Opener-Policy", "same-origin", false, ""},
		{"Cross-Origin-Resource-Policy", "Cross-Origin-Resource-Policy", "same-origin", false, ""},
		{"CSP has default-src self", "Content-Security-Policy", "default-src 'self'", true, ""},
		{"CSP has script-src self", "Content-Security-Policy", "script-src 'self'", true, ""},
		{"CSP no unsafe-inline", "Content-Security-Policy", "", true, "unsafe-inline"},
		{"CSP no unsafe-eval", "Content-Security-Policy", "", true, "unsafe-eval"},
		{"CSP no ws://localhost", "Content-Security-Policy", "", true, "ws://localhost"},
		{"CSP has frame-ancestors none", "Content-Security-Policy", "frame-ancestors 'none'", true, ""},
		{"CSP has object-src none", "Content-Security-Policy", "object-src 'none'", true, ""},
		{"CSP has upgrade-insecure-requests", "Content-Security-Policy", "upgrade-insecure-requests", true, ""},
		{"CSP has block-all-mixed-content", "Content-Security-Policy", "block-all-mixed-content", true, ""},
		{"CSP has worker-src self", "Content-Security-Policy", "worker-src 'self'", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resp.Header.Get(tt.header)
			if got == "" {
				t.Errorf("header %s not set", tt.header)
				return
			}
			if tt.notContains != "" {
				if strings.Contains(got, tt.notContains) {
					t.Errorf("header %s = %q, must NOT contain %q", tt.header, got, tt.notContains)
				}
				return
			}
			if tt.contains {
				if !strings.Contains(got, tt.want) {
					t.Errorf("header %s = %q, want to contain %q", tt.header, got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
				}
			}
		})
	}
}

func TestSecurityHeaders_DevMode(t *testing.T) {
	t.Parallel()

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(true)(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("CSP header not set in dev mode")
	}

	// Dev mode must allow Vite HMR WebSocket connections on localhost.
	if !strings.Contains(csp, "ws://localhost:*") {
		t.Errorf("dev CSP missing ws://localhost:*, got %q", csp)
	}
	if !strings.Contains(csp, "http://localhost:*") {
		t.Errorf("dev CSP missing http://localhost:*, got %q", csp)
	}

	// Dev mode must NOT have upgrade-insecure-requests (Vite dev runs over HTTP).
	if strings.Contains(csp, "upgrade-insecure-requests") {
		t.Errorf("dev CSP must NOT contain upgrade-insecure-requests, got %q", csp)
	}

	// No unsafe-inline or unsafe-eval in dev mode either.
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("dev CSP must NOT contain unsafe-inline, got %q", csp)
	}
	if strings.Contains(csp, "unsafe-eval") {
		t.Errorf("dev CSP must NOT contain unsafe-eval, got %q", csp)
	}

	// Core security headers must still be present in dev mode.
	headersToCheck := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
	}
	for hdr, want := range headersToCheck {
		got := resp.Header.Get(hdr)
		if got != want {
			t.Errorf("dev mode header %s = %q, want %q", hdr, got, want)
		}
	}
}
