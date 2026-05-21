// Package recovery wraps kratos's recovery middleware to emit a Prometheus
// panic counter and return a stable kratos error.
//
// Plugging this into the default chain means /metrics carries
// `kris_panics_total{op="..."}` — alert on rate > 0 to surface latent bugs
// that would otherwise be visible only in logs.
package recovery

import (
	"context"

	pkgmetric "github.com/kris/go-infrastructure/pkg/metric"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport"
)

// ErrPanic is the kratos error surfaced after a panic is recovered.
var ErrPanic = kerrors.InternalServer("PANIC", "handler panicked")

// Server wraps kratos's recovery.Recovery to (1) increment kris_panics_total
// labeled by op and (2) return a stable PANIC kratos error to the caller.
// The underlying recovery already logs the stack trace.
func Server() middleware.Middleware {
	return recovery.Recovery(recovery.WithHandler(func(ctx context.Context, _, _ any) error {
		op := ""
		if tr, ok := transport.FromServerContext(ctx); ok {
			op = tr.Operation()
		}
		pkgmetric.PanicsTotal.WithLabelValues(op).Inc()
		return ErrPanic
	}))
}
