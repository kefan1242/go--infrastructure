package timeout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/timeout"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func TestServer_FastHandlerPassesThrough(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := timeout.Server(timeout.WithTimeout(time.Second))
	reply, err := mw(func(context.Context, any) (any, error) { return "ok", nil })(ctx, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply: want ok, got %v", reply)
	}
}

func TestServer_SlowHandlerReturnsDeadlineExceeded(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := timeout.Server(timeout.WithTimeout(50 * time.Millisecond))
	handler := func(ctx context.Context, _ any) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return "late", nil
		}
	}
	start := time.Now()
	_, err := mw(handler)(ctx, nil)
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("middleware should return promptly on deadline, took %v", elapsed)
	}
	var k *kerrors.Error
	if !errors.As(err, &k) || k.Code != 504 {
		t.Fatalf("want kratos 504 gateway timeout, got %v", err)
	}
	if k.Reason != "DEADLINE_EXCEEDED" {
		t.Errorf("reason: want DEADLINE_EXCEEDED, got %s", k.Reason)
	}
}

func TestServer_ZeroTimeoutIsPassthrough(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := timeout.Server(timeout.WithTimeout(0)) // disabled
	called := false
	handler := func(ctx context.Context, _ any) (any, error) {
		called = true
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			t.Error("expected no deadline on ctx when timeout=0")
		}
		return nil, nil
	}
	if _, err := mw(handler)(ctx, nil); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !called {
		t.Error("handler should have run")
	}
}

func TestServer_SkipBypassesTimeout(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/stream/Watch"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := timeout.Server(
		timeout.WithTimeout(50*time.Millisecond),
		timeout.WithSkip(func(op string) bool { return op == "/stream/Watch" }),
	)
	handler := func(ctx context.Context, _ any) (any, error) {
		// no deadline since skip applied
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			t.Error("expected no deadline on skip path")
		}
		time.Sleep(150 * time.Millisecond)
		return "ok", nil
	}
	reply, err := mw(handler)(ctx, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply: want ok, got %v", reply)
	}
}

func TestServer_HandlerErrorPassesThrough(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := timeout.Server(timeout.WithTimeout(time.Second))
	boom := errors.New("boom")
	_, err := mw(func(context.Context, any) (any, error) { return nil, boom })(ctx, nil)
	if !errors.Is(err, boom) {
		t.Errorf("expected boom passthrough, got %v", err)
	}
}
