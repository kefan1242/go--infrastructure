package server

import (
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/access"
	"github.com/kris/go-infrastructure/pkg/middleware/logid"
	pkgmetricmw "github.com/kris/go-infrastructure/pkg/middleware/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// GRPCConfig is the cross-service gRPC server configuration.
type GRPCConfig struct {
	Network string
	Addr    string
	Timeout time.Duration
}

// GRPCRegistrar is the callback through which the caller registers their
// generated service handlers.
type GRPCRegistrar func(*grpc.Server)

// NewGRPCServer returns a gRPC server wired with the default middleware
// chain: recovery -> tracing -> logid -> access -> metric.
// Callers can append optional middlewares (auth / ratelimit / ...) via `extra`.
func NewGRPCServer(cfg GRPCConfig, logger log.Logger, register GRPCRegistrar, extra ...middleware.Middleware) *grpc.Server {
	mws := defaultChain(logger)
	mws = append(mws, extra...)
	opts := []grpc.ServerOption{
		grpc.Middleware(mws...),
	}
	if cfg.Network != "" {
		opts = append(opts, grpc.Network(cfg.Network))
	}
	if cfg.Addr != "" {
		opts = append(opts, grpc.Address(cfg.Addr))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, grpc.Timeout(cfg.Timeout))
	}
	srv := grpc.NewServer(opts...)
	if register != nil {
		register(srv)
	}
	return srv
}

// defaultChain is the shared inbound middleware sequence for gRPC and HTTP.
//
// Order rationale:
//   - recovery is outermost so panics never escape
//   - tracing.Server runs before logid so logid.FromContext can read the OTel TraceID
//     (when pkg/trace.Init hasn't been called, the global provider is noop — harmless)
//   - access / metric come last so they observe the final trace_id and code
func defaultChain(logger log.Logger) []middleware.Middleware {
	return []middleware.Middleware{
		recovery.Recovery(),
		tracing.Server(),
		logid.Server(),
		access.Server(logger),
		pkgmetricmw.Server(),
	}
}
