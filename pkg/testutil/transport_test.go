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

func TestFakeTransport_ClientContextRoundTrip(t *testing.T) {
	ft := testutil.NewFakeTransport(
		testutil.WithOp("/svc/Method"),
		testutil.WithReqHeader("k1", "v1"),
	)
	ctx := testutil.InjectClientContext(context.Background(), ft)

	tr, ok := transport.FromClientContext(ctx)
	if !ok {
		t.Fatal("FromClientContext returned !ok")
	}
	if tr.Kind() != transport.KindGRPC {
		t.Errorf("kind: want grpc, got %s", tr.Kind())
	}
	if tr.Endpoint() != "fake://test" {
		t.Errorf("endpoint: want fake://test, got %s", tr.Endpoint())
	}
	if tr.Operation() != "/svc/Method" {
		t.Errorf("op: got %s", tr.Operation())
	}
}

func TestFakeTransport_HeaderAddKeysValues(t *testing.T) {
	ft := testutil.NewFakeTransport(
		testutil.WithReqHeader("k1", "v1"),
		testutil.WithReqHeader("k2", "v2"),
	)
	h := ft.RequestHeader()
	h.Add("k3", "v3")
	if h.Get("k3") != "v3" {
		t.Errorf("Add: want v3, got %q", h.Get("k3"))
	}
	keys := h.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys: want 3, got %d (%v)", len(keys), keys)
	}
	values := h.Values("k2")
	if len(values) != 1 || values[0] != "v2" {
		t.Errorf("Values(k2): want [v2], got %v", values)
	}
	if missing := h.Values("nope"); missing != nil {
		t.Errorf("Values(missing): want nil, got %v", missing)
	}
}

func TestFakeTransport_ReplyHeaderWritable(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ft.ReplyHeader().Set("x-trace-id", "t-1")
	if got := ft.ReplyHeader().Get("x-trace-id"); got != "t-1" {
		t.Errorf("reply header: got %q", got)
	}
}
