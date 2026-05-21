package testutil_test

import (
	"context"
	"testing"

	"github.com/kris/go-infrastructure/pkg/testutil"

	"github.com/go-kratos/kratos/v2/transport"
)

func TestFakeTransport_ServerContext(t *testing.T) {
	ft := testutil.NewFakeTransport(
		testutil.WithKind(transport.KindHTTP),
		testutil.WithOp("/v1/foo"),
		testutil.WithReqHeader("x-trace-id", "abc"),
	)
	ctx := testutil.InjectServerContext(context.Background(), ft)

	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		t.Fatal("FromServerContext returned !ok")
	}
	if tr.Kind() != transport.KindHTTP {
		t.Errorf("kind: want http, got %s", tr.Kind())
	}
	if tr.Operation() != "/v1/foo" {
		t.Errorf("op: want /v1/foo, got %s", tr.Operation())
	}
	if tr.RequestHeader().Get("x-trace-id") != "abc" {
		t.Errorf("header: want abc, got %s", tr.RequestHeader().Get("x-trace-id"))
	}
}

func TestNewContextWithTraceID(t *testing.T) {
	ctx := testutil.NewContextWithTraceID("trace-xyz")
	_ = ctx
}
