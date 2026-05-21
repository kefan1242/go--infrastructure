# Middleware

go-infrastructure middlewares are plain kratos middlewares — same shape,
same composition rules — so anything you've seen in the kratos ecosystem
plugs in.

## The contract

```go
type Middleware func(Handler) Handler
type Handler    func(ctx context.Context, req any) (any, error)
```

That's it. A middleware wraps a `Handler`, returning a new `Handler`.

Composition is **outer-first**: the first middleware passed to
`middleware.Chain` sees the request first and the response last.

## The default chain

Both gRPC and business HTTP get this chain from `pkg/runtime/server`:

```
recovery -> tracing.Server -> logid.Server -> access.Server -> metric.Server
```

The sidecar listener has **no** middleware. Health, metrics, and pprof must
remain reachable even when business middleware is unhealthy.

## Adding optional middlewares

`NewGRPCServer` and `NewBizHTTPServer` accept an `extra ...middleware.Middleware`
variadic, appended to the default chain (so optional middlewares see the
trace_id already in ctx):

```go
authMW := auth.Server(auth.WithValidator(myValidator))
rlMW   := ratelimit.Server(ratelimit.WithRate(100, 200))

hs := pkgserver.NewBizHTTPServer(cfg, logger, registerFn, authMW, rlMW)
```

Order inside `extra` is the order you pass them. Pick deliberately:
`auth` usually comes before `ratelimit` so anonymous requests are rejected
without spending a token.

## Writing a new middleware

A boilerplate template:

```go
package mymw

import (
    "context"

    "github.com/go-kratos/kratos/v2/middleware"
    "github.com/go-kratos/kratos/v2/transport"
)

func Server(opts ...Option) middleware.Middleware {
    o := defaultOptions()
    for _, opt := range opts { opt(o) }
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req any) (any, error) {
            // pre-handler logic — read headers, derive a key, etc.
            tr, _ := transport.FromServerContext(ctx)
            _ = tr // tr.Operation(), tr.RequestHeader(), tr.ReplyHeader(), tr.Kind()

            reply, err := handler(ctx, req)

            // post-handler logic — observe outcome, emit metrics, etc.
            return reply, err
        }
    }
}
```

Two things to keep in mind:

1. **Don't swallow errors silently.** If you transform an error to make it
   actionable (e.g. mapping a downstream `403` to a domain error), wrap
   the original with `fmt.Errorf("... %w", err)` or a `kerrors.Error`.
2. **Return the same Go types kratos expects.** A middleware that swaps the
   reply type breaks the generated handler — the handler will type-assert
   into a generated proto struct and panic on mismatch.

## Where to put it

| Lives in pkg if…                                                | Lives in service if…                                                |
|------------------------------------------------------------------|----------------------------------------------------------------------|
| reusable across services (cross-cutting concern)                | tied to a specific business object or schema                         |
| no business deps; only ctx + headers + a generic Validator hook | needs your domain types, repository interfaces, or feature flags     |

When in doubt, start service-local. Promote to `pkg` only after a second
service needs the same logic — premature pkg additions are hard to remove.

## Testing your middleware

`pkg/testutil` provides `FakeTransport` + context injectors so you can drive a
middleware without standing up a real gRPC server:

```go
ft := testutil.NewFakeTransport(
    testutil.WithKind(transport.KindHTTP),
    testutil.WithOp("/v1/foo"),
    testutil.WithReqHeader("authorization", "Bearer t"),
)
ctx := testutil.InjectServerContext(context.Background(), ft)

mw := mymw.Server(mymw.WithValidator(v))
reply, err := mw(myHandler)(ctx, nil)
```

See `pkg/middleware/auth/auth_test.go` for a full example with skip-paths,
generic errors, and kratos-error pass-through.

## Client-side middlewares

`pkg/client` installs `recovery -> tracing.Client -> logid.Client` by default.
To plug additional client middlewares (e.g. retry, circuit breaker), use the
`extra ...grpc.DialOption` / `khttp.ClientOption` variadic on `client.New` /
`client.NewHTTP`:

```go
conn, cleanup, err := client.New(cfg, logger,
    grpc.WithDefaultServiceConfig(`{...retry policy...}`),
)
```

Or wrap the returned conn with your own interceptor.
