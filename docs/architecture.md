# Architecture

go-infrastructure is a Go service scaffold built on [go-kratos](https://github.com/go-kratos/kratos).
Two layers:

1. **`pkg/`** — reusable infrastructure module. Everything cross-cutting:
   transport, middleware, data-store factories, observability, config, test
   helpers. Imported by services.
2. **`kris-*/`** — example services. Each is an independent Go module that
   imports `pkg`. They demonstrate wiring patterns, not business logic.

## Listener layout

Every service runs **three** listeners:

| Listener        | Port (example, alpha) | What it serves                                          |
|-----------------|----------------------:|----------------------------------------------------------|
| gRPC            | `50051`               | business RPCs                                            |
| business HTTP   | `8080`                | unary REST / SSE / WebSocket — the public surface        |
| **sidecar HTTP**| `8081`                | `/healthz`, `/readyz`, `/metrics`, `/debug/pprof`, `/version` |

The sidecar listener is kept separate so that probes and observability remain
reachable independent of business-side auth / rate-limit / network policy.
Pprof + metrics sitting behind business middleware is a foot-gun.

## Default middleware chain

`pkgserver.NewGRPCServer` and `NewBizHTTPServer` apply the same chain:

```
recovery -> tracing.Server -> logid.Server -> access.Server -> metric.Server
```

Order rationale, outermost first:

| Position | Middleware       | Why here                                                                 |
|---------:|------------------|--------------------------------------------------------------------------|
| 1 (outer)| `recovery`       | panic must not escape — wrap everything else                             |
| 2        | `tracing.Server` | start span before `logid` so `logid.FromContext` can prefer OTel TraceID |
| 3        | `logid.Server`   | inject custom trace_id when no OTel; mirror to reply header              |
| 4        | `access.Server`  | structured access log with trace_id + final code                         |
| 5 (inner)| `metric.Server`  | RED metrics observe the final code                                       |

Optional middlewares (`auth`, `ratelimit`) plug in via the `extra` variadic
on each constructor — they sit **after** the default chain so they can see
`trace_id` in their own logs.

## pkg module map

```
pkg/
├── client/         downstream RPC client factories (gRPC, HTTP)
├── config/         layered .env + env-var loader
├── data/           mysql / redis / mongo connection factories (+ pool metrics)
├── log/            seeded kratos logger
├── metric/         shared Prometheus metric defs + pool collectors
├── middleware/     access / auth / logid / metric / ratelimit
├── runtime/server/ gRPC + business HTTP + sidecar HTTP constructors
├── testutil/       memory logger + fake transport for middleware tests
├── trace/          OTel tracer-provider initialization
├── version/        build-info struct + /version handler
└── third_party/    shared .proto includes (kratos errors, validate, google APIs)
```

## Service module shape

A service is a thin assembler. The kris-alpha template:

```
kris-alpha/
├── cmd/alpha/main.go     wire up: logger, servers, optional clients
├── Makefile              build / run / test / proto / wire
├── Dockerfile            multi-stage: golang -> debian-slim
└── go.mod
```

Real services grow `api/<svc>/v1/*.proto`, `internal/{biz,data,service,server}`,
generated `*.pb.go` and `wire_gen.go` — but **never** anything that overlaps
with what `pkg` provides. If you find yourself reimplementing one of the
listed bullets above, prefer extending `pkg` instead.

## What does NOT live in pkg

- **Business types.** No domain entities, no API definitions, no service
  interfaces. Services own their own `api/` and `internal/`.
- **Service-specific config schemas.** `pkg/config` is a generic loader; each
  service declares its own struct.
- **DI container.** Each service runs its own `wire` setup. `pkg/runtime/server`
  intentionally exports plain constructors so any DI tool works.

## Trade-offs we deliberately accepted

- **Single in-process rate limiter.** Each replica enforces its own quota.
  Cluster-wide limits belong in an Envoy rate-limit service or Redis Lua,
  not in `pkg/middleware/ratelimit`.
- **No mocks of mysql / redis / mongo.** Tests should hit a real instance
  (the `dev-deps-up` stack) or use the driver's own test helpers. Mocking
  drivers masks migration / dialect / pool bugs.
- **OTel-or-custom, not both.** When `pkg/trace.Init` is called, the OTel
  TraceID wins inside `logid.FromContext`. The custom `x-trace-id` header
  is the fallback for deployments without an OTel collector.
