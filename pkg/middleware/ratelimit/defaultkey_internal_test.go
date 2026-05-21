package ratelimit

import (
	"context"
	"net"
	"testing"

	"github.com/kris/go-infrastructure/pkg/testutil"

	"google.golang.org/grpc/peer"
)

// White-box tests for defaultKey — the public Server() only exposes its
// behavior indirectly. These pin down the precedence:
//
//	x-forwarded-for first hop -> x-real-ip -> gRPC peer addr -> "global"

func TestDefaultKey_PrefersXForwardedForFirstHop(t *testing.T) {
	ft := testutil.NewFakeTransport(
		testutil.WithReqHeader("x-forwarded-for", "10.0.0.1, 192.168.1.1, 172.16.0.1"),
		testutil.WithReqHeader("x-real-ip", "8.8.8.8"),
	)
	ctx := testutil.InjectServerContext(context.Background(), ft)

	if got := defaultKey(ctx, ""); got != "10.0.0.1" {
		t.Errorf("xff first hop: want 10.0.0.1, got %q", got)
	}
}

func TestDefaultKey_XForwardedForSingleHop(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("x-forwarded-for", "10.0.0.42"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	if got := defaultKey(ctx, ""); got != "10.0.0.42" {
		t.Errorf("xff single: want 10.0.0.42, got %q", got)
	}
}

func TestDefaultKey_XForwardedForTrimsWhitespace(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("x-forwarded-for", "  10.0.0.7  , 192.168.1.1"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	if got := defaultKey(ctx, ""); got != "10.0.0.7" {
		t.Errorf("xff trim: want 10.0.0.7, got %q", got)
	}
}

func TestDefaultKey_FallsBackToXRealIP(t *testing.T) {
	ft := testutil.NewFakeTransport(testutil.WithReqHeader("x-real-ip", "1.2.3.4"))
	ctx := testutil.InjectServerContext(context.Background(), ft)

	if got := defaultKey(ctx, ""); got != "1.2.3.4" {
		t.Errorf("x-real-ip: want 1.2.3.4, got %q", got)
	}
}

func TestDefaultKey_FallsBackToGRPCPeer(t *testing.T) {
	ft := testutil.NewFakeTransport() // no IP headers
	ctx := testutil.InjectServerContext(context.Background(), ft)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("203.0.113.5"), Port: 1234},
	})

	if got := defaultKey(ctx, ""); got != "203.0.113.5" {
		t.Errorf("grpc peer: want 203.0.113.5, got %q", got)
	}
}

func TestDefaultKey_GlobalFallback(t *testing.T) {
	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	if got := defaultKey(ctx, ""); got != "global" {
		t.Errorf("fallback: want global, got %q", got)
	}
}

func TestDefaultKey_NoTransportContextFallsBackToGlobal(t *testing.T) {
	// no transport in ctx at all
	if got := defaultKey(context.Background(), ""); got != "global" {
		t.Errorf("no transport: want global, got %q", got)
	}
}
