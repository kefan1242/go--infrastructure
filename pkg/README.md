# pkg

Reusable Go infrastructure module. Drop-in scaffolding for any service:
kratos-based gRPC + HTTP server with a sensible default middleware chain,
observability, data-store factories, config loader, test helpers.

Module path: `github.com/kris/go-infrastructure/pkg`

## Layout

| Subpackage          | What it offers                                                          |
|---------------------|--------------------------------------------------------------------------|
| `client`            | gRPC client factory with default middlewares + fail-fast Dial            |
| `config`            | layered `.env` / env-var loader with project-root auto-discovery         |
| `data`              | `mysql` / `redis` / `mongo` connection factories (ping at startup)       |
| `log`               | seeded kratos logger (text + JSON) + slog.Handler adapter                |
| `metric`            | shared Prometheus metric definitions                                     |
| `middleware/access` | structured per-request access log                                        |
| `middleware/auth`   | token-based auth with pluggable Validator and skip rules                 |
| `middleware/cors`   | CORS filter for the HTTP business listener (origin allowlist + preflight) |
| `middleware/logid`  | trace_id propagation header / metadata + OTel SpanContext bridge         |
| `middleware/metric` | RED metrics into pkg/metric                                              |
| `middleware/ratelimit` | in-process token-bucket rate limiter, per-key                         |
| `middleware/recovery` | recover panics + emit `kris_panics_total{op}` + stable PANIC error    |
| `middleware/timeout` | per-request deadline with kratos `GatewayTimeout` on overrun              |
| `runtime/server`    | gRPC + business HTTP + sidecar HTTP (`/healthz`, `/readyz`, `/metrics`, pprof) |
| `testutil`          | memory logger + fake transport for middleware tests                      |
| `trace`             | OpenTelemetry tracer-provider initialization                             |
| `version`           | build-info struct + `/version` JSON handler                              |
| `third_party`       | shared `.proto` imports (errors, validate, google APIs)                  |

## Default server middleware chain

```
recovery -> tracing.Server -> logid.Server -> access.Server -> metric.Server
```

Optional middlewares (`auth`, `ratelimit`) plug in via the `extra` parameter on
`NewGRPCServer` / `NewBizHTTPServer`.

## Bringing it into your service

```go
import (
    pkglog    "github.com/kris/go-infrastructure/pkg/log"
    pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
)

logger := pkglog.New("my-service", "v0.1.0", instanceID)
gs := pkgserver.NewGRPCServer(grpcCfg, logger, func(s *grpc.Server) {
    myv1.RegisterMyServiceServer(s, svc)
})
hs := pkgserver.NewBizHTTPServer(httpCfg, logger, func(s *khttp.Server) {
    myv1.RegisterMyServiceHTTPServer(s, svc)
})
oh := pkgserver.NewOtherHTTPServer(otherCfg, logger, probes)
```

See the `kris-alpha`, `kris-beta`, `kris-gamma` example services for working wirings.
