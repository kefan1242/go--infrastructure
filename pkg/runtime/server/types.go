package server

import (
	"net/http"

	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// BizHTTPServer wraps the kratos HTTP server for the business listener.
//
// `S` is the underlying *khttp.Server (for proto-generated `Register*HTTPServer`
// calls, or for raw HandleFunc routes that should bypass middleware).
// `chain` is the middleware chain (default + extra) that was passed at
// construction; HandleFunc / Handle on BizHTTPServer wrap routes through it.
type BizHTTPServer struct {
	S     *khttp.Server
	chain []middleware.Middleware
}

// HandleFunc registers a raw HTTP handler at `path`, wrapped through the
// kratos middleware chain configured at NewBizHTTPServer time. Use this
// instead of `s.S.HandleFunc` when you want logid / access log / metric /
// auth / ratelimit to apply to your handler.
//
// Proto-generated handlers (`Register*HTTPServer(s.S, svc)`) already use the
// chain via kratos's native middleware path — no wrapping needed.
func (b *BizHTTPServer) HandleFunc(path string, h http.HandlerFunc) {
	b.S.HandleFunc(path, wrapHandler(b.chain, h))
}

// Handle is the http.Handler variant of HandleFunc.
func (b *BizHTTPServer) Handle(path string, h http.Handler) {
	b.S.HandleFunc(path, wrapHandler(b.chain, h.ServeHTTP))
}

// OtherHTTPServer is the named wrapper for the sidecar HTTP listener.
// It exposes /healthz, /readyz, /metrics, /debug/pprof and does NOT wrap
// handlers in any middleware — probes must remain reachable independent of
// business gating.
type OtherHTTPServer struct{ S *khttp.Server }
