package observability

import (
	"context"
	"testing"
)

func TestStartSpan_CreatesSpan(t *testing.T) {
	// Initialize tracer (noop for tests)
	shutdown, err := InitTracer("test-service", "")
	if err != nil {
		t.Fatalf("InitTracer() error: %v", err)
	}
	defer shutdown(context.Background())

	// Start a span
	ctx, span := StartSpan(context.Background(), "test-operation")
	if span == nil {
		t.Error("StartSpan() returned nil span")
	}
	if ctx == nil {
		t.Error("StartSpan() returned nil context")
	}

	span.End()
}

func TestInjectExtractTraceContext_RoundTrip(t *testing.T) {
	// Initialize tracer
	shutdown, err := InitTracer("test-service", "")
	if err != nil {
		t.Fatalf("InitTracer() error: %v", err)
	}
	defer shutdown(context.Background())

	// Start a span to get a trace context
	ctx, span := StartSpan(context.Background(), "test-operation")
	defer span.End()

	// Inject trace context into headers
	headers := make(map[string]string)
	InjectTraceContext(ctx, headers)

	// Headers should contain trace information (at minimum, we shouldn't panic)
	// Note: With noop exporter, headers might be empty which is acceptable

	// Extract trace context from headers
	extractedCtx := ExtractTraceContext(context.Background(), headers)
	if extractedCtx == nil {
		t.Error("ExtractTraceContext() returned nil context")
	}
}

func TestTracerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TracerConfig
		wantErr bool
	}{
		{
			name: "valid config with endpoint",
			cfg: TracerConfig{
				ServiceName: "test-service",
				Endpoint:    "http://localhost:4318",
			},
			wantErr: false,
		},
		{
			name: "valid config without endpoint (noop)",
			cfg: TracerConfig{
				ServiceName: "test-service",
				Endpoint:    "",
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			cfg: TracerConfig{
				ServiceName: "",
				Endpoint:    "http://localhost:4318",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSpanWithAttributes(t *testing.T) {
	shutdown, err := InitTracer("test-service", "")
	if err != nil {
		t.Fatalf("InitTracer() error: %v", err)
	}
	defer shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-with-attributes")
	defer span.End()

	// Add attributes - should not panic
	AddSpanAttributes(ctx, map[string]interface{}{
		"string_attr": "value",
		"int_attr":    42,
		"bool_attr":   true,
	})

	// Record error - should not panic
	RecordSpanError(ctx, nil)
}
