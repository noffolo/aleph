package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestInitTelemetry_Disabled(t *testing.T) {
	cfg := Config{Disabled: true}
	shutdown, err := InitTelemetry(context.Background(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitTelemetry_NoopEndpoint(t *testing.T) {
	// No endpoint configured, should use noop
	cfg := Config{ServiceName: "test-service"}
	shutdown, err := InitTelemetry(context.Background(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTracer(t *testing.T) {
	tr := Tracer("test.tracer")
	assert.NotNil(t, tr)
}

func TestHTTPRequestAttributes(t *testing.T) {
	attrs := HTTPRequestAttributes("GET", "/api/test", "test-agent")
	assert.Len(t, attrs, 3)
	assert.Equal(t, attribute.String("http.method", "GET"), attrs[0])
	assert.Equal(t, attribute.String("http.url", "/api/test"), attrs[1])
	assert.Equal(t, attribute.String("http.user_agent", "test-agent"), attrs[2])
}

func TestHTTPResponseAttributes(t *testing.T) {
	attrs := HTTPResponseAttributes(200, 1024)
	assert.Len(t, attrs, 2)
	assert.Equal(t, attribute.Int("http.status_code", 200), attrs[0])
	assert.Equal(t, attribute.Int64("http.response_content_length", 1024), attrs[1])
}

func TestComponentAttributes(t *testing.T) {
	attrs := ComponentAttributes("my-component", "service")
	assert.Len(t, attrs, 2)
	assert.Equal(t, attribute.String("component.name", "my-component"), attrs[0])
	assert.Equal(t, attribute.String("component.type", "service"), attrs[1])
}

func TestProjectAttributes(t *testing.T) {
	attrs := ProjectAttributes("proj-123")
	assert.Len(t, attrs, 1)
	assert.Equal(t, attribute.String("project.id", "proj-123"), attrs[0])
}

func TestAgentAttributes(t *testing.T) {
	attrs := AgentAttributes("agent-1", "anthropic")
	assert.Len(t, attrs, 2)
	assert.Equal(t, attribute.String("agent.id", "agent-1"), attrs[0])
	assert.Equal(t, attribute.String("agent.provider", "anthropic"), attrs[1])
}

func TestErrorAttribute(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		attr := ErrorAttribute(errors.New("something went wrong"))
		assert.Equal(t, attribute.String("error", "something went wrong"), attr)
	})

	t.Run("with nil", func(t *testing.T) {
		attr := ErrorAttribute(nil)
		assert.Equal(t, attribute.String("error", "nil"), attr)
	})
}

func TestHeaderCarrier(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")

	carrier := HeaderCarrier(h)
	assert.Equal(t, "application/json", carrier.Get("Content-Type"))

	carrier.Set("X-Custom", "value")
	assert.Equal(t, "value", h.Get("X-Custom"))

	keys := carrier.Keys()
	assert.Contains(t, keys, "Content-Type")
	assert.Contains(t, keys, "X-Custom")
}

func TestResponseWriter(t *testing.T) {
	t.Run("WriteHeader records status", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, rw.statusCode)
		assert.True(t, rw.wroteHeader)
	})

	t.Run("WriteHeader only once", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		rw.WriteHeader(http.StatusInternalServerError) // should be ignored
		assert.Equal(t, http.StatusNotFound, rw.statusCode)
	})

	t.Run("Write calls WriteHeader with StatusOK", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		n, err := rw.Write([]byte("hello"))
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, http.StatusOK, rw.statusCode)
		assert.True(t, rw.wroteHeader)
		assert.Equal(t, 5, rw.size)
	})
}

func TestWrapHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := WrapHandler(handler, "test_handler")
	assert.NotNil(t, wrapped)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	wrapped.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestInstrumentFunction(t *testing.T) {
	ctx := context.Background()
	ctx, span := InstrumentFunction(ctx, "test_func",
		attribute.String("extra", "value"),
	)
	assert.NotNil(t, span)
	span.End()
}

func TestWithSpanAttributes_NilSpan(t *testing.T) {
	// Should not panic when context has no span
	ctx := context.Background()
	WithSpanAttributes(ctx, attribute.String("key", "value"))
}

func TestRecordError_NilError(t *testing.T) {
	// Should not panic with nil error
	ctx := context.Background()
	RecordError(ctx, nil)
}

func TestLogWithTrace_NoSpan(t *testing.T) {
	// Should not panic when context has no recording span
	ctx := context.Background()
	LogWithTrace(ctx, 0, "test message")
}

func TestMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw := Middleware(handler)
	assert.NotNil(t, mw)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/test", nil)
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConnectRPCMiddleware(t *testing.T) {
	mw := ConnectRPCMiddleware()
	assert.NotNil(t, mw)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/rpc", nil)
	mw(handler).ServeHTTP(w, r)
}

// ── PrometheusMiddleware tests ──────────────────────────────────────────

func TestPrometheusMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw := PrometheusMiddleware(handler)
	assert.NotNil(t, mw)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/test", nil)
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestPrometheusMiddleware_500(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	mw := PrometheusMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/error", nil)
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPrometheusMiddleware_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mw := PrometheusMiddleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/nonexistent", nil)
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMetricsHandler(t *testing.T) {
	h := MetricsHandler()
	assert.NotNil(t, h)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/metrics", nil)
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "aleph_")
}

// ── promResponseWriter tests ─────────────────────────────────────────────

func TestPromResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &promResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.True(t, rw.wroteHeader)

	// second write should be ignored
	rw.WriteHeader(http.StatusInternalServerError)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
}

func TestPromResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &promResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	n, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.True(t, rw.wroteHeader)
}

func TestPromResponseWriter_WriteAfterWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &promResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusCreated)

	n, err := rw.Write([]byte("data"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
}

// ── responseWriter.Flush test ─────────────────────────────────────────────

func TestResponseWriter_Flush(t *testing.T) {
	// httptest.ResponseRecorder implements http.Flusher
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}
	rw.Flush() // should not panic
	assert.True(t, w.Flushed)
}

// ── PrometheusMiddleware: multiple requests ──────────────────────────────

func TestPrometheusMiddleware_MultipleRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := PrometheusMiddleware(handler)

	paths := []string{"/api/a", "/api/b", "/api/c"}
	for _, path := range paths {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		mw.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// ── Middleware edge cases ─────────────────────────────────────────────────

func TestMiddleware_500Status(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	mw := Middleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/error", nil)
	r.ContentLength = 0
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMiddleware_WithContentLength(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw := Middleware(handler)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api", nil)
	r.ContentLength = 256
	mw.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWriter_Flush_WithFlusher(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w}
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("test"))
	rw.Flush()
	assert.True(t, w.Flushed)
}

// ── WrapHandler edge cases ────────────────────────────────────────────────

func TestWrapHandler_Non200Status(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := WrapHandler(handler, "error_handler")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/bad", nil)
	wrapped.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── InstrumentFunction with multiple attrs ────────────────────────────────

func TestInstrumentFunction_MultipleAttrs(t *testing.T) {
	ctx := context.Background()
	ctx, span := InstrumentFunction(ctx, "multi_func",
		attribute.String("k1", "v1"),
		attribute.Bool("k2", true),
		attribute.Int("k3", 42),
	)
	assert.NotNil(t, span)
	span.End()
}

// ── WithSpanAttributes with recording span ────────────────────────────────

func TestWithSpanAttributes_RecordingSpan(t *testing.T) {
	// Create a context with a recording span from InstrumentFunction
	ctx, span := InstrumentFunction(context.Background(), "test_span")
	defer span.End()

	// This should set attributes on the recording span
	WithSpanAttributes(ctx, attribute.String("test.key", "test.value"))
}

// ── RecordError with recording span ───────────────────────────────────────

func TestRecordError_RecordingSpan(t *testing.T) {
	ctx, span := InstrumentFunction(context.Background(), "test_span")
	defer span.End()

	RecordError(ctx, errors.New("test error"), attribute.String("extra", "info"))
}

func TestRecordError_NilErrorOnRecordingSpan(t *testing.T) {
	ctx, span := InstrumentFunction(context.Background(), "test_span")
	defer span.End()

	// Should not panic with nil error and recording span
	RecordError(ctx, nil, attribute.String("extra", "info"))
}

// ── LogWithTrace with recording span ──────────────────────────────────────

func TestLogWithTrace_RecordingSpan(t *testing.T) {
	ctx, span := InstrumentFunction(context.Background(), "test_span")
	defer span.End()

	LogWithTrace(ctx, 0, "test with trace",
		// attrs are slog.Attr values, but we need to use the right interface
	)
}

// ── HeaderCarrier edge cases ──────────────────────────────────────────────

func TestHeaderCarrier_EmptyHeaders(t *testing.T) {
	h := http.Header{}
	carrier := HeaderCarrier(h)
	assert.Empty(t, carrier.Get("anything"))
	assert.Empty(t, carrier.Keys())
}

// ── Telemetry Config tests ────────────────────────────────────────────────

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}
	assert.Empty(t, cfg.ServiceName)
	assert.Empty(t, cfg.Endpoint)
	assert.False(t, cfg.Disabled)
}

func TestConfig_AllFields(t *testing.T) {
	cfg := Config{
		ServiceName: "test-svc",
		Endpoint:    "localhost:4317",
		Disabled:    true,
	}
	assert.Equal(t, "test-svc", cfg.ServiceName)
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.True(t, cfg.Disabled)
}

func TestInitTelemetry_EmptyServiceName(t *testing.T) {
	// Empty service name, no endpoint → should use "aleph-v2" default and noop
	cfg := Config{}
	shutdown, err := InitTelemetry(context.Background(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	err = shutdown(context.Background())
	assert.NoError(t, err)
}
