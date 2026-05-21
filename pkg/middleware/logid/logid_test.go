package logid_test

import (
	"context"
	"testing"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"
	"github.com/kris/go-infrastructure/pkg/testutil"
)

func TestServer_GeneratesIDWhenMissing(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	var seen string
	handler := func(ctx context.Context, _ any) (any, error) {
		seen = logid.FromContext(ctx)
		return nil, nil
	}
	if _, err := logid.Server()(handler)(ctx, nil); err != nil {
		t.Fatalf("server: %v", err)
	}
	if seen == "" {
		t.Fatal("expected generated trace_id, got empty")
	}
	if got := ft.ReplyHeader().Get(logid.MetadataKey); got != seen {
		t.Errorf("reply header: want %s, got %s", seen, got)
	}
}

func TestServer_PropagatesExistingID(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader(logid.MetadataKey, "trace-incoming"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	var seen string
	handler := func(ctx context.Context, _ any) (any, error) {
		seen = logid.FromContext(ctx)
		return nil, nil
	}
	_, _ = logid.Server()(handler)(ctx, nil)

	if seen != "trace-incoming" {
		t.Errorf("expected propagated id, got %q", seen)
	}
	if got := ft.ReplyHeader().Get(logid.MetadataKey); got != "trace-incoming" {
		t.Errorf("reply header: want trace-incoming, got %s", got)
	}
}

func TestNewContext_RoundTrip(t *testing.T) {
	ctx := logid.NewContext(context.Background(), "trace-abc")
	if got := logid.FromContext(ctx); got != "trace-abc" {
		t.Errorf("round-trip: want trace-abc, got %s", got)
	}
}

func TestFromContext_EmptyWhenAbsent(t *testing.T) {
	if got := logid.FromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestClient_WritesHeaderWhenIDPresent(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectClientContext(logid.NewContext(context.Background(), "trace-xyz"), ft)

	handler := func(context.Context, any) (any, error) { return nil, nil }
	if _, err := logid.Client()(handler)(ctx, nil); err != nil {
		t.Fatalf("client: %v", err)
	}
	if got := ft.RequestHeader().Get(logid.MetadataKey); got != "trace-xyz" {
		t.Errorf("outbound header: want trace-xyz, got %q", got)
	}
}

func TestClient_NoHeaderWhenIDMissing(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectClientContext(context.Background(), ft)

	handler := func(context.Context, any) (any, error) { return nil, nil }
	if _, err := logid.Client()(handler)(ctx, nil); err != nil {
		t.Fatalf("client: %v", err)
	}
	if got := ft.RequestHeader().Get(logid.MetadataKey); got != "" {
		t.Errorf("outbound header should be empty when ctx has no id, got %q", got)
	}
}
