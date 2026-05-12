package middleware

import (
	"net/http"
	"strconv"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a map of allowed origins (map for O(1) lookup).
	// If empty, all origins are blocked (no Access-Control-Allow-Origin set).
	AllowedOrigins map[string]bool

	// AllowedMethods is the value for the Access-Control-Allow-Methods header.
	// Defaults to "GET, POST, PUT, DELETE, OPTIONS".
	AllowedMethods string

	// AllowedHeaders is the value for the Access-Control-Allow-Headers header.
	// Defaults to "Content-Type, Authorization, X-Aleph-Api-Key, X-Request-Id, X-Project-Id".
	AllowedHeaders string

	// ExposeHeaders is the value for the Access-Control-Expose-Headers header.
	// Defaults to "Grpc-Status, Grpc-Message".
	ExposeHeaders string

	// AllowCredentials controls the Access-Control-Allow-Credentials header.
	// When true, sets the header to "true" for requests from allowed origins.
	// Defaults to true.
	AllowCredentials bool

	// MaxAge is the value for the Access-Control-Max-Age header (in seconds).
	// When > 0, sets the header on preflight responses.
	MaxAge int
}

// DefaultCORSAllowedOrigins returns the default set of allowed origins.
func DefaultCORSAllowedOrigins() map[string]bool {
	return map[string]bool{
		"http://localhost:5173": true,
		"http://localhost:3000": true,
		"http://localhost:8081": true,
		"http://localhost:5174": true,
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
//
// The middleware:
//   - Sets Access-Control-Allow-Origin to the request's Origin if it matches the configured allowlist
//   - Sets Access-Control-Allow-Credentials to "true" for allowed origins when AllowCredentials is enabled
//   - Sets Access-Control-Allow-Methods, Allow-Headers, and Expose-Headers on every response
//   - Handles preflight OPTIONS requests by returning 204 with CORS headers (without calling next)
//   - Passes non-OPTIONS requests through to the next handler
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := cfg.AllowedMethods
	if allowedMethods == "" {
		allowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
	}

	allowedHeaders := cfg.AllowedHeaders
	if allowedHeaders == "" {
		allowedHeaders = "Content-Type, Authorization, X-Aleph-Api-Key, X-Request-Id, X-Project-Id"
	}

	exposeHeaders := cfg.ExposeHeaders
	if exposeHeaders == "" {
		exposeHeaders = "Grpc-Status, Grpc-Message"
	}

	origins := cfg.AllowedOrigins
	if origins == nil {
		origins = make(map[string]bool)
	}

	maxAge := cfg.MaxAge

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set Allow-Origin and Credentials only for explicitly allowed origins
			if origin != "" && origins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)

			if maxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
			}

			// Preflight: return 204 without calling next handler
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}


