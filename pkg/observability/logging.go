package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// InitLogger initializes a structured JSON logger
func InitLogger(serviceName string, level string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler).With(
		slog.String("service", serviceName),
	)

	return logger
}

// WithTraceID adds trace ID to the logger if present in context
func WithTraceID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return logger
}

// LogError logs an error with context
func LogError(ctx context.Context, logger *slog.Logger, msg string, err error, attrs ...slog.Attr) {
	logger = WithTraceID(ctx, logger)
	args := make([]any, 0, len(attrs)*2+2)
	args = append(args, slog.String("error", err.Error()))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	logger.Error(msg, args...)
}

// LogInfo logs an info message with context
func LogInfo(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
	logger = WithTraceID(ctx, logger)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr)
	}
	logger.Info(msg, args...)
}

// LogDebug logs a debug message with context
func LogDebug(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
	logger = WithTraceID(ctx, logger)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr)
	}
	logger.Debug(msg, args...)
}
