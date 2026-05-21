package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kris/go-infrastructure/pkg/middleware/auth"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func passValidator(_ context.Context, token string) (*auth.Claims, error) {
	if token == "ok" {
		return &auth.Claims{Subject: "alice"}, nil
	}
	return nil, errors.New("nope")
}

func TestServer_RejectsMissingToken(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := auth.Server(auth.WithValidator(passValidator))
	_, err := mw(func(context.Context, any) (any, error) { return nil, nil })(ctx, nil)

	var k *kerrors.Error
	if !errors.As(err, &k) || k.Code != 401 {
		t.Fatalf("expected 401 kratos error, got %v", err)
	}
}

func TestServer_AcceptsValidTokenAndInjectsClaims(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("authorization", "Bearer ok"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	var got *auth.Claims
	handler := func(ctx context.Context, _ any) (any, error) {
		c, ok := auth.FromContext(ctx)
		if !ok {
			t.Fatal("expected claims in ctx")
		}
		got = c
		return nil, nil
	}
	mw := auth.Server(auth.WithValidator(passValidator))
	if _, err := mw(handler)(ctx, nil); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got == nil || got.Subject != "alice" {
		t.Errorf("subject: want alice, got %+v", got)
	}
}

func TestServer_SkipsConfiguredOps(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/public/Health"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	called := false
	mw := auth.Server(
		auth.WithValidator(passValidator),
		auth.WithSkip(func(op string) bool { return op == "/public/Health" }),
	)
	_, err := mw(func(context.Context, any) (any, error) { called = true; return nil, nil })(ctx, nil)
	if err != nil {
		t.Fatalf("skip path returned: %v", err)
	}
	if !called {
		t.Fatal("handler should have run on skip path")
	}
}

func TestServer_PanicsWithoutValidator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when WithValidator omitted")
		}
	}()
	_ = auth.Server()
}

func TestServer_PassesThroughKratosValidatorError(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("authorization", "Bearer bad"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	v := func(_ context.Context, _ string) (*auth.Claims, error) {
		return nil, kerrors.New(403, "FORBIDDEN", "go away")
	}
	mw := auth.Server(auth.WithValidator(v))
	_, err := mw(func(context.Context, any) (any, error) { return nil, nil })(ctx, nil)

	var k *kerrors.Error
	if !errors.As(err, &k) || k.Reason != "FORBIDDEN" {
		t.Fatalf("want FORBIDDEN kratos error, got %v", err)
	}
}
