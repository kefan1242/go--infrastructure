# Changelog

All notable changes to this project are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
the repo uses semver once it cuts a `v0.1.0`.

## [Unreleased]

### Added
- **`pkg/middleware/retry`** — opt-in exponential-backoff + jitter retry
  middleware. Default `RetryOn` covers net errors + kratos 503/504/429.
  Wired into `pkg/client.Config.Retry` for both gRPC and HTTP — set
  `MaxAttempts > 1` per call site. Idempotency caveats documented.
- **`pkg/config.MustGetSecret(key)`** + `GetSecret` — fail-fast at boot
  when required env vars are unset. Eliminates the silent-misconfig class
  of bug where empty creds CrashLoopBackOff with confusing auth errors.
- **`docs/secrets.md`** — the rule (secrets never in `.env`), how to wire
  k8s Secret / Sealed Secrets / External Secrets / Vault, logging hygiene,
  rotation. Helm chart `values.yaml` comments updated to point here.
- **`.github/workflows/release.yml`** — fires on `v*` tag push. Buildx
  multi-arch (amd64+arm64) build, pushes to ghcr.io with SBOM + provenance,
  cosign keyless OIDC signing. Prints the verify command in the job log.
- **golang-migrate integration** — `tools/install.sh` installs `migrate`
  (v4.18.1); root Makefile gains `migrate-create/up/down/status`.
  `kris-alpha/migrations/0001_init.{up,down}.sql` is the example.
- **`scripts/smoke-test.sh` + CI `e2e` job** — boots all 3 kris-* via
  `make demo`, asserts 11 endpoint outcomes (status + body / contains),
  tears down. Catches integration regressions like the one fixed below.
- **Dockerfiles for `kris-beta` + `kris-gamma`** — same distroless / static
  pattern as alpha, with each service's own EXPOSE ports.
- **Client circuit breaker** in `pkg/client.New` / `NewHTTP` default chains
  (kratos `circuitbreaker.Client()` — Google SRE adaptive algorithm, per-op
  bucket). Returns kratos `503 / CIRCUITBREAKER` when the breaker trips.
  Opt out via `Config.NoCircuitBreaker` when caller wraps its own retry.
- `pkg/log.RedactHeaders(h, extra...)` + `RedactValue(key, value)` — mask
  sensitive HTTP headers before logging. Default list covers `Authorization`,
  `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `X-Auth-Token`,
  `X-Api-Key`, `X-Csrf-Token`. Case-insensitive, canonical keys on output,
  never mutates input. Tested + documented in observability.md.
- `make vuln-check` + dedicated CI `vuln` job — runs Go official
  `govulncheck` (call-graph aware) against every module. Fails on real CVE
  matches in the actively-called surface.
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
- `pkg/runtime/server.wrapHandler` now surfaces middleware errors via
  kratos `DefaultErrorEncoder`. Previously when auth (or any chain
  middleware) returned an error, the client got an empty `200 OK` instead
  of the proper `401` / status — silent auth bypass on raw HandleFunc'd
  routes. Caught by the new E2E smoke test.


- `scripts/new-service.sh` portable-sed compatibility (BSD/macOS): drop
  `\b`, use explicit anchors so Dockerfile EXPOSE / Makefile `NAME :=` /
  helm `values.yaml` ports / `./alpha"` in CMD all rewrite cleanly.
- `pkg/runtime/server` raw `HandleFunc` no longer silently bypasses the
  default middleware chain.

## [0.0.0] — bootstrap

Initial commit: empty README + LICENSE.
