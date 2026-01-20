package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// LoggerConfig holds the configuration for the logger
type LoggerConfig struct {
	Level  string    // "debug", "info", "warn", "error"
	Format string    // "json" or "text"
	Output io.Writer // Output writer (defaults to os.Stdout)
}

// Validate checks if the logger configuration is valid
func (c *LoggerConfig) Validate() error {
	switch c.Level {
	case "debug", "info", "warn", "error", "":
		// Valid
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Level)
	}

	switch c.Format {
	case "json", "text", "":
		// Valid
	default:
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.Format)
	}

	return nil
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	logger *slog.Logger
	output io.Writer
}

// NewLogger creates a new structured logger
func NewLogger(cfg LoggerConfig) *Logger {
	if err := cfg.Validate(); err != nil {
		// Fall back to defaults on invalid config
		cfg.Level = "info"
		cfg.Format = "json"
	}

	// Set default output
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Parse level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info", "":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Create handler based on format
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(output, opts)
	case "json", "":
		handler = slog.NewJSONHandler(output, opts)
	}

	return &Logger{
		logger: slog.New(handler),
		output: output,
	}
}

// NewDefaultLogger creates a logger with default settings (JSON, INFO level)
func NewDefaultLogger() *Logger {
	return NewLogger(LoggerConfig{
		Level:  "info",
		Format: "json",
	})
}

// With returns a new logger with the specified key-value pairs added
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
		output: l.output,
	}
}

// WithContext returns a new logger with trace information from the context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	if traceID == "" && spanID == "" {
		return l
	}

	args := make([]any, 0, 4)
	if traceID != "" && traceID != "00000000000000000000000000000000" {
		args = append(args, "trace_id", traceID)
	}
	if spanID != "" && spanID != "0000000000000000" {
		args = append(args, "span_id", spanID)
	}

	if len(args) == 0 {
		return l
	}

	return l.With(args...)
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// DebugContext logs at debug level with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Debug(msg, args...)
}

// InfoContext logs at info level with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Info(msg, args...)
}

// WarnContext logs at warn level with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Warn(msg, args...)
}

// ErrorContext logs at error level with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Error(msg, args...)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(cfg LoggerConfig) {
	globalLogger = NewLogger(cfg)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewDefaultLogger()
	}
	return globalLogger
}

// Package-level logging functions using global logger

// Debug logs at debug level using global logger
func Debug(msg string, args ...any) {
	GetGlobalLogger().Debug(msg, args...)
}

// Info logs at info level using global logger
func Info(msg string, args ...any) {
	GetGlobalLogger().Info(msg, args...)
}

// Warn logs at warn level using global logger
func Warn(msg string, args ...any) {
	GetGlobalLogger().Warn(msg, args...)
}

// Error logs at error level using global logger
func Error(msg string, args ...any) {
	GetGlobalLogger().Error(msg, args...)
}
