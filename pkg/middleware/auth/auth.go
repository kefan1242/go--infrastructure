// Package auth provides an inbound request authentication middleware.
//
// Usage:
//
//	v := func(ctx context.Context, token string) (*auth.Claims, error) {
//	    // JWT decode / session lookup / remote IAM call
//	    return &auth.Claims{Subject: "user-123", Scopes: []string{"read"}}, nil
//	}
//
//	mw := auth.Server(
//	    auth.WithValidator(v),
//	    auth.WithSkip(func(op string) bool {
//	        return op == "/pkg.svc.v1.FooService/HealthCheck"
//	    }),
//	)
//
//	// plug into pkg/runtime/server's extra middlewares
//
// Compose after pkg/middleware/logid so auth errors carry the trace_id.
package auth

import (
	"context"
	"errors"
	"strings"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Claims is the validated subject returned by the auth Validator.
// Callers retrieve it with FromContext.
type Claims struct {
	Subject string
	Scopes  []string
	Raw     map[string]any
}

// Validator is the caller-supplied token verification function.
type Validator func(ctx context.Context, token string) (*Claims, error)

// SkipFn returns true when the given operation should bypass auth (e.g. health checks).
type SkipFn func(op string) bool

type options struct {
	validator Validator
	skipFn    SkipFn
	header    string // default "authorization"
	scheme    string // default "Bearer "; empty means the whole header value is the token
}

// Option is a functional config setter.
type Option func(*options)

// WithValidator is required.
func WithValidator(v Validator) Option { return func(o *options) { o.validator = v } }

// WithSkip configures which operations skip auth.
func WithSkip(fn SkipFn) Option { return func(o *options) { o.skipFn = fn } }

// WithHeader overrides the header name and scheme. An empty scheme means the
// entire header value is treated as the token.
func WithHeader(header, scheme string) Option {
	return func(o *options) { o.header = header; o.scheme = scheme }
}

// ErrUnauthorized is returned when credentials are missing or invalid.
var ErrUnauthorized = kerrors.Unauthorized("UNAUTHORIZED", "missing or invalid credentials")

type ctxKey struct{}

// FromContext returns the active Claims, or (nil, false) when unauthenticated.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok && c != nil
}

// Server is the inbound auth middleware.
func Server(opts ...Option) middleware.Middleware {
	o := &options{header: "authorization", scheme: "Bearer "}
	for _, opt := range opts {
		opt(o)
	}
	if o.validator == nil {
		panic("auth.Server: WithValidator is required")
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			op := ""
			var hdr string
			if tr, ok := transport.FromServerContext(ctx); ok {
				op = tr.Operation()
				hdr = tr.RequestHeader().Get(o.header)
			}
			if o.skipFn != nil && o.skipFn(op) {
				return handler(ctx, req)
			}
			token := extractToken(hdr, o.scheme)
			if token == "" {
				return nil, ErrUnauthorized
			}
			claims, err := o.validator(ctx, token)
			if err != nil {
				return nil, mapErr(err)
			}
			if claims == nil {
				return nil, ErrUnauthorized
			}
			ctx = context.WithValue(ctx, ctxKey{}, claims)
			return handler(ctx, req)
		}
	}
}

func extractToken(header, scheme string) string {
	if header == "" {
		return ""
	}
	if scheme == "" {
		return strings.TrimSpace(header)
	}
	if len(header) <= len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return ""
	}
	return strings.TrimSpace(header[len(scheme):])
}

// mapErr passes kratos errors through unchanged and wraps everything else as 401.
func mapErr(err error) error {
	var k *kerrors.Error
	if errors.As(err, &k) {
		return err
	}
	return kerrors.Unauthorized("UNAUTHORIZED", err.Error())
}
