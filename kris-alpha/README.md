# kris-alpha

Minimal example service demonstrating the **standard wiring**:

- gRPC server on `:50051`
- business HTTP on `:8080`
- sidecar HTTP on `:8081` (`/healthz`, `/readyz`, `/metrics`, `/debug/pprof`)
- default middleware chain only (recovery -> tracing -> logid -> access -> metric)

No business logic — registers a single `GET /` HTTP handler so the HTTP server
has something to serve.

Also serves as the **template** that `scripts/new-service.sh` copies from.

## Run

```bash
go run ./cmd/alpha
# or:
make run

curl localhost:8080/
curl localhost:8081/healthz
curl localhost:8081/version
curl localhost:8081/metrics
```

## Build with versioned identity

```bash
go build \
  -ldflags "-X main.Name=kris-alpha \
            -X main.Version=$(git describe --tags --always) \
            -X main.Commit=$(git rev-parse HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o ./bin/alpha ./cmd/alpha
```
