package ratelimit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kris/go-infrastructure/pkg/middleware/ratelimit"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func TestServer_AllowsThenBlocks(t *testing.T) {
	// rps=0 means no refill; burst=1 means only one request fits before block.
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := ratelimit.Server(
		ratelimit.WithRate(0, 1),
		ratelimit.WithKey(func(context.Context, string) string { return "k" }),
	)
	handler := func(context.Context, any) (any, error) { return nil, nil }

	if _, err := mw(handler)(ctx, nil); err != nil {
		t.Fatalf("first call should pass: %v", err)
	}
	_, err := mw(handler)(ctx, nil)
	var k *kerrors.Error
	if !errors.As(err, &k) || k.Code != 429 {
		t.Fatalf("second call: expected 429 kratos error, got %v", err)
	}
}

func TestServer_PerKeyBuckets(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("x-real-ip", "1.1.1.1"))
	ctx1 := testutil.InjectServerContext(context.Background(), ft)
	ft2 := testutil.NewFakeTransport(testutil.WithReqHeader("x-real-ip", "2.2.2.2"))
	ctx2 := testutil.InjectServerContext(context.Background(), ft2)

	mw := ratelimit.Server(ratelimit.WithRate(0, 1))
	handler := func(context.Context, any) (any, error) { return nil, nil }

	if _, err := mw(handler)(ctx1, nil); err != nil {
		t.Fatalf("ip1 first: %v", err)
	}
	// ip1 is now exhausted — but ip2 has its own bucket.
	if _, err := mw(handler)(ctx2, nil); err != nil {
		t.Fatalf("ip2 first should pass under its own key: %v", err)
	}
}

func TestServer_SkipsConfiguredOp(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/healthz"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := ratelimit.Server(
		ratelimit.WithRate(0, 0), // would block everything
		ratelimit.WithSkip(func(op string) bool { return op == "/healthz" }),
	)
	handler := func(context.Context, any) (any, error) { return nil, nil }
	if _, err := mw(handler)(ctx, nil); err != nil {
		t.Fatalf("skip path returned: %v", err)
	}
}
