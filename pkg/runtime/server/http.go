package server

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HTTPConfig is the cross-service HTTP server configuration.
//
// Filters run BEFORE the kratos middleware chain — use them for raw-HTTP
// concerns like CORS, body-size limits, request id at the edge.
type HTTPConfig struct {
	Network string
	Addr    string
	Timeout time.Duration
	Filters []khttp.FilterFunc
}

// HTTPRegistrar is the callback through which the caller registers HTTP handlers.
type HTTPRegistrar func(*khttp.Server)

// NewBizHTTPServer returns the business HTTP listener. Default chain is the
// shared sequence (recovery -> tracing -> logid -> access -> metric).
// Callers append optional middlewares (auth / ratelimit) via `extra`.
func NewBizHTTPServer(cfg HTTPConfig, logger log.Logger, register HTTPRegistrar, extra ...middleware.Middleware) *BizHTTPServer {
	srv := buildHTTP(cfg, logger, extra...)
	if register != nil {
		register(srv)
	}
	return &BizHTTPServer{S: srv}
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

func buildHTTP(cfg HTTPConfig, logger log.Logger, extra ...middleware.Middleware) *khttp.Server {
	mws := defaultChain(logger)
	mws = append(mws, extra...)
	opts := []khttp.ServerOption{
		khttp.Middleware(mws...),
	}
	opts = append(opts, baseHTTPOpts(cfg)...)
	return khttp.NewServer(opts...)
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
