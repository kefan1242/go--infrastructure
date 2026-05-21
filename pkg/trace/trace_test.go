package trace_test

import (
	"context"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/trace"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestInit_RequiresServiceName(t *testing.T) {
	_, err := trace.Init(trace.Config{})
	if err == nil {
		t.Fatal("expected error when ServiceName is empty")
	}
}

func TestInit_StdoutExporterWhenEndpointEmpty(t *testing.T) {
	shutdown, err := trace.Init(trace.Config{
		ServiceName:    "kris-trace-test",
		ServiceVersion: "v0.0.0",
		SampleRatio:    1.0,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	tp := otel.GetTracerProvider()
	if _, ok := tp.(*sdktrace.TracerProvider); !ok {
		t.Errorf("global TracerProvider not configured: %T", tp)
	}
}

func TestInit_SamplerClamping(t *testing.T) {
	cases := []struct {
		name  string
		ratio float64
	}{
		{"zero", 0},
		{"half", 0.5},
		{"one", 1.0},
		{"over", 2.0}, // clamped to AlwaysSample
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shutdown, err := trace.Init(trace.Config{
				ServiceName: "kris-trace-test-" + tc.name,
				SampleRatio: tc.ratio,
			})
			if err != nil {
				t.Fatalf("Init: %v", err)
			}
			defer shutdown(context.Background())
		})
	}
}

func TestTraceIDFromContext_EmptyWithoutSpan(t *testing.T) {
	if got := trace.TraceIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty trace_id without span, got %q", got)
	}
}

func TestTraceIDFromContext_PopulatedWithSpan(t *testing.T) {
	shutdown, err := trace.Init(trace.Config{
		ServiceName: "kris-trace-test-span",
		SampleRatio: 1.0,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer shutdown(context.Background())

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	got := trace.TraceIDFromContext(ctx)
	if got == "" {
		t.Fatal("expected non-empty trace_id inside active span")
	}
	want := span.SpanContext().TraceID().String()
	if got != want {
		t.Errorf("trace_id: want %s, got %s", want, got)
	}
	_ = oteltrace.SpanFromContext(ctx) // ensure imported pkg compiles
}

func TestInit_PropagatorIsTraceContextBaggage(t *testing.T) {
	shutdown, err := trace.Init(trace.Config{
		ServiceName: "kris-trace-test-prop",
		SampleRatio: 1.0,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer shutdown(context.Background())

	prop := otel.GetTextMapPropagator()
	if prop == nil {
		t.Fatal("nil propagator")
	}
	fields := prop.Fields()
	want := map[string]bool{"traceparent": false, "tracestate": false, "baggage": false}
	for _, f := range fields {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("propagator missing field %q (got %v)", k, fields)
		}
	}
}
