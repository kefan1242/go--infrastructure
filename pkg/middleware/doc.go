// Package middleware groups cross-service reusable kratos middlewares.
//
// Layout:
//
//   - access     request-level access log (trace_id / op / latency / code)
//   - auth       inbound auth (Token / mTLS subject) with Claims context
//   - logid      inject/propagate trace_id across services
//   - metric     RED metrics emitted to the package-level Prometheus registry
//   - ratelimit  in-process token-bucket rate limit, per-key
//
// All middlewares satisfy the kratos `middleware.Middleware` interface and
// are designed to compose in the order baked into pkg/runtime/server's
// default chain. Optional ones (auth, ratelimit) plug in via the `extra`
// parameter of NewGRPCServer / NewBizHTTPServer.
package middleware
