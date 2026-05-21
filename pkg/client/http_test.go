package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pkgclient "github.com/kris/go-infrastructure/pkg/client"
	"github.com/kris/go-infrastructure/pkg/testutil"
)

func TestNewHTTP_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	logger, _ := testutil.NewMemoryLogger()
	cli, cleanup, err := pkgclient.NewHTTP(pkgclient.HTTPConfig{
		Endpoint:    srv.URL,
		Timeout:     time.Second,
		DialTimeout: 2 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("NewHTTP: %v", err)
	}
	defer cleanup()
	if cli == nil {
		t.Fatal("client is nil")
	}
}

func TestNewHTTP_RejectsEmptyEndpoint(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	_, _, err := pkgclient.NewHTTP(pkgclient.HTTPConfig{}, logger)
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestNewHTTP_LogsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	logger, sink := testutil.NewMemoryLogger()
	_, cleanup, err := pkgclient.NewHTTP(pkgclient.HTTPConfig{
		Endpoint: srv.URL,
	}, logger)
	if err != nil {
		t.Fatalf("NewHTTP: %v", err)
	}
	defer cleanup()

	out := sink.LinesText()
	if !contains(out, "downstream http client ready") {
		t.Errorf("expected ready info line, got:\n%s", out)
	}
	if !contains(out, srv.URL) {
		t.Errorf("expected endpoint %s in log, got:\n%s", srv.URL, out)
	}
}

func TestNewHTTP_CleanupIsIdempotentSafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	logger, _ := testutil.NewMemoryLogger()
	_, cleanup, err := pkgclient.NewHTTP(pkgclient.HTTPConfig{Endpoint: srv.URL}, logger)
	if err != nil {
		t.Fatalf("NewHTTP: %v", err)
	}
	cleanup()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second cleanup panicked: %v", r)
		}
	}()
	cleanup()
}
