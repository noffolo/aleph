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
	}{
		{
			name:   "X-Content-Type-Options",
			header: "X-Content-Type-Options",
			want:   "nosniff",
		},
		{
			name:   "X-Frame-Options",
			header: "X-Frame-Options",
			want:   "DENY",
		},
		{
			name:   "Referrer-Policy",
			header: "Referrer-Policy",
			want:   "same-origin",
		},
		{
			name:     "Content-Security-Policy",
			header:   "Content-Security-Policy",
			want:     "default-src 'self'",
			contains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resp.Header.Get(tt.header)
			if got == "" {
				t.Errorf("header %s not set", tt.header)
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