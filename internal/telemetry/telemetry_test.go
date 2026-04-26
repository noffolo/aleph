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
