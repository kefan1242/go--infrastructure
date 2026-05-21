# pkg/testutil

Helpers for service unit tests. Import only from `_test.go` files.

## What it provides

| File            | Provides                                                                                       |
|-----------------|------------------------------------------------------------------------------------------------|
| `logger.go`     | `MemoryLogger`: captures kratos log output into an in-memory slice for assertions              |
| `context.go`    | `NewContextWithTraceID(id)`: build a ctx carrying a trace_id                                   |
| `transport.go`  | `FakeTransport` + `InjectServerContext/InjectClientContext`: drive kratos middlewares in tests |

## Typical patterns

### Assert a Usecase logs the trace_id

```go
import "github.com/kris/go-infrastructure/pkg/testutil"

func TestUsecase_LogsTrace(t *testing.T) {
    logger, sink := testutil.NewMemoryLogger()
    uc := biz.NewFooUsecase(cfg, repo, logger)

    ctx := testutil.NewContextWithTraceID("trace-abc")
    _ = uc.Bar(ctx, "x")

    require.Contains(t, sink.LinesText(), "trace=trace-abc")
}
```

### Drive a middleware

```go
import (
    "github.com/kris/go-infrastructure/pkg/middleware/logid"
    "github.com/kris/go-infrastructure/pkg/testutil"
)

func TestLogid_Server_GeneratesIDWhenMissing(t *testing.T) {
    ft := testutil.NewFakeTransport()  // no x-trace-id header
    ctx := testutil.InjectServerContext(context.Background(), ft)

    var got string
    handler := func(ctx context.Context, _ any) (any, error) {
        got = logid.FromContext(ctx)
        return nil, nil
    }
    _, _ = logid.Server()(handler)(ctx, nil)

    require.NotEmpty(t, got)  // generated UUID
}
```

## What it explicitly doesn't do

- No gRPC / HTTP mock server: use `httptest.Server` / `bufconn` directly.
- No fixture loader: keep test data alongside the `_test.go` file in a `testdata/` dir.
