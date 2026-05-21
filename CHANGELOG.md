# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
the repo uses semver once it cuts a `v0.1.0`.

## [Unreleased]

### Added
- `pkg/page` — generic `Param{PageNo, PageSize}` + `Result[T]{Total, Pages, List}`.
  `Param.Normalize / Offset / Limit` for SQL queries; `New / Empty / Map` for
  building responses. No ORM coupling — works above any data layer.
- `docs/data-layering.md` — when (and when not) to separate Entity / Domain /
  API DTO in Go. Documents the one hard rule (sensitive fields never share a
  struct with API output) and lets adopters pick deliberately.
- **TLS support** in `pkg/runtime/server`. `HTTPConfig.TLSConfig` and
  `GRPCConfig.TLSConfig` accept `*tls.Config`. `TLSFromFiles(cert, key,
  clientCAFile)` loads cert+key, with optional client-CA bundle to enable
  mTLS (sets `ClientAuth=RequireAndVerifyClientCert`). `MinVersion` pinned
  to TLS 1.2. Tested with self-signed cert end-to-end.
- **Graceful-shutdown drain delay** via `server.NewDrainer(delay)`. On
  SIGTERM kratos invokes `drainer.BeforeStop`, which flips
  `drainer.Draining() = true` (so the wired readiness Pinger fails /readyz)
  then sleeps for the configured grace before letting the servers stop.
  Eliminates the SIGTERM-to-kube-proxy-sync gap that drops connections
  during k8s rolling deploys. kris-alpha wires it via `-drain-grace=10s`.
- `make demo` / `make demo-stop` boot all three kris-* services with
  non-colliding default ports (28xxx) and print a curl cheat-sheet.
- `BenchmarkDefaultChain_PassesThroughNoopHandler` pins per-request
  overhead of the default middleware chain (~1.3µs / 2.2KB / 35 allocs
  baseline on Apple M5 Pro).
- `pkg/middleware/recovery` wraps kratos's bare recovery to emit
  `kris_panics_total{op}` and return a stable kratos InternalServer /
  PANIC. Now the outer of the default chain.
- `pkg/log.SlogHandler(logger)` — slog.Handler adapter so libraries that
  accept the stdlib `log/slog` API can route through the kratos pipeline.
  Maps slog DEBUG/INFO/WARN/ERROR to kratos levels; honors WithAttrs +
  WithGroup key prefixing.
- `make cover-gate` (default 75.0%) — CI fails if total pkg coverage drops.
- `pkg/client/http` HTTP client factory with default recovery/tracing/logid chain.
- `pkg/middleware/cors` CORS filter (origin allowlist, preflight short-circuit, credentials echo).
- `pkg/middleware/timeout` per-request deadline (kratos `GatewayTimeout` on overrun).
- `pkg/log.NewJSON` / `NewJSONTo` structured JSON logger.
- `pkg/metric.SQLCollector` / `RedisCollector` + mongo PoolMonitor counters
  (`kris_db_pool_*`, `kris_redis_pool_*`, `kris_mongo_pool_*`).
- `pkg/version.Info` + JSON `/version` handler.
- `BizHTTPServer.HandleFunc` wraps raw HTTP handlers through the middleware chain
  (kratos's native middleware only wraps proto-dispatched paths — plain
  HandleFunc silently bypassed them).
- Local-deps stack (`docker-compose.dev.yml` + `scripts/dev/*`), pinned
  toolchain installer (`tools/install.sh`), `scripts/new-service.sh`
  scaffolder, helm-chart template under `kris-alpha/helm-charts/`.
- Docs: `docs/{getting-started,architecture,observability,middleware}.md`.
- Goleak (`go.uber.org/goleak`) in 13 test packages.
- CI: build matrix (Go 1.25 + 1.24), lint (golangci-lint v1.62.2 with
  `--build-tags=integration`), integration job (mysql/redis/mongo service
  containers), scaffold smoke job, fmt-check.

- `.github/dependabot.yml` — weekly grouped updates for pkg + each kris-*
  Go module, github-actions, and docker.
- `.github/PULL_REQUEST_TEMPLATE.md` + ISSUE_TEMPLATE bug / feature stubs.
- `make fmt-check` (verify-only) + `make ci-local` (full mirror of CI).
- Scaffolder smoke job in CI runs `scripts/new-service.sh` end-to-end.

### Fixed
- `scripts/new-service.sh` portable-sed compatibility (BSD/macOS): drop
  `\b`, use explicit anchors so Dockerfile EXPOSE / Makefile `NAME :=` /
  helm `values.yaml` ports / `./alpha"` in CMD all rewrite cleanly.
- `pkg/runtime/server` raw `HandleFunc` no longer silently bypasses the
  default middleware chain.

## [0.0.0] — bootstrap

Initial commit: empty README + LICENSE.
