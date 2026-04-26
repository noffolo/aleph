package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const defaultEndpoint = "localhost:4317"

// Config holds OpenTelemetry configuration.
type Config struct {
	// ServiceName is the name of the service.
	ServiceName string
	// Endpoint is the OTLP gRPC endpoint (optional).
	// If empty, uses OTEL_EXPORTER_OTLP_ENDPOINT env var or falls back to noop.
	Endpoint string
	// Disabled indicates whether telemetry is disabled.
	Disabled bool
}

// InitTelemetry initializes OpenTelemetry tracing and metrics.
// Returns a shutdown function that should be called on application exit.
// If no endpoint is configured, it uses noop providers so the app works without OTel.
func InitTelemetry(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.Disabled {
		slog.Info("OpenTelemetry disabled by config")
		return func(context.Context) error { return nil }, nil
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if serviceName == "" {
		serviceName = "aleph-v2"
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if endpoint == "" {
		// No endpoint configured — use noop provider so app works without OTel.
		slog.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set, using noop telemetry provider")
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		otel.SetMeterProvider(sdkmetric.NewMeterProvider()) // no exporters = effectively noop
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return func(context.Context) error { return nil }, nil
	}

	slog.Info("initializing OpenTelemetry", "service", serviceName, "endpoint", endpoint)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	var shutdowns []func(context.Context) error

	// Setup traces.
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		slog.Warn("failed to create trace exporter, continuing without tracing", "error", err)
	} else {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0))),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	// Setup metrics.
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		slog.Warn("failed to create metric exporter, continuing without metrics", "error", err)
	} else {
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
				sdkmetric.WithInterval(30*time.Second),
			)),
		)
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		slog.Info("shutting down OpenTelemetry")
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		var first error
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil && first == nil {
				first = err
			}
		}
		return first
	}

	return shutdown, nil
}

// Tracer returns a tracer for the given name.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// HTTPRequestAttributes creates attributes for an HTTP request.
func HTTPRequestAttributes(method, path, userAgent string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.url", path),
		attribute.String("http.user_agent", userAgent),
	}
}

// HTTPResponseAttributes creates attributes for an HTTP response.
func HTTPResponseAttributes(statusCode int, contentLength int64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("http.status_code", statusCode),
		attribute.Int64("http.response_content_length", contentLength),
	}
}

// ComponentAttributes creates attributes for a component.
func ComponentAttributes(componentName, componentType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("component.name", componentName),
		attribute.String("component.type", componentType),
	}
}

// ProjectAttributes creates attributes for a project.
func ProjectAttributes(projectID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("project.id", projectID),
	}
}

// AgentAttributes creates attributes for an agent.
func AgentAttributes(agentID, provider string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("agent.id", agentID),
		attribute.String("agent.provider", provider),
	}
}

// ErrorAttribute creates an error attribute.
func ErrorAttribute(err error) attribute.KeyValue {
	if err == nil {
		return attribute.String("error", "nil")
	}
	return attribute.String("error", err.Error())
}
