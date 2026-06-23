// Package otelx initialises OpenTelemetry tracing. When no OTLP endpoint is
// configured it still installs a TracerProvider (with no exporter) and the W3C
// propagator, so application code can always create spans and the returned
// shutdown is safe to call.
package otelx

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config configures the tracer provider.
type Config struct {
	ServiceName  string
	Environment  string
	OTLPEndpoint string // empty => no exporter (spans dropped)
	SampleRatio  float64
	Insecure     bool
}

// ShutdownFunc flushes and releases tracing resources.
type ShutdownFunc func(context.Context) error

// InitTracer installs the global tracer provider + propagator.
func InitTracer(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(cfg.ServiceName),
		semconv.DeploymentEnvironment(cfg.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithResource(res),
	}

	if cfg.OTLPEndpoint != "" {
		expOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
		if cfg.Insecure {
			expOpts = append(expOpts, otlptracegrpc.WithInsecure())
		}
		exporter, err := otlptracegrpc.New(ctx, expOpts...)
		if err != nil {
			return nil, fmt.Errorf("otlp exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown, nil
}
