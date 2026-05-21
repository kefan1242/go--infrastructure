// Package access provides a per-request access-log middleware.
//
// Differences from kratos's built-in logging.Server:
//   - reads trace_id from ctx (injected by pkg/middleware/logid)
//   - fixed structured fields: kind/op/latency_ms/code/err/trace_id —
//     so log shippers can parse it consistently
package access

import (
	"context"
	"errors"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Server is the inbound access-log middleware.
func Server(logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(log.With(logger, "module", "access"))
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			reply, err := handler(ctx, req)

			kind, op := "", ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				kind = string(tr.Kind())
				op = tr.Operation()
			}
			code := codeOf(err)
			latencyMs := time.Since(start).Milliseconds()
			traceID := logid.FromContext(ctx)

			fields := []any{
				"trace_id", traceID,
				"kind", kind,
				"op", op,
				"code", code,
				"latency_ms", latencyMs,
			}
			if err != nil {
				fields = append(fields, "err", err.Error())
				helper.WithContext(ctx).Warnw(fields...)
			} else {
				helper.WithContext(ctx).Infow(fields...)
			}
			return reply, err
		}
	}
}

// codeOf maps an error to a readable code string.
// kratos errors expose a `Reason` which we surface directly; everything else
// falls back to "error" (and nil → "ok").
func codeOf(err error) string {
	if err == nil {
		return "ok"
	}
	var kerr *kerrors.Error
	if errors.As(err, &kerr) {
		return kerr.Reason
	}
	return "error"
}
