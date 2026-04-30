package middleware

import (
	"net/http"
	"strings"
)

// CSRFProtection validates Origin/Referer headers to prevent CSRF attacks.
// This is sufficient because auth uses X-Aleph-Api-Key header (not cookies).
// If session cookies are used in the future, upgrade to double-submit cookie pattern.
func CSRFProtection(allowedOrigins []string) func(http.Handler) http.Handler {
	originMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originMap[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for GET, HEAD, OPTIONS
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			referer := r.Header.Get("Referer")

			// If neither Origin nor Referer is present, allow (may be CLI/internal client)
			if origin == "" && referer == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check Origin against allowed list
			if origin != "" {
				if originMap[origin] {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Fallback: check Referer origin
			if referer != "" {
				for _, allowed := range allowedOrigins {
					if strings.HasPrefix(referer, allowed) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			http.Error(w, "CSRF validation failed", http.StatusForbidden)
		})
	}
}
