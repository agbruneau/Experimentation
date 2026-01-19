package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LoggerConfig holds configuration for the logger.
type LoggerConfig struct {
	Level       string
	Format      string
	ServiceName string
	Output      io.Writer
	AddSource   bool
}

// DefaultLoggerConfig returns a default logger configuration.
func DefaultLoggerConfig(serviceName string) LoggerConfig {
	return LoggerConfig{
		Level:       "INFO",
		Format:      "json",
		ServiceName: serviceName,
		Output:      os.Stdout,
		AddSource:   false,
	}
}

// InitLogger initializes the structured logger.
func InitLogger(config LoggerConfig) *slog.Logger {
	level := parseLevel(config.Level)
	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	if strings.ToLower(config.Format) == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	// Add service name as default attribute
	logger := slog.New(handler).With(
		slog.String("service", config.ServiceName),
	)

	// Set as default logger
	slog.SetDefault(logger)

	return logger
}

// parseLevel parses a log level string.
func parseLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithTraceID adds trace ID to the logger from context.
func WithTraceID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	traceID := TraceID(ctx)
	spanID := SpanID(ctx)

	if traceID != "" {
		logger = logger.With(slog.String("trace_id", traceID))
	}
	if spanID != "" {
		logger = logger.With(slog.String("span_id", spanID))
	}

	return logger
}

// LoggerFromContext returns a logger with trace context.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	return WithTraceID(ctx, slog.Default())
}

// ContextWithLogger adds a logger to the context.
type loggerKey struct{}

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// Logger retrieves the logger from context, or returns the default logger.
func Logger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// L is a shorthand for Logger.
func L(ctx context.Context) *slog.Logger {
	return Logger(ctx)
}

// Helper functions for common log patterns

// LogInfo logs an info message with context.
func LogInfo(ctx context.Context, msg string, args ...any) {
	Logger(ctx).InfoContext(ctx, msg, args...)
}

// LogDebug logs a debug message with context.
func LogDebug(ctx context.Context, msg string, args ...any) {
	Logger(ctx).DebugContext(ctx, msg, args...)
}

// LogWarn logs a warning message with context.
func LogWarn(ctx context.Context, msg string, args ...any) {
	Logger(ctx).WarnContext(ctx, msg, args...)
}

// LogError logs an error message with context.
func LogError(ctx context.Context, msg string, args ...any) {
	Logger(ctx).ErrorContext(ctx, msg, args...)
}

// LogWithError logs a message with an error.
func LogWithError(ctx context.Context, msg string, err error, args ...any) {
	allArgs := append([]any{slog.Any("error", err)}, args...)
	Logger(ctx).ErrorContext(ctx, msg, allArgs...)
}

// Event logging helpers for EDA-Lab

// LogEventProduced logs a produced event.
func LogEventProduced(ctx context.Context, logger *slog.Logger, topic, eventID, eventType string) {
	logger.InfoContext(ctx, "event produced",
		slog.String("topic", topic),
		slog.String("event_id", eventID),
		slog.String("event_type", eventType),
	)
}

// LogEventConsumed logs a consumed event.
func LogEventConsumed(ctx context.Context, logger *slog.Logger, topic, eventID, eventType string) {
	logger.InfoContext(ctx, "event consumed",
		slog.String("topic", topic),
		slog.String("event_id", eventID),
		slog.String("event_type", eventType),
	)
}

// LogEventProcessed logs a processed event.
func LogEventProcessed(ctx context.Context, logger *slog.Logger, eventID, eventType string, durationMs int64) {
	logger.InfoContext(ctx, "event processed",
		slog.String("event_id", eventID),
		slog.String("event_type", eventType),
		slog.Int64("duration_ms", durationMs),
	)
}

// LogEventFailed logs a failed event.
func LogEventFailed(ctx context.Context, logger *slog.Logger, eventID, eventType string, err error) {
	logger.ErrorContext(ctx, "event processing failed",
		slog.String("event_id", eventID),
		slog.String("event_type", eventType),
		slog.Any("error", err),
	)
}
