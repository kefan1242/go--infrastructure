// Package client provides a reusable downstream gRPC client factory.
//
// Compared to a bare grpc.Dial, this adds four things:
//  1. Default kratos client middlewares: recovery + tracing.Client + logid
//     propagation + per-operation SRE circuit breaker.
//  2. Insecure by default (intra-cluster plaintext); callers opt into TLS via DialOption.
//  3. Fail-fast at startup: a bounded Dial so connectivity errors surface immediately
//     rather than on the first RPC.
//  4. Opt out of the breaker per-call via Config.NoCircuitBreaker when the
//     caller has its own backoff/retry layer that the breaker would interfere with.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"
	"github.com/kris/go-infrastructure/pkg/middleware/retry"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"
)

// Config is the dial configuration for a downstream gRPC service.
type Config struct {
	Endpoint    string        // host:port, e.g. "kris-alpha:50051"
	Timeout     time.Duration // per-RPC default timeout (0 = unset)
	DialTimeout time.Duration // dial timeout; default 5s
	// NoCircuitBreaker disables the default per-operation SRE circuit breaker.
	// Leave false unless your caller wraps this conn with its own retry/backoff
	// layer that needs to see every failure (otherwise the breaker eats requests
	// before retries can fire).
	NoCircuitBreaker bool
	// Retry, when non-zero MaxAttempts, installs retry middleware BEFORE the
	// circuit breaker. **Only enable for idempotent RPCs** — the middleware
	// does not enforce idempotency.
	Retry RetryConfig
}

// New dials a *grpc.ClientConn and returns a cleanup function. It injects the
// standard recovery + tracing + logid client middlewares; additional dial
// options can be passed in `extra`.
func New(cfg Config, logger log.Logger, extra ...grpc.DialOption) (*grpc.ClientConn, func(), error) {
	if cfg.Endpoint == "" {
		return nil, func() {}, fmt.Errorf("grpc client: empty Endpoint")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/client/grpc", "endpoint", cfg.Endpoint))

	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	// Middleware order:
	//   - recovery catches panics
	//   - tracing.Client opens a span (writes traceparent if trace.Init was called;
	//     no-op otherwise — global is the noop provider until Init runs)
	//   - logid.Client copies trace_id into x-trace-id; FromContext prefers OTel TraceID
	//   - circuitbreaker (innermost) trips per-op when downstream errors spike (SRE algorithm).
	mws := []middleware.Middleware{
		recovery.Recovery(),
		tracing.Client(),
		logid.Client(),
	}
	if cfg.Retry.MaxAttempts > 1 {
		retryOpts := []retry.Option{retry.WithMaxAttempts(cfg.Retry.MaxAttempts)}
		if cfg.Retry.InitialBackoff > 0 {
			retryOpts = append(retryOpts, retry.WithInitialBackoff(cfg.Retry.InitialBackoff))
		}
		if cfg.Retry.MaxBackoff > 0 {
			retryOpts = append(retryOpts, retry.WithMaxBackoff(cfg.Retry.MaxBackoff))
		}
		mws = append(mws, retry.Client(retryOpts...))
	}
	if !cfg.NoCircuitBreaker {
		mws = append(mws, circuitbreaker.Client())
	}
	opts := []kgrpc.ClientOption{
		kgrpc.WithEndpoint(cfg.Endpoint),
		kgrpc.WithMiddleware(mws...),
	}
	if cfg.Timeout > 0 {
		opts = append(opts, kgrpc.WithTimeout(cfg.Timeout))
	}
	if len(extra) > 0 {
		opts = append(opts, kgrpc.WithOptions(extra...))
	}

	conn, err := kgrpc.DialInsecure(ctx, opts...)
	if err != nil {
		return nil, func() {}, fmt.Errorf("grpc client dial %s: %w", cfg.Endpoint, err)
	}
	helper.Info("downstream connected")

	cleanup := func() {
		if err := conn.Close(); err != nil {
			helper.Errorf("conn close: %v", err)
			return
		}
		helper.Info("downstream closed")
	}
	return conn, cleanup, nil
}
