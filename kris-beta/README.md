# kris-beta

Example service demonstrating **optional middlewares** (`auth`, `ratelimit`)
layered on top of the default chain. HTTP-only; no gRPC server.

- business HTTP on `:8082` with `auth.Server` + `ratelimit.Server` appended
- `GET /` is in the auth skip list (public)
- `GET /whoami` requires a `Bearer demo-<subject>` token
- sidecar HTTP on `:8083`

## Run

```bash
go run ./cmd/beta
curl localhost:8082/                                        # 200
curl localhost:8082/whoami                                  # 401
curl -H "Authorization: Bearer demo-alice" localhost:8082/whoami  # hello, alice
```
