package metric_test

import (
	"context"
	"errors"
	"testing"

	pkgmetric "github.com/kris/go-infrastructure/pkg/metric"
	"github.com/kris/go-infrastructure/pkg/middleware/metric"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	dto "github.com/prometheus/client_model/go"
)

func counterValue(t *testing.T, labels map[string]string) float64 {
	t.Helper()
	c, err := pkgmetric.RequestsTotal.GetMetricWith(labels)
	if err != nil {
		t.Fatalf("get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := c.(interface{ Write(*dto.Metric) error }).Write(m); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

func TestServer_IncrementsOK(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/svc/Ok"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	before := counterValue(t, map[string]string{"kind": "grpc", "op": "/svc/Ok", "code": "ok"})
	mw := metric.Server()
	_, err := mw(func(context.Context, any) (any, error) { return nil, nil })(ctx, nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	after := counterValue(t, map[string]string{"kind": "grpc", "op": "/svc/Ok", "code": "ok"})
	if after != before+1 {
		t.Errorf("ok counter: want +1, got %v -> %v", before, after)
	}
}

func TestServer_IncrementsKratosReason(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/svc/Err"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	labels := map[string]string{"kind": "grpc", "op": "/svc/Err", "code": "FORBIDDEN"}
	before := counterValue(t, labels)

	mw := metric.Server()
	_, err := mw(func(context.Context, any) (any, error) {
		return nil, kerrors.New(403, "FORBIDDEN", "no")
	})(ctx, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if after := counterValue(t, labels); after != before+1 {
		t.Errorf("reason counter: want +1, got %v -> %v", before, after)
	}
}

func TestServer_IncrementsGenericError(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithOp("/svc/Generic"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	labels := map[string]string{"kind": "grpc", "op": "/svc/Generic", "code": "error"}
	before := counterValue(t, labels)

	mw := metric.Server()
	_, _ = mw(func(context.Context, any) (any, error) { return nil, errors.New("boom") })(ctx, nil)
	if after := counterValue(t, labels); after != before+1 {
		t.Errorf("error counter: want +1, got %v -> %v", before, after)
	}
}
