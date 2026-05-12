package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// CSRFProtection validates Origin/Referer headers on mutating requests to prevent CSRF.
// Safe methods (GET/HEAD/OPTIONS) pass through. Mutating methods require an Origin or
// Referer that exactly matches an allowed origin (scheme+host, no path prefix matching).
func CSRFProtection(allowedOrigins []string) func(http.Handler) http.Handler {
	originMap := make(map[string]bool, len(allowedOrigins))
	hostMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originMap[o] = true
		if u, err := url.Parse(o); err == nil {
			hostMap[u.Host] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Session creation must work without Origin (first request, no cookie yet)
			if r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/session" {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			referer := r.Header.Get("Referer")

			if origin == "" && referer == "" {
				http.Error(w, "CSRF validation failed: missing Origin and Referer", http.StatusForbidden)
				return
			}

			if origin != "" {
				if originMap[origin] {
					next.ServeHTTP(w, r)
					return
				}
			}

			if referer != "" {
				u, err := url.Parse(referer)
				if err == nil && u.Host != "" && hostMap[u.Host] {
					next.ServeHTTP(w, r)
					return
				}
				for _, allowed := range allowedOrigins {
					if strings.EqualFold(u.String(), allowed) || strings.HasPrefix(strings.ToLower(referer), strings.ToLower(allowed+"/")) {
						if parsed, pErr := url.Parse(allowed); pErr == nil && parsed.Host == u.Host && parsed.Scheme == u.Scheme {
							next.ServeHTTP(w, r)
							return
						}
					}
				}
			}

			http.Error(w, "CSRF validation failed", http.StatusForbidden)
		})
	}
}
