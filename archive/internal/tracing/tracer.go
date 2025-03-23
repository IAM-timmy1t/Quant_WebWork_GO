package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// TracingConfig defines configuration for distributed tracing
type TracingConfig struct {
	Enabled        bool   `json:"enabled"`
	ServiceName    string `json:"serviceName"`
	CollectorURL   string `json:"collectorURL"`
	SampleRate     float64 `json:"sampleRate"`
	BatchTimeout   time.Duration `json:"batchTimeout"`
	ExportTimeout  time.Duration `json:"exportTimeout"`
	MaxQueueSize   int    `json:"maxQueueSize"`
	MaxExportBatch int    `json:"maxExportBatch"`
}

// Tracer manages distributed tracing functionality
type Tracer struct {
	mu sync.RWMutex

	config TracingConfig
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
}

// NewTracer creates a new distributed tracer
func NewTracer(config TracingConfig) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{config: config}, nil
	}

	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(config.CollectorURL),
			otlptracegrpc.WithDialOption(grpc.WithBlock()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %v", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	// Configure trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(config.MaxQueueSize),
			sdktrace.WithMaxExportBatchSize(config.MaxExportBatch),
			sdktrace.WithBatchTimeout(config.BatchTimeout),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(config.ServiceName)

	return &Tracer{
		config: config,
		tp:     tp,
		tracer: tracer,
	}, nil
}

// StartSpan starts a new trace span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !t.config.Enabled {
		return ctx, &noopSpan{}
	}
	return t.tracer.Start(ctx, name, opts...)
}

// AddEvent adds an event to the current span
func (t *Tracer) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttribute sets an attribute on the current span
func (t *Tracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", value)))
	}
}

// RecordError records an error on the current span
func (t *Tracer) RecordError(ctx context.Context, err error) {
	if span := trace.SpanFromContext(ctx); span != nil {
		span.RecordError(err)
	}
}

// Shutdown stops the tracer provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.config.Enabled {
		return nil
	}
	return t.tp.Shutdown(ctx)
}

// noopSpan implements a no-op span for when tracing is disabled
type noopSpan struct{}

func (s *noopSpan) End(options ...trace.SpanEndOption)                       {}
func (s *noopSpan) AddEvent(name string, options ...trace.EventOption)      {}
func (s *noopSpan) IsRecording() bool                                       { return false }
func (s *noopSpan) RecordError(err error, opts ...trace.EventOption)       {}
func (s *noopSpan) SpanContext() trace.SpanContext                          { return trace.SpanContext{} }
func (s *noopSpan) SetStatus(code trace.StatusCode, description string)     {}
func (s *noopSpan) SetName(name string)                                     {}
func (s *noopSpan) SetAttributes(kv ...attribute.KeyValue)                  {}
func (s *noopSpan) TracerProvider() trace.TracerProvider                    { return nil }
