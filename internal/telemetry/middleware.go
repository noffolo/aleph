package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "aleph.v2.http"

var (
	httpRequestDuration metric.Float64Histogram
	httpRequestCount    metric.Int64Counter
	httpRequestSize     metric.Int64Histogram
	httpResponseSize    metric.Int64Histogram
)

func initMetrics() error {
	meter := otel.GetMeterProvider().Meter("aleph.v2.http")

	var err error
	httpRequestDuration, err = meter.Float64Histogram(
		"http.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return fmt.Errorf("create http.request.duration histogram: %w", err)
	}

	httpRequestCount, err = meter.Int64Counter(
		"http.request.count",
		metric.WithDescription("Total HTTP requests"),
	)
	if err != nil {
		return fmt.Errorf("create http.request.count counter: %w", err)
	}

	httpRequestSize, err = meter.Int64Histogram(
		"http.request.size",
		metric.WithUnit("bytes"),
		metric.WithDescription("HTTP request size in bytes"),
	)
	if err != nil {
		return fmt.Errorf("create http.request.size histogram: %w", err)
	}

	httpResponseSize, err = meter.Int64Histogram(
		"http.response.size",
		metric.WithUnit("bytes"),
		metric.WithDescription("HTTP response size in bytes"),
	)
	if err != nil {
		return fmt.Errorf("create http.response.size histogram: %w", err)
	}

	return nil
}

func Middleware(next http.Handler) http.Handler {
	if err := initMetrics(); err != nil {
		slog.Warn("failed to initialize HTTP metrics, proceeding without telemetry", "error", err)
		return next
	}

	tracer := Tracer(tracerName)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		spanCtx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(HTTPRequestAttributes(r.Method, r.URL.Path, r.UserAgent())...),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		if r.ContentLength > 0 {
			httpRequestSize.Record(spanCtx, r.ContentLength)
		}

		r = r.WithContext(spanCtx)
		next.ServeHTTP(rw, r)

		duration := float64(time.Since(start)) / float64(time.Second)

		attributes := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.Int("http.status_code", rw.statusCode),
		}

		httpRequestDuration.Record(spanCtx, duration, metric.WithAttributes(attributes...))
		httpRequestCount.Add(spanCtx, 1, metric.WithAttributes(attributes...))
		httpResponseSize.Record(spanCtx, int64(rw.size))

		span.SetAttributes(HTTPResponseAttributes(rw.statusCode, int64(rw.size))...)

		if rw.statusCode >= 500 {
			span.SetAttributes(attribute.String("error.type", "http.server_error"))
		}
	})
}

type headerCarrier struct {
	headers http.Header
}

func (c headerCarrier) Get(key string) string {
	return c.headers.Get(key)
}

func (c headerCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for k := range c.headers {
		keys = append(keys, k)
	}
	return keys
}

func HeaderCarrier(h http.Header) headerCarrier {
	return headerCarrier{headers: h}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	size        int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	rw.size += len(b)
	return rw.ResponseWriter.Write(b)
}

func ConnectRPCMiddleware() func(http.Handler) http.Handler {
	return Middleware
}

func WrapHandler(handler http.Handler, handlerName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tracer := Tracer("aleph.v2.connectrpc")

		spanCtx, span := tracer.Start(ctx, handlerName,
			trace.WithAttributes(
				attribute.String("handler.name", handlerName),
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		r = r.WithContext(spanCtx)
		handler.ServeHTTP(w, r)
	})
}

func InstrumentFunction(ctx context.Context, funcName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := Tracer("aleph.v2.function")
	allAttrs := append([]attribute.KeyValue{
		attribute.String("function.name", funcName),
	}, attrs...)
	return tracer.Start(ctx, funcName,
		trace.WithAttributes(allAttrs...),
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

func WithSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	if err == nil {
		return
	}
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		allAttrs := append([]attribute.KeyValue{ErrorAttribute(err)}, attrs...)
		span.SetAttributes(allAttrs...)
		span.SetStatus(codes.Error, err.Error())
	}
}

func LogWithTrace(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	span := trace.SpanFromContext(ctx)
	args := make([]any, 0, len(attrs)+2)
	if span.IsRecording() {
		args = append(args,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	for _, a := range attrs {
		args = append(args, a)
	}
	slog.Log(ctx, level, msg, args...)
}
