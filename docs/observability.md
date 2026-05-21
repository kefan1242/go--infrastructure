# Observability

go-infrastructure ships three observability planes pre-wired: logs, metrics,
traces. All three flow through the default middleware chain in
`pkg/runtime/server`.

## trace_id propagation

Trace_id is the single key that ties logs, metrics, and traces together.

### Two implementations, one accessor

```
business code:
    id := logid.FromContext(ctx)
```

`logid.FromContext` returns:

1. **OTel SpanContext TraceID** if `pkg/trace.Init` was called at startup
   (real tracer provider, propagated via the W3C `traceparent` header).
2. **Custom trace_id** injected by `pkg/middleware/logid` when no OTel is
   configured (propagated via the `x-trace-id` header / gRPC metadata).
3. **Empty string** if neither is present.

This means business code never branches on "is OTel on?" — it just reads the
context.

### Inbound flow

```
client request
   │ traceparent: 00-<trace-id>-<span-id>-01   (W3C, optional)
   │ x-trace-id : <id>                          (custom, optional)
   ▼
tracing.Server  ── creates / inherits OTel span
   ▼
logid.Server    ── reads x-trace-id or generates UUID, writes to reply header
   ▼
business handler
```

### Outbound flow (pkg/client)

The gRPC / HTTP client factories in `pkg/client` install three middlewares:

```
recovery -> tracing.Client -> logid.Client
```

`tracing.Client` writes the `traceparent` header; `logid.Client` writes
`x-trace-id`. Both read from ctx — the same ctx your inbound handler received,
so the trace_id is propagated transparently.

## Logs

`pkg/log.New(name, version, instanceID)` returns a kratos logger seeded with
service identity. Every line gets `ts`, `caller`, `service.id`, `service.name`,
`service.version` prepended.

### Access log schema

The `pkg/middleware/access` middleware emits one line per RPC with:

| Field        | Source                                                  |
|--------------|---------------------------------------------------------|
| `trace_id`   | `logid.FromContext(ctx)`                                |
| `kind`       | `grpc` or `http` (kratos transport)                     |
| `op`         | `/svc.Method` (gRPC) or HTTP route                      |
| `code`       | `ok` / kratos `reason` / `error` (generic Go error)     |
| `latency_ms` | wall time around the handler                            |
| `err`        | `error.Error()` when non-nil                            |

Log level: Info on success, Warn on error.

### Sensitive headers — redact before logging

If you add per-request body / header logging beyond the default access middleware
(which only emits `kind/op/code/latency_ms/trace_id`), route the headers
through `pkglog.RedactHeaders` first:

```go
import pkglog "github.com/kris/go-infrastructure/pkg/log"

masked := pkglog.RedactHeaders(r.Header, "X-Tenant-Token")
helper.Infow("inbound", "headers", masked, "path", r.URL.Path)
```

The default mask list covers `Authorization`, `Proxy-Authorization`, `Cookie`,
`Set-Cookie`, `X-Auth-Token`, `X-Api-Key`, `X-Csrf-Token`. Extend per call.
`RedactValue(key, value)` is the single-pair variant.

Compliance / privacy audits routinely catch credential leaks in log shippers
— even one missed log line with `Authorization: Bearer eyJ...` triggers an
incident. Wire the helper as soon as you add custom request logging.

### Adding fields

To add a structured field, wrap the kratos logger with `log.With(...)`:

```go
logger = log.With(logger, "tenant", tenantID)
```

Don't rebind the global; pass the wrapped logger explicitly.

## Metrics

Exposed on the sidecar listener at `/metrics`. Uses the Prometheus default
registry; everything in `pkg/metric` is `MustRegister`'d at package init.

### Catalog

| Metric                                | Type       | Labels                | Source                              |
|---------------------------------------|------------|-----------------------|--------------------------------------|
| `kris_requests_total`                 | Counter    | `kind`, `op`, `code`  | `pkg/middleware/metric`              |
| `kris_request_latency_seconds`        | Histogram  | `kind`, `op`          | `pkg/middleware/metric`              |
| `kris_panics_total`                   | Counter    | `op`                  | `pkg/middleware/recovery`            |
| `kris_db_pool_open_connections`       | Gauge      | `name`                | `pkg/metric.SQLCollector`            |
| `kris_db_pool_in_use`                 | Gauge      | `name`                | "                                    |
| `kris_db_pool_idle`                   | Gauge      | `name`                | "                                    |
| `kris_db_pool_wait_count_total`       | Counter    | `name`                | "                                    |
| `kris_db_pool_wait_seconds_total`     | Counter    | `name`                | "                                    |
| `kris_redis_pool_total_connections`   | Gauge      | `name`                | `pkg/metric.RedisCollector`          |
| `kris_redis_pool_idle_connections`    | Gauge      | `name`                | "                                    |
| `kris_redis_pool_stale_connections`   | Gauge      | `name`                | "                                    |
| `kris_redis_pool_hits_total`          | Counter    | `name`                | "                                    |
| `kris_redis_pool_misses_total`        | Counter    | `name`                | "                                    |
| `kris_redis_pool_timeouts_total`      | Counter    | `name`                | "                                    |
| `kris_mongo_pool_connections_created_total`  | Counter | `name`           | `pkg/data/mongo` PoolMonitor          |
| `kris_mongo_pool_connections_closed_total`   | Counter | `name`, `reason` | "                                    |
| `kris_mongo_pool_checkouts_started_total`    | Counter | `name`           | "                                    |
| `kris_mongo_pool_checkouts_failed_total`     | Counter | `name`, `reason` | "                                    |
| `kris_mongo_pool_checkouts_succeeded_total`  | Counter | `name`           | "                                    |
| `kris_mongo_pool_checkins_total`             | Counter | `name`           | "                                    |

SQL/Redis collectors are registered by `NewMySQL` / `NewRedis` automatically
and unregistered in the cleanup function. Mongo is event-driven: `NewMongo`
attaches a `PoolMonitor` so counters increment as the driver fires events —
no Collector to register.

### Latency histogram buckets

```
0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10  (seconds)
```

Tuned for typical synchronous RPCs. For background jobs or LLM streaming,
either declare a separate histogram or fork the buckets in `pkg/metric/prom.go`.

### Adding your own metrics

Register from a service-local package, not by editing `pkg/metric`:

```go
var MyCounter = promauto.NewCounter(prometheus.CounterOpts{
    Name: "myservice_widgets_built_total",
    Help: "Widgets built since process start.",
})
```

Use `kris_` prefix only for metrics that genuinely come from `pkg`.

## Traces

`pkg/trace.Init` brings up an OTel tracer provider with W3C `TraceContext` +
`Baggage` propagators. Two exporters:

- `Endpoint` set → OTLP HTTP, e.g. `localhost:4318`
- `Endpoint` empty → stdout (dev)

Sampling is `TraceIDRatioBased`. `0.0` disables, `1.0` samples all.

```go
shutdown, err := trace.Init(trace.Config{
    ServiceName: "kris-worker",
    Endpoint:    "localhost:4318",
    Insecure:    true,
    SampleRatio: 0.1,
})
if err != nil { return err }
defer shutdown(context.Background())
```

Call this **before** starting your servers so the default chain's
`tracing.Server` picks up the provider.

## Local dev pipeline

`make dev-deps-up` starts Prometheus on `localhost:9090` and Grafana on
`localhost:3000`. The Prometheus config in `scripts/dev/prometheus.yml`
scrapes the kris-* sidecar ports via `host.docker.internal`. Add new
service ports there as you scaffold them.
