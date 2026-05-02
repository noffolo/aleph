package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(noopHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	tests := []struct {
		name     string
		header   string
		want     string
		contains bool
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