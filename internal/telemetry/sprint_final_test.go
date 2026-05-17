package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestRecordError_WithRealSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := InstrumentFunction(context.Background(), "test_recording")
	defer span.End()

	RecordError(ctx, errors.New("test error"), attribute.String("detail", "info"))
}

func TestWithSpanAttributes_WithRealSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := InstrumentFunction(context.Background(), "test_attrs")
	defer span.End()

	WithSpanAttributes(ctx, attribute.String("k", "v"), attribute.Bool("ok", true))
}

func TestLogWithTrace_WithRealSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := InstrumentFunction(context.Background(), "test_log")
	defer span.End()

	LogWithTrace(ctx, slog.LevelInfo, "trace test")
}

func TestInitTelemetry_WithEndpoint(t *testing.T) {
	cfg := Config{
		ServiceName: "test-e2e",
		Endpoint:    "0.1.2.3:4317",
	}
	shutdown, err := InitTelemetry(context.Background(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	_ = shutdown(context.Background())
}
