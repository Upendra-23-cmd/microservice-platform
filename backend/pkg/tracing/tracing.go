// Package tracing sets up OpenTelemetry distributed tracing with OTLP export.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Provider wraps the OpenTelemetry TracerProvider and exposes Shutdown.
type Provider struct {
	tp     *sdktrace.TracerProvider
	logger *zap.Logger
}

// Init initialises the global OTEL tracer provider.
// Call Shutdown(ctx) on graceful termination to flush pending spans.
func Init(ctx context.Context, serviceName, serviceVersion, endpoint string, logger *zap.Logger) (*Provider, error) {
	conn, err := grpc.DialContext(ctx, endpoint, //nolint:staticcheck
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: connect to OTLP endpoint %q: %w", endpoint, err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("tracing: create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))), // 10% sampling in prod
	)

	// Set global provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("tracing: initialised",
		zap.String("service", serviceName),
		zap.String("endpoint", endpoint),
	)

	return &Provider{tp: tp, logger: logger}, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Shutdown flushes buffered spans and releases resources.
func (p *Provider) Shutdown(ctx context.Context) error {
	if err := p.tp.Shutdown(ctx); err != nil {
		p.logger.Error("tracing: shutdown error", zap.Error(err))
		return err
	}
	p.logger.Info("tracing: shutdown complete")
	return nil
}
