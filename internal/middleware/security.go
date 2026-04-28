package middleware

import (
	"net/http"
)

// SecurityHeaders is HTTP middleware that adds security-related headers
// to all responses to protect against common web vulnerabilities.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")

		csp := "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self' ws://localhost:*;"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}