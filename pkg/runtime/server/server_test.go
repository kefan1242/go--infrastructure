package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"
	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
	"github.com/kris/go-infrastructure/pkg/testutil"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// freePort returns an available local TCP port for the duration of the test.
func freePort(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()
	return addr
}

func TestNewBizHTTPServer_DefaultChainRuns(t *testing.T) {
	logger, sink := testutil.NewMemoryLogger()
	addr := freePort(t)

	srv := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: addr},
		logger,
		func(s *pkgserver.BizHTTPServer) {
			s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})
		},
	)

	go func() { _ = srv.S.Start(context.Background()) }()
	t.Cleanup(func() { _ = srv.S.Stop(context.Background()) })

	// Poll until ready (Start is async).
	url := "http://" + addr + "/"
	waitForReachable(t, addr)

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body: want ok, got %q", string(body))
	}

	// logid.Server mirrors x-trace-id into reply headers.
	if got := resp.Header.Get(logid.MetadataKey); got == "" {
		t.Errorf("expected %s reply header from default chain, headers=%v", logid.MetadataKey, resp.Header)
	}

	// access log fired with code=ok.
	if !strings.Contains(sink.LinesText(), "code=ok") {
		t.Errorf("expected access log code=ok, got:\n%s", sink.LinesText())
	}
}

func TestNewBizHTTPServer_PropagatesIncomingTraceID(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	addr := freePort(t)

	srv := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: addr},
		logger,
		func(s *pkgserver.BizHTTPServer) {
			s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
		},
	)
	go func() { _ = srv.S.Start(context.Background()) }()
	t.Cleanup(func() { _ = srv.S.Stop(context.Background()) })
	waitForReachable(t, addr)

	req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
	req.Header.Set(logid.MetadataKey, "trace-from-client")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get(logid.MetadataKey); got != "trace-from-client" {
		t.Errorf("trace_id propagation: want trace-from-client, got %q", got)
	}
}

func TestNewOtherHTTPServer_ServesSidecarPaths(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	addr := freePort(t)

	probeRan := false
	probes := &pkgserver.ReadinessProbes{
		All: []pkgserver.Pinger{{
			Name: "stub",
			Ping: func(context.Context) error {
				probeRan = true
				return nil
			},
		}},
	}

	srv := pkgserver.NewOtherHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: addr},
		logger,
		probes,
	)
	go func() { _ = srv.S.Start(context.Background()) }()
	t.Cleanup(func() { _ = srv.S.Stop(context.Background()) })
	waitForReachable(t, addr)

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		resp, err := http.Get("http://" + addr + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: want 200, got %d", path, resp.StatusCode)
		}
	}
	if !probeRan {
		t.Error("readyz probe should have run")
	}
}

func TestNewBizHTTPServer_FilterRunsBeforeMiddleware(t *testing.T) {
	// A custom filter that injects a header. If the auth middleware later
	// inspects that header, it will see the filter's effect.
	logger, _ := testutil.NewMemoryLogger()
	addr := freePort(t)

	tagged := make(chan string, 1)
	tagFilter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tagged <- r.URL.Path
			next.ServeHTTP(w, r)
		})
	}

	srv := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{
			Network: "tcp", Addr: addr,
			Filters: []khttp.FilterFunc{tagFilter},
		},
		logger,
		func(s *pkgserver.BizHTTPServer) {
			s.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {})
		},
	)
	go func() { _ = srv.S.Start(context.Background()) }()
	t.Cleanup(func() { _ = srv.S.Stop(context.Background()) })
	waitForReachable(t, addr)

	resp, err := http.Get("http://" + addr + "/foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case path := <-tagged:
		if path != "/foo" {
			t.Errorf("filter saw %q, want /foo", path)
		}
	case <-time.After(time.Second):
		t.Fatal("filter never ran")
	}
}

// waitForReachable polls until the TCP port accepts a connection (2s budget).
func waitForReachable(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server not reachable at %s within 2s", addr)
}
