// Package timeout enforces a per-request deadline on inbound handlers.
//
// The middleware wraps the request context with context.WithTimeout(d) and
// returns kratos `GatewayTimeout (DEADLINE_EXCEEDED)` if the deadline fires
// before the handler returns. Well-behaved handlers should observe ctx.Done()
// and abort their own work promptly.
//
// Note: this middleware cannot forcibly stop a handler that ignores ctx
// cancellation — the goroutine continues until the handler finally returns.
// Treat goroutine leaks on timeout as a handler bug, not a middleware bug.
package timeout

import (
	"context"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// SkipFn returns true to bypass the timeout for the given operation.
type SkipFn func(op string) bool

type options struct {
	timeout time.Duration
	skipFn  SkipFn
}

// Option is a functional config setter.
type Option func(*options)

// WithTimeout sets the per-request deadline. A non-positive value disables
// the middleware (it becomes a pass-through).
func WithTimeout(d time.Duration) Option { return func(o *options) { o.timeout = d } }

// WithSkip exempts ops from the timeout (e.g. streaming endpoints, /healthz).
func WithSkip(fn SkipFn) Option { return func(o *options) { o.skipFn = fn } }

// ErrDeadlineExceeded is returned when the handler doesn't complete within
// the configured timeout.
var ErrDeadlineExceeded = kerrors.GatewayTimeout("DEADLINE_EXCEEDED", "handler exceeded configured timeout")

// Server is the inbound timeout middleware.
func Server(opts ...Option) middleware.Middleware {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if o.timeout <= 0 {
				return handler(ctx, req)
			}
			if o.skipFn != nil {
				op := ""
				if tr, ok := transport.FromServerContext(ctx); ok {
					op = tr.Operation()
				}
				if o.skipFn(op) {
					return handler(ctx, req)
				}
			}

			ctx, cancel := context.WithTimeout(ctx, o.timeout)
			defer cancel()

			type result struct {
				reply any
				err   error
			}
			ch := make(chan result, 1) // buffered so a slow handler can still publish
			go func() {
				reply, err := handler(ctx, req)
				ch <- result{reply: reply, err: err}
			}()

			select {
			case r := <-ch:
				return r.reply, r.err
			case <-ctx.Done():
				return nil, ErrDeadlineExceeded
			}
		}
	}
}
