package sdk

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupOTel configures OpenTelemetry with an OTLP/HTTP exporter.
// endpoint is the CloudMock Cloud ingest URL (e.g. "otel.cloudmock.app:4318")
// or a local CloudMock instance (e.g. "localhost:4318").
// apiKey is the CloudMock API key for authentication.
func SetupOTel(ctx context.Context, endpoint, apiKey, serviceName string) (func(), error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}

	// Use insecure for localhost
	if isLocalhost(endpoint) {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// Add API key header if provided
	if apiKey != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
			"X-Api-Key": apiKey,
		}))
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	shutdown := func() {
		tp.Shutdown(context.Background())
	}
	return shutdown, nil
}

func isLocalhost(endpoint string) bool {
	return len(endpoint) > 0 && (endpoint[0] == 'l' || endpoint[0] == '1')
}
