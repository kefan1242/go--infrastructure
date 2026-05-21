package server

import khttp "github.com/go-kratos/kratos/v2/transport/http"

// BizHTTPServer and OtherHTTPServer are named wrappers around *khttp.Server.
// They exist so wire (or any DI) can tell apart the business HTTP listener
// (unary handlers) from the sidecar listener (/healthz + /readyz + /metrics + /debug/pprof).
type BizHTTPServer struct{ S *khttp.Server }
type OtherHTTPServer struct{ S *khttp.Server }
