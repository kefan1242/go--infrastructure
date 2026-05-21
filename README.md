# go-infrastructure

[![ci](https://github.com/kefan1242/go--infrastructure/actions/workflows/ci.yml/badge.svg)](https://github.com/kefan1242/go--infrastructure/actions/workflows/ci.yml)
[![go version](https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go)](go.work)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Reusable Go service scaffolding. The `pkg` module bundles the kratos-based
gRPC + HTTP server, observability, data-store factories, config loader and
test helpers; the `kris-*` modules are minimal example services that show
how to wire them up.

```
go-infrastructure/
├── pkg/                  # the reusable infrastructure module
├── kris-alpha/           # default chain (also the new-service + helm template)
│   └── helm-charts/      # reference Helm chart
├── kris-beta/            # default chain + auth + ratelimit
├── kris-gamma/           # gRPC server + downstream client + readiness probe
├── docs/                 # architecture / observability / middleware / getting-started
├── tools/install.sh      # pinned codegen toolchain installer
├── scripts/
│   ├── new-service.sh    # scaffold a new kris-<name>
│   └── dev/              # local-dev configs (prometheus / grafana / mysql)
├── .github/workflows/    # CI: build + vet + test + lint matrix
├── docker-compose.dev.yml  # mysql / redis / mongo / prometheus / grafana
├── Makefile              # build-all / test-all / lint / fmt / new-service
├── .golangci.yml
├── .editorconfig
├── .gitignore
├── go.work
├── LICENSE
└── README.md
```

See `docs/` for the deep dive: [getting-started](docs/getting-started.md),
[architecture](docs/architecture.md), [observability](docs/observability.md),
[middleware](docs/middleware.md).

## What's inside

| Layer            | Modules / files                                                                                                                                                            |
|------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| transport        | `pkg/runtime/server` (gRPC + biz HTTP + sidecar), `pkg/client` (gRPC + HTTP factories)                                                                                     |
| middleware       | `pkg/middleware/{recovery, access, auth, cors, logid, metric, ratelimit, timeout}` + default chain                                                                         |
| data             | `pkg/data` MySQL / Redis / Mongo factories with Prometheus pool collectors                                                                                                 |
| observability    | `pkg/trace` (OTel), `pkg/log` (text / JSON / `slog.Handler` adapter), `pkg/metric` (RED + pools + panics)                                                                  |
| config           | `pkg/config` layered `.env` + env-var loader                                                                                                                               |
| test helpers     | `pkg/testutil` (memory logger, fake transport)                                                                                                                             |
| service          | `pkg/version` build-info handler; `BizHTTPServer.HandleFunc` (chain-aware) + `S.HandleFunc` (raw)                                                                          |
| examples         | `kris-alpha` (default chain), `kris-beta` (auth + ratelimit + CORS), `kris-gamma` (downstream client + timeout + readiness probe)                                          |
| dev              | `docker-compose.dev.yml` (mysql/redis/mongo/prometheus/grafana), `make demo`, `make dev-deps-up`                                                                           |
| ci / quality     | matrix CI, golangci-lint v1.62.2, `make cover-gate` (≥75%), goleak across 14 test packages, scaffold smoke job                                                             |
| scaffolding      | `scripts/new-service.sh`, `kris-alpha/helm-charts/`, distroless Dockerfile template                                                                                        |

## Using `pkg` in your own project

The infrastructure module lives at `github.com/kris/go-infrastructure/pkg`.
Depend on it like any Go module:

```go
import (
    pkglog    "github.com/kris/go-infrastructure/pkg/log"
    pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
)
```

Then refer to the example services for working `main.go` wirings:

| Service       | Demonstrates                                                    |
|---------------|------------------------------------------------------------------|
| `kris-alpha`  | gRPC + business HTTP + sidecar (`/healthz`, `/metrics`, pprof)   |
| `kris-beta`   | adding `auth` + `ratelimit` on top of the default chain          |
| `kris-gamma`  | downstream gRPC client via `pkg/client` with a readiness probe   |

Default server middleware chain wired by `pkgserver.NewGRPCServer` /
`NewBizHTTPServer`:

```
recovery -> tracing.Server -> logid.Server -> access.Server -> metric.Server
```

Optional middlewares are appended via the `extra ...middleware.Middleware`
variadic on each constructor.

## Local development

The included `go.work` lets every example resolve `pkg` from this checkout:

```bash
make tools-install            # one-time: pinned protoc / wire / golangci-lint / etc.
make dev-deps-up              # mysql + redis + mongo + prometheus + grafana
make build-all                # build every kris-* service
make test-all                 # test pkg + every service
make lint                     # golangci-lint
```

Run a single service:

```bash
go run ./kris-alpha/cmd/alpha
curl localhost:8081/healthz
curl localhost:8081/version
curl localhost:8081/metrics
```

Scaffold a new service:

```bash
make new-service NAME=worker GRPC=50054 HTTP=8086 OTHER=8087
```

## Conventions

- **Module path** for the infrastructure is `github.com/kris/go-infrastructure/pkg`; rename to your own VCS host before publishing.
- **Service names** are the directory names under the project root. `pkg/config.NewLoader("kris-alpha")` searches `<project-root>/kris-alpha/` for env files. No hard-coded service-name prefix.
- **Metric names** use a `kris_` prefix. Adjust in `pkg/metric/prom.go` if you fork.
- **Build-time identity** uses `-ldflags "-X main.Name=... -X main.Version=... -X main.Commit=... -X main.BuildTime=..."`; the sidecar listener exposes the result at `/version` via `pkg/version.Handler`.
- **GOMAXPROCS** is set from the container's CPU quota by blank-importing `go.uber.org/automaxprocs` in each service's main.go.
- **No business logic** lives in `pkg`. `pkg/third_party/` ships the standard kratos / Google / envoy validate `.proto` descriptors.
