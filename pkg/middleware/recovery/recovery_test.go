package recovery_test

import (
	"context"
	"errors"
	"testing"

	pkgmetric "github.com/kris/go-infrastructure/pkg/metric"
	"github.com/kris/go-infrastructure/pkg/middleware/recovery"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		t.Fatalf("write: %v", err)
	}
	return m.GetCounter().GetValue()
}

func TestServer_RecoversPanicAndIncrementsCounter(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/svc/Boom"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	before := counterValue(t, pkgmetric.PanicsTotal.WithLabelValues("/svc/Boom"))

	mw := recovery.Server()
	_, err := mw(func(context.Context, any) (any, error) {
		panic("explode")
	})(ctx, nil)

	var k *kerrors.Error
	if !errors.As(err, &k) || k.Reason != "PANIC" {
		t.Fatalf("want kratos PANIC error, got %v", err)
	}
	after := counterValue(t, pkgmetric.PanicsTotal.WithLabelValues("/svc/Boom"))
	if after-before != 1 {
		t.Errorf("panic counter: want +1, got +%v (before=%v after=%v)", after-before, before, after)
	}
}

func TestServer_PassesThroughOnNormalReturn(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := recovery.Server()
	reply, err := mw(func(context.Context, any) (any, error) { return "ok", nil })(ctx, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply: want ok, got %v", reply)
	}
}

func TestServer_PreservesHandlerError(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := recovery.Server()
	want := errors.New("downstream boom")
	_, err := mw(func(context.Context, any) (any, error) { return nil, want })(ctx, nil)
	if !errors.Is(err, want) {
		t.Errorf("expected handler error passthrough, got %v", err)
	}
}
