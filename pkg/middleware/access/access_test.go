package access_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kris/go-infrastructure/pkg/middleware/access"
	"github.com/kris/go-infrastructure/pkg/middleware/logid"
	"github.com/kris/go-infrastructure/pkg/testutil"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func TestServer_LogsOKWithTraceID(t *testing.T) {
	logger, sink := testutil.NewMemoryLogger()

	ft := testutil.NewFakeTransport(testutil.WithOp("/svc/Method"))
	ctx := testutil.InjectServerContext(logid.NewContext(context.Background(), "trace-x"), ft)

	mw := access.Server(logger)
	_, err := mw(func(context.Context, any) (any, error) { return nil, nil })(ctx, nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	out := sink.LinesText()
	for _, want := range []string{"trace_id=trace-x", "op=/svc/Method", "code=ok"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestServer_LogsKratosReason(t *testing.T) {
	logger, sink := testutil.NewMemoryLogger()
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	kerr := kerrors.New(403, "FORBIDDEN", "denied")
	mw := access.Server(logger)
	_, err := mw(func(context.Context, any) (any, error) { return nil, kerr })(ctx, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(sink.LinesText(), "code=FORBIDDEN") {
		t.Errorf("expected code=FORBIDDEN; got:\n%s", sink.LinesText())
	}
}

func TestServer_LogsGenericError(t *testing.T) {
	logger, sink := testutil.NewMemoryLogger()
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	mw := access.Server(logger)
	_, _ = mw(func(context.Context, any) (any, error) { return nil, errors.New("boom") })(ctx, nil)

	out := sink.LinesText()
	if !strings.Contains(out, "code=error") {
		t.Errorf("expected code=error; got:\n%s", out)
	}
	if !strings.Contains(out, "err=boom") {
		t.Errorf("expected err=boom; got:\n%s", out)
	}
}
