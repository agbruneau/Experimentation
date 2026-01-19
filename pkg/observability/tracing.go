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
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds configuration for the tracer.
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	SampleRate     float64
	Enabled        bool
}

// DefaultTracerConfig returns a default tracer configuration.
func DefaultTracerConfig(serviceName string) TracerConfig {
	return TracerConfig{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       "localhost:4318",
		SampleRate:     1.0,
		Enabled:        true,
	}
}

// Tracer wraps the OpenTelemetry tracer.
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	config   TracerConfig
}

// InitTracer initializes the OpenTelemetry tracer.
func InitTracer(ctx context.Context, config TracerConfig) (*Tracer, error) {
	if !config.Enabled {
		// Return a no-op tracer
		return &Tracer{
			tracer: otel.Tracer(config.ServiceName),
			config: config,
		}, nil
	}

	// Create OTLP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(config.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Tracer{
		provider: provider,
		tracer:   provider.Tracer(config.ServiceName),
		config:   config,
	}, nil
}

// Shutdown shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new span with the given name.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with attributes.
func (t *Tracer) StartSpanWithAttributes(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// SpanFromContext returns the span from the context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// TraceID returns the trace ID from the context.
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID returns the span ID from the context.
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// HeaderCarrier implements propagation.TextMapCarrier for map[string]string.
type HeaderCarrier map[string]string

// Get returns the value for the key.
func (c HeaderCarrier) Get(key string) string {
	return c[key]
}

// Set sets a key-value pair.
func (c HeaderCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns all keys.
func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceContext injects the trace context into headers.
func InjectTraceContext(ctx context.Context, headers map[string]string) {
	carrier := HeaderCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractTraceContext extracts the trace context from headers.
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	carrier := HeaderCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// NewNoopTracer returns a no-op tracer for testing.
func NewNoopTracer() *Tracer {
	return &Tracer{
		tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		config: TracerConfig{Enabled: false},
	}
}
