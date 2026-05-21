// Package trace provides OpenTelemetry tracing initialization.
//
// Relationship to pkg/middleware/logid:
//   - logid is a lightweight custom trace_id (x-trace-id header); it has no span tree.
//   - trace is the OTel standard pipeline with spans / parents / durations /
//     attributes; ships to Jaeger / Tempo / cloud APMs.
//   - When OTel is initialized, logid.Server() prefers the OTel TraceID from
//     the active SpanContext.
//
// Usage (called from each service's cmd/main.go at startup):
//
//	shutdown, err := trace.Init(trace.Config{
//	    ServiceName: "kris-alpha",
//	    Endpoint:    "localhost:4318",     // OTLP HTTP
//	    Insecure:    true,
//	    SampleRatio: 1.0,
//	})
//	if err != nil { return err }
//	defer shutdown(context.Background())
//
// When Endpoint is empty the SDK falls back to a stdout exporter (handy in dev).
package trace

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// Config controls OTel initialization.
type Config struct {
	// ServiceName is required; written as a resource attribute.
	ServiceName string
	// ServiceVersion is optional.
	ServiceVersion string
	// Endpoint is the OTLP HTTP collector address, e.g. "localhost:4318".
	// Empty value falls back to the stdout exporter (dev).
	Endpoint string
	// Insecure switches to plain HTTP (default is HTTPS for OTLP).
	Insecure bool
	// SampleRatio in [0,1]; 0 disables, 1 samples all.
	SampleRatio float64
	// ExportTimeout is the per-export timeout; default 10s.
	ExportTimeout time.Duration
}

// Shutdown closes the tracer provider; call it via defer in main.
type Shutdown func(context.Context) error

// Init configures the global tracer provider and propagator. Returns a
// Shutdown function; defer it so any buffered spans get flushed.
func Init(cfg Config) (Shutdown, error) {
	if cfg.ServiceName == "" {
		return nopShutdown, fmt.Errorf("trace: ServiceName is required")
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nopShutdown, fmt.Errorf("trace: build resource: %w", err)
	}

	exporter, err := buildExporter(cfg)
	if err != nil {
		return nopShutdown, err
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRatio)
	if cfg.SampleRatio <= 0 {
		sampler = sdktrace.NeverSample()
	} else if cfg.SampleRatio >= 1 {
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	// Propagate via W3C TraceContext (traceparent header).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func buildExporter(cfg Config) (sdktrace.SpanExporter, error) {
	if cfg.Endpoint == "" {
		return stdouttrace.New(stdouttrace.WithWriter(os.Stderr))
	}
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	timeout := cfg.ExportTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	opts = append(opts, otlptracehttp.WithTimeout(timeout))
	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

// TraceIDFromContext returns the current span's trace_id ("" when there is no
// active span). Complements pkg/middleware/logid.FromContext: logid serves the
// custom header path, this serves the OTel path.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

func nopShutdown(_ context.Context) error { return nil }
