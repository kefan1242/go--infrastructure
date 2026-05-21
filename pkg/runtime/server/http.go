package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HTTPConfig is the cross-service HTTP server configuration.
//
// Filters run before any handler — use them for raw-HTTP concerns like
// CORS, body-size limits, request id at the edge.
type HTTPConfig struct {
	Network string
	Addr    string
	Timeout time.Duration
	Filters []khttp.FilterFunc
}

// HTTPRegistrar is the callback through which the caller registers HTTP handlers.
// It receives a *BizHTTPServer so callers can register both middleware-aware
// handlers (BizHTTPServer.HandleFunc) and raw HandleFunc routes
// (BizHTTPServer.S.HandleFunc, escape hatch).
type HTTPRegistrar func(*BizHTTPServer)

// NewBizHTTPServer returns the business HTTP listener.
//
// The default chain (recovery -> tracing -> logid -> access -> metric, plus
// any `extra`) is installed via kratos's native middleware mechanism so
// proto-generated handlers get it automatically. For raw http.HandlerFunc
// routes use BizHTTPServer.HandleFunc — that wraps each handler so the same
// chain applies.
func NewBizHTTPServer(cfg HTTPConfig, logger log.Logger, register HTTPRegistrar, extra ...middleware.Middleware) *BizHTTPServer {
	mws := defaultChain(logger)
	mws = append(mws, extra...)

	opts := []khttp.ServerOption{khttp.Middleware(mws...)}
	opts = append(opts, baseHTTPOpts(cfg)...)
	srv := khttp.NewServer(opts...)

	biz := &BizHTTPServer{S: srv, chain: mws}
	if register != nil {
		register(biz)
	}
	return biz
}

// NewOtherHTTPServer returns the sidecar HTTP listener exposing
// /healthz + /readyz + /metrics + /debug/pprof.
// It has no business middlewares attached — these paths must remain reachable
// without auth or rate limiting.
func NewOtherHTTPServer(cfg HTTPConfig, logger log.Logger, probes *ReadinessProbes) *OtherHTTPServer {
	srv := buildHTTPNoMiddleware(cfg)
	srv.HandleFunc("/healthz", healthHandler)
	srv.HandleFunc("/readyz", readyHandler(probes))
	srv.HandleFunc("/metrics", metricsHandler)
	registerPprof(srv)
	return &OtherHTTPServer{S: srv}
}

func buildHTTPNoMiddleware(cfg HTTPConfig) *khttp.Server {
	return khttp.NewServer(baseHTTPOpts(cfg)...)
}

func baseHTTPOpts(cfg HTTPConfig) []khttp.ServerOption {
	var opts []khttp.ServerOption
	if cfg.Network != "" {
		opts = append(opts, khttp.Network(cfg.Network))
	}
	if cfg.Addr != "" {
		opts = append(opts, khttp.Address(cfg.Addr))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, khttp.Timeout(cfg.Timeout))
	}
	if len(cfg.Filters) > 0 {
		opts = append(opts, khttp.Filter(cfg.Filters...))
	}
	return opts
}

// wrapHandler chains the configured middlewares around a raw http.HandlerFunc.
// The kratos transport context is already on r.Context() by the time this
// wrapper runs (kratos's server installs it via a router middleware), so
// middlewares like logid.Server / access.Server / metric.Server see the
// expected RequestHeader / ReplyHeader / Operation values.
func wrapHandler(mws []middleware.Middleware, h http.HandlerFunc) http.HandlerFunc {
	if len(mws) == 0 {
		return h
	}
	chained := middleware.Chain(mws...)
	return func(w http.ResponseWriter, r *http.Request) {
		kh := middleware.Handler(func(ctx context.Context, _ any) (any, error) {
			h(w, r.WithContext(ctx))
			return nil, nil
		})
		_, _ = chained(kh)(r.Context(), r)
	}
}
