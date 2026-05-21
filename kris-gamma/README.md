# kris-gamma

Example service demonstrating:

- a **downstream gRPC client** dialed through `pkg/client` (recovery + tracing + logid client middlewares)
- a **readiness probe** that fails `/readyz` when the downstream connection isn't `Ready`/`Idle`
- gRPC server on `:50053` and sidecar HTTP on `:8085`; no business HTTP

## Run

```bash
go run ./cmd/gamma -upstream=kris-alpha:50051
curl localhost:8085/readyz
```

The probe surfaces an `upstream: fail: ...` entry in the JSON response when
the downstream connection isn't healthy.
