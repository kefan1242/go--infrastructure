package client_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	pkgclient "github.com/kris/go-infrastructure/pkg/client"
	"github.com/kris/go-infrastructure/pkg/testutil"

	"google.golang.org/grpc/connectivity"
)

// accepting TCP listener that closes incoming sockets immediately. Good enough
// to verify that the kratos gRPC dial completes without business handlers.
func startAcceptingListener(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = lis.Close() })
	go func() {
		for {
			c, err := lis.Accept()
			if err != nil {
				return
			}
			go func() {
				_, _ = io.Copy(io.Discard, c)
				_ = c.Close()
			}()
		}
	}()
	return lis
}

func TestNewGRPC_HappyPath(t *testing.T) {
	lis := startAcceptingListener(t)
	logger, _ := testutil.NewMemoryLogger()

	conn, cleanup, err := pkgclient.New(pkgclient.Config{
		Endpoint:    lis.Addr().String(),
		DialTimeout: 2 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer cleanup()
	if conn == nil {
		t.Fatal("conn is nil")
	}
	if state := conn.GetState(); state == connectivity.Shutdown {
		t.Errorf("unexpected Shutdown state: %s", state)
	}
}

func TestNewGRPC_BreakerOptOut(t *testing.T) {
	// Smoke: NoCircuitBreaker=true should still construct a usable conn.
	lis := startAcceptingListener(t)
	logger, _ := testutil.NewMemoryLogger()

	conn, cleanup, err := pkgclient.New(pkgclient.Config{
		Endpoint:         lis.Addr().String(),
		DialTimeout:      2 * time.Second,
		NoCircuitBreaker: true,
	}, logger)
	if err != nil {
		t.Fatalf("New with NoCircuitBreaker: %v", err)
	}
	defer cleanup()
	if conn == nil {
		t.Fatal("conn nil")
	}
}

func TestNewGRPC_RejectsEmptyEndpoint(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	_, _, err := pkgclient.New(pkgclient.Config{}, logger)
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestNewGRPC_CleanupIsIdempotentSafe(t *testing.T) {
	// Calling cleanup more than once should not panic.
	lis := startAcceptingListener(t)
	logger, _ := testutil.NewMemoryLogger()
	_, cleanup, err := pkgclient.New(pkgclient.Config{
		Endpoint:    lis.Addr().String(),
		DialTimeout: 2 * time.Second,
		Timeout:     500 * time.Millisecond,
	}, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cleanup()
	// second call only logs the error from already-closed conn; must not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second cleanup panicked: %v", r)
		}
	}()
	cleanup()
}

// Smoke check that the dial actually used the configured DialTimeout.
// We point at a black-hole IP (1.1.1.1:81) so dial cannot complete; with a
// short DialTimeout the construction should still return cleanly because
// kratos DialInsecure is non-blocking — we just want to ensure New doesn't
// hang on an unreachable endpoint.
func TestNewGRPC_NonBlockingOnUnreachable(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	done := make(chan struct{})
	go func() {
		conn, cleanup, err := pkgclient.New(pkgclient.Config{
			Endpoint:    "1.1.1.1:81",
			DialTimeout: 200 * time.Millisecond,
		}, logger)
		_ = err
		_ = conn
		if cleanup != nil {
			cleanup()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("New blocked on unreachable endpoint (should be non-blocking)")
	}
}

// Catch a regression where the helper's `endpoint` log field would be empty,
// signalling that the config wasn't threaded through.
func TestNewGRPC_LogsEndpoint(t *testing.T) {
	lis := startAcceptingListener(t)
	logger, sink := testutil.NewMemoryLogger()
	_, cleanup, err := pkgclient.New(pkgclient.Config{
		Endpoint:    lis.Addr().String(),
		DialTimeout: 2 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer cleanup()
	out := sink.LinesText()
	if !contains(out, "endpoint=") {
		t.Errorf("expected endpoint= field in log lines, got:\n%s", out)
	}
	// ensure non-empty value
	if !contains(out, lis.Addr().String()) {
		t.Errorf("expected actual endpoint %s in log, got:\n%s", lis.Addr().String(), out)
	}
	// ensure the success log fired
	if !contains(out, "downstream connected") {
		t.Errorf("expected 'downstream connected' info line, got:\n%s", out)
	}
	_ = context.Background()
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
