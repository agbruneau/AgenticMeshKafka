package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds the configuration for tracing
type TracerConfig struct {
	ServiceName string
	Endpoint    string // OTLP HTTP endpoint (e.g., "http://localhost:4318")
	Insecure    bool
}

// Validate checks if the tracer configuration is valid
func (c *TracerConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	return nil
}

// InitTracer initializes the OpenTelemetry tracer
// Returns a shutdown function to flush and close the tracer
func InitTracer(serviceName, endpoint string) (func(context.Context) error, error) {
	cfg := TracerConfig{
		ServiceName: serviceName,
		Endpoint:    endpoint,
	}
	return InitTracerWithConfig(cfg)
}

// InitTracerWithConfig initializes the tracer with full configuration
func InitTracerWithConfig(cfg TracerConfig) (func(context.Context) error, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	var tp *sdktrace.TracerProvider

	if cfg.Endpoint != "" {
		// Create OTLP HTTP exporter
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}

		exporter, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter: %w", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
	} else {
		// Use noop tracer provider when no endpoint is configured
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
	}

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}

// StartSpan starts a new span with the given name
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("")
	return tracer.Start(ctx, name)
}

// StartSpanWithTracer starts a new span using a specific tracer
func StartSpanWithTracer(ctx context.Context, tracerName, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName)
}

// InjectTraceContext injects the trace context into a map of headers
func InjectTraceContext(ctx context.Context, headers map[string]string) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(headers))
}

// ExtractTraceContext extracts the trace context from a map of headers
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.MapCarrier(headers))
}

// AddSpanAttributes adds attributes to the current span in the context
func AddSpanAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	var otelAttrs []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			otelAttrs = append(otelAttrs, attribute.String(k, val))
		case int:
			otelAttrs = append(otelAttrs, attribute.Int(k, val))
		case int64:
			otelAttrs = append(otelAttrs, attribute.Int64(k, val))
		case float64:
			otelAttrs = append(otelAttrs, attribute.Float64(k, val))
		case bool:
			otelAttrs = append(otelAttrs, attribute.Bool(k, val))
		case []string:
			otelAttrs = append(otelAttrs, attribute.StringSlice(k, val))
		}
	}

	span.SetAttributes(otelAttrs...)
}

// RecordSpanError records an error on the current span
func RecordSpanError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}
}

// GetTraceID returns the trace ID from the context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID returns the span ID from the context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().SpanID().String()
}
