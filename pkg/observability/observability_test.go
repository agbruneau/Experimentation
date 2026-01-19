package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInitLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "INFO",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("Expected JSON format with msg field, got: %s", output)
	}
	if !strings.Contains(output, `"service":"test-service"`) {
		t.Errorf("Expected service name in output, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("Expected key-value in output, got: %s", output)
	}
}

func TestInitLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "DEBUG",
		Format:      "text",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)
	logger.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("Expected debug message in output, got: %s", output)
	}
}

func TestInitLogger_LevelFilter(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "WARN",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)

	// This should not appear
	logger.Info("info message")
	// This should appear
	logger.Warn("warn message")

	output := buf.String()
	if strings.Contains(output, "info message") {
		t.Errorf("Info message should be filtered out at WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Errorf("Warn message should appear")
	}
}

func TestMetrics_RecordMessageProduced(t *testing.T) {
	metrics := NewMetrics("test")

	// This should not panic
	metrics.RecordMessageProduced("test-service", "test-topic")

	// Record multiple times
	for i := 0; i < 10; i++ {
		metrics.RecordMessageProduced("test-service", "test-topic")
	}
}

func TestMetrics_RecordProcessingLatency(t *testing.T) {
	metrics := NewMetrics("test")

	// Record various latencies
	metrics.RecordProcessingLatency("test-service", "CompteOuvert", 10*time.Millisecond)
	metrics.RecordProcessingLatency("test-service", "CompteOuvert", 50*time.Millisecond)
	metrics.RecordProcessingLatency("test-service", "DepotEffectue", 100*time.Millisecond)
}

func TestTimer(t *testing.T) {
	timer := NewTimer()
	time.Sleep(10 * time.Millisecond)
	elapsed := timer.Elapsed()

	if elapsed < 10*time.Millisecond {
		t.Errorf("Expected elapsed >= 10ms, got %v", elapsed)
	}
}

func TestWithTraceID(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "INFO",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)

	// Without trace context
	ctx := context.Background()
	loggerWithTrace := WithTraceID(ctx, logger)
	loggerWithTrace.Info("message without trace")

	// The output should not have trace_id since context has no trace
	// Just verify it doesn't panic
}

func TestLoggerFromContext(t *testing.T) {
	// Without logger in context - should return default
	ctx := context.Background()
	logger := LoggerFromContext(ctx)
	if logger == nil {
		t.Error("LoggerFromContext should not return nil")
	}
}

func TestContextWithLogger(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "INFO",
		Format:      "json",
		ServiceName: "custom-service",
		Output:      &buf,
	}

	customLogger := InitLogger(config)

	ctx := WithLogger(context.Background(), customLogger)
	logger := Logger(ctx)

	logger.Info("test from context")

	if !strings.Contains(buf.String(), "custom-service") {
		t.Errorf("Expected custom logger output, got: %s", buf.String())
	}
}

func TestLogHelpers(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "DEBUG",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)
	ctx := WithLogger(context.Background(), logger)

	LogInfo(ctx, "info message")
	LogDebug(ctx, "debug message")
	LogWarn(ctx, "warn message")

	output := buf.String()
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message")
	}
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Expected warn message")
	}
}

func TestNewNoopTracer(t *testing.T) {
	tracer := NewNoopTracer()
	if tracer == nil {
		t.Error("NewNoopTracer should not return nil")
	}

	ctx, span := tracer.StartSpan(context.Background(), "test-span")
	if ctx == nil {
		t.Error("Context should not be nil")
	}
	span.End()
}

func TestInjectExtractTraceContext(t *testing.T) {
	headers := make(map[string]string)

	// Inject into empty context (should not panic)
	ctx := context.Background()
	InjectTraceContext(ctx, headers)

	// Extract from empty headers (should not panic)
	extractedCtx := ExtractTraceContext(context.Background(), headers)
	if extractedCtx == nil {
		t.Error("ExtractTraceContext should not return nil")
	}
}

func TestHeaderCarrier(t *testing.T) {
	carrier := HeaderCarrier(make(map[string]string))

	carrier.Set("key1", "value1")
	carrier.Set("key2", "value2")

	if carrier.Get("key1") != "value1" {
		t.Errorf("Expected value1, got %s", carrier.Get("key1"))
	}

	keys := carrier.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestLogEventHelpers(t *testing.T) {
	var buf bytes.Buffer

	config := LoggerConfig{
		Level:       "INFO",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	}

	logger := InitLogger(config)
	ctx := context.Background()

	LogEventProduced(ctx, logger, "test-topic", "event-123", "CompteOuvert")

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["topic"] != "test-topic" {
		t.Errorf("Expected topic=test-topic, got %v", logEntry["topic"])
	}
	if logEntry["event_id"] != "event-123" {
		t.Errorf("Expected event_id=event-123, got %v", logEntry["event_id"])
	}
	if logEntry["event_type"] != "CompteOuvert" {
		t.Errorf("Expected event_type=CompteOuvert, got %v", logEntry["event_type"])
	}
}
