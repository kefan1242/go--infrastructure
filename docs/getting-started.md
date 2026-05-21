# Getting started

This guide takes a fresh checkout of go-infrastructure and gets you to a
running service with logs, metrics, and traces in about five minutes.

## Prerequisites

- Go **1.25+**
- Docker (for the local dev stack)
- `make`, `git`

## 1. Tools and local deps

```bash
make tools-install   # pinned protoc-gen-* / wire / golangci-lint / etc.
make dev-deps-up     # mysql + redis + mongo + prometheus + grafana
```

The dev stack listens on:

| Service     | Host port                          |
|-------------|-------------------------------------|
| MySQL       | `127.0.0.1:3306`                    |
| Redis       | `127.0.0.1:6379`                    |
| MongoDB     | `127.0.0.1:27017`                   |
| Prometheus  | `127.0.0.1:9090`                    |
| Grafana     | `127.0.0.1:3000` (no login, anon admin) |

Volumes live under `./.dev-data/` (`.gitignored`).

## 2. Run an example service

```bash
go run ./kris-alpha/cmd/alpha
```

In another shell:

```bash
curl localhost:8080/                  # business HTTP
curl localhost:8081/healthz           # sidecar liveness
curl localhost:8081/readyz            # readiness (probes downstream deps)
curl localhost:8081/version           # build info
curl localhost:8081/metrics           # Prometheus
```

Open `http://localhost:9090/targets` in Prometheus — you should see the kris-*
sidecar endpoints listed as scrape targets.

## 3. Scaffold a new service

```bash
make new-service NAME=worker GRPC=50054 HTTP=8086 OTHER=8087
```

This creates `kris-worker/` from the `kris-alpha` template, rewires
identifiers, patches default ports, adds the new module to `go.work`, runs
`go mod tidy`, and builds.

```bash
./kris-worker/bin/worker
curl localhost:8087/version
```

## 4. Add a handler

Open `kris-worker/cmd/worker/main.go`. The template registers a default
`GET /` handler — add more next to it. For gRPC, register a generated
service inside the `pkgserver.NewGRPCServer` callback:

```go
gs := pkgserver.NewGRPCServer(cfg, logger, func(s *grpc.Server) {
    workerv1.RegisterWorkerServiceServer(s, mySvc)
})
```

## 5. Add a `.proto` (when ready)

Drop your file under `kris-worker/api/worker/v1/worker.proto`. The
service-level `Makefile` already has `api` / `errors` / `validate` targets
that no-op when there's no `.proto` — once you create one, run:

```bash
cd kris-worker
make api      # generates *.pb.go, *_grpc.pb.go, *_http.pb.go, openapi.yaml
make tidy
make build
```

Shared `.proto` includes (kratos errors, validate, google api) live in
`pkg/third_party/` — the Makefile already wires the `--proto_path`.

## 6. Add a downstream client

```go
import pkgclient "github.com/kris/go-infrastructure/pkg/client"

conn, cleanup, err := pkgclient.New(pkgclient.Config{
    Endpoint:    "kris-alpha:50051",
    Timeout:     2 * time.Second,
    DialTimeout: 3 * time.Second,
}, logger)
if err != nil { return err }
defer cleanup()
```

`recovery`, `tracing.Client`, and `logid.Client` are pre-wired. Pass extra
DialOptions to the variadic when you need TLS, retry policy, or service
discovery.

See `kris-gamma/cmd/gamma/main.go` for a working example, including how to
publish the connection state as a `/readyz` probe.

## 7. Build with versioned identity

```bash
go build \
  -ldflags "-X main.Name=kris-worker \
            -X main.Version=$(git describe --tags --always) \
            -X main.Commit=$(git rev-parse HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o ./bin/worker ./cmd/worker
```

The `kris-alpha/Makefile` template already wires `-X main.Version=$(git describe)`.
Extend the `build` target for the other fields if you need them.

## 8. Containerize

`kris-alpha/Dockerfile` is a two-stage build (`golang` → `debian-slim`).
Copy it into your service and update the binary name + ports.

```bash
cd kris-worker
cp ../kris-alpha/Dockerfile .
# edit binary name and EXPOSE ports
docker build -t kris-worker:dev .
```

## Where to go next

- [Architecture](./architecture.md) — module map, listener layout, defaults
- [Observability](./observability.md) — log fields, metric catalog, OTel
- [Middleware](./middleware.md) — writing your own + testing pattern
