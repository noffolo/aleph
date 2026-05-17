package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkMiddlewareChain(b *testing.B) {
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := RequestID(Recovery(CORS(CORSConfig{
		AllowedOrigins:   map[string]bool{"*": true},
		AllowedMethods:   "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	})(noop)))

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, req)
	}
}
