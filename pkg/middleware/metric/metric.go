// Package metric provides RPC-level Prometheus RED middleware.
// Metric definitions live in pkg/metric.
package metric

import (
	"context"
	"errors"
	"time"

	pkgmetric "github.com/kris/go-infrastructure/pkg/metric"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Server is the inbound RED-metric middleware.
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			reply, err := handler(ctx, req)

			kind, op := "", ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				kind = string(tr.Kind())
				op = tr.Operation()
			}
			code := "ok"
			if err != nil {
				code = "error"
				var kerr *kerrors.Error
				if errors.As(err, &kerr) && kerr.Reason != "" {
					code = kerr.Reason
				}
			}
			pkgmetric.RequestsTotal.WithLabelValues(kind, op, code).Inc()
			pkgmetric.RequestLatencySeconds.WithLabelValues(kind, op).Observe(time.Since(start).Seconds())
			return reply, err
		}
	}
}
