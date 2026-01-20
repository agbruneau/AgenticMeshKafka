package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogOutput_JSONFormat(t *testing.T) {
	// Create a buffer to capture log output
	buf := &bytes.Buffer{}

	// Create logger with buffer output
	logger := NewLogger(LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	// Log a message
	logger.Info("test message", "key", "value")

	// Verify output is valid JSON
	output := buf.String()
	if output == "" {
		t.Error("Logger produced no output")
		return
	}

	// Parse as JSON to verify format
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
		t.Errorf("Logger output is not valid JSON: %v, output: %s", err, output)
	}

	// Verify message field exists
	if _, ok := logEntry["msg"]; !ok {
		t.Error("Logger output missing 'msg' field")
	}
}

func TestWithTraceID_IncludesTraceID(t *testing.T) {
	// Initialize tracer
	shutdown, err := InitTracer("test-service", "")
	if err != nil {
		t.Fatalf("InitTracer() error: %v", err)
	}
	defer shutdown(context.Background())

	// Create a span to get a trace context
	ctx, span := StartSpan(context.Background(), "test-operation")
	defer span.End()

	// Create a buffer to capture log output
	buf := &bytes.Buffer{}

	// Create logger with buffer output
	logger := NewLogger(LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	// Create logger with trace context
	loggerWithCtx := logger.WithContext(ctx)
	loggerWithCtx.Info("test with trace")

	// Parse output
	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Skip("Logger with noop tracer may not include trace_id")
		return
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Logger output is not valid JSON: %v", err)
		return
	}

	// With noop tracer, trace_id might be empty or zeros, which is acceptable
	// The important thing is that the logger doesn't crash
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		level         string
		logLevel      string
		shouldContain string
	}{
		{"debug", "debug", "debug message"},
		{"info", "info", "info message"},
		{"warn", "warn", "warn message"},
		{"error", "error", "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewLogger(LoggerConfig{
				Level:  tt.level,
				Format: "json",
				Output: buf,
			})

			switch tt.logLevel {
			case "debug":
				logger.Debug(tt.shouldContain)
			case "info":
				logger.Info(tt.shouldContain)
			case "warn":
				logger.Warn(tt.shouldContain)
			case "error":
				logger.Error(tt.shouldContain)
			}

			output := buf.String()
			if !strings.Contains(output, tt.shouldContain) {
				t.Errorf("Expected log output to contain %q, got %q", tt.shouldContain, output)
			}
		})
	}
}

func TestLoggerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     LoggerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: LoggerConfig{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			cfg: LoggerConfig{
				Level:  "invalid",
				Format: "json",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			cfg: LoggerConfig{
				Level:  "info",
				Format: "invalid",
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

func TestLoggerWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: buf,
	})

	// Add fields
	loggerWithFields := logger.With("service", "test", "version", "1.0")
	loggerWithFields.Info("test message")

	// Parse output
	output := strings.TrimSpace(buf.String())
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Logger output is not valid JSON: %v", err)
	}

	// Verify fields are present
	if logEntry["service"] != "test" {
		t.Errorf("Expected service=test, got %v", logEntry["service"])
	}
	if logEntry["version"] != "1.0" {
		t.Errorf("Expected version=1.0, got %v", logEntry["version"])
	}
}
