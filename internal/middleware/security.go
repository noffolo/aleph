package middleware

import (
	"net/http"
)

// SecurityHeaders is HTTP middleware that adds security-related headers
// to all responses to protect against common web vulnerabilities.
// CSP is strict: no unsafe-inline, no unsafe-eval, no websocket localhost.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS — force HTTPS for 1 year, include subdomains, allow preload
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content type and framing protection
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer and permissions
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Cross-origin isolation
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

		// CSP — strict policy, NO unsafe-inline, NO ws://localhost
		csp := "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"object-src 'none'; " +
			"worker-src 'self'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"upgrade-insecure-requests; " +
			"block-all-mixed-content"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}