// Package logid propagates trace_id across services via header / metadata.
//
// Flow:
//
//	inbound  → read x-trace-id from gRPC metadata / HTTP header
//	         → generate a UUID if absent
//	         → inject into context (key: ctxKey{})
//	outbound → read trace_id from context, write to downstream request header
//	logging  → log.FromContext reads via FromContext to surface trace_id
package logid

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	otrace "go.opentelemetry.io/otel/trace"
)

// MetadataKey is the metadata / header key used to propagate trace_id.
const MetadataKey = "x-trace-id"

type ctxKey struct{}

// FromContext returns the active trace_id:
//   - first OTel SpanContext.TraceID() if a tracer provider is configured
//   - then the custom trace_id injected by Server() / NewContext()
//   - empty string if neither is set
//
// So business code can call FromContext unconditionally regardless of whether
// the OTel SDK has been initialized.
func FromContext(ctx context.Context) string {
	if sc := otrace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// NewContext explicitly injects a trace_id into ctx.
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// Server is the inbound middleware: read trace_id from transport headers,
// generate a UUID if missing, mirror it back into the reply header.
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			id := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				id = tr.RequestHeader().Get(MetadataKey)
			}
			if id == "" {
				id = uuid.NewString()
			}
			ctx = NewContext(ctx, id)
			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set(MetadataKey, id)
			}
			return handler(ctx, req)
		}
	}
}

// Client is the outbound middleware: copy trace_id from ctx into the
// downstream request header. Does not generate a new id when missing —
// outbound ctx should originate from an inbound request; an empty id usually
// indicates a bug upstream.
func Client() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if id := FromContext(ctx); id != "" {
				if tr, ok := transport.FromClientContext(ctx); ok {
					tr.RequestHeader().Set(MetadataKey, id)
				}
			}
			return handler(ctx, req)
		}
	}
}
