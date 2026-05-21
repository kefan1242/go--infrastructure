# kris-beta

Example service demonstrating **optional middlewares** (`auth`, `ratelimit`)
and an **HTTP filter** (`cors`) layered on top of the default chain.
HTTP-only; no gRPC server.

- business HTTP on `:8082` with CORS (open) + `auth.Server` + `ratelimit.Server`
- `GET /` is in the auth skip list (public)
- `GET /whoami` requires a `Bearer demo-<subject>` token
- preflight `OPTIONS` short-circuits with 204 + CORS headers, no auth gate
- sidecar HTTP on `:8083`

## Run

```bash
go run ./cmd/beta
curl localhost:8082/                                        # 200
curl localhost:8082/whoami                                  # 401
curl -H "Authorization: Bearer demo-alice" localhost:8082/whoami  # hello, alice
```
