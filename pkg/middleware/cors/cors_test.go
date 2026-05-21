package cors_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kris/go-infrastructure/pkg/middleware/cors"
)

func newServer(filter func(http.Handler) http.Handler) *httptest.Server {
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("upstream:" + r.Method))
	})
	return httptest.NewServer(filter(upstream))
}

func TestCORS_NoOriginPassesThrough(t *testing.T) {
	srv := newServer(cors.New(cors.WithAllowedOrigins("https://app.example.com")))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: want 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO without Origin, got %q", got)
	}
}

func TestCORS_AllowedOriginGetsHeaders(t *testing.T) {
	srv := newServer(cors.New(
		cors.WithAllowedOrigins("https://app.example.com"),
		cors.WithAllowCredentials(true),
	))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("ACAO: want https://app.example.com, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("ACAC: want true, got %q", got)
	}
	if got := resp.Header.Get("Vary"); !strings.Contains(got, "Origin") {
		t.Errorf("Vary should include Origin, got %q", got)
	}
}

func TestCORS_DisallowedOriginPassesThroughWithoutHeaders(t *testing.T) {
	srv := newServer(cors.New(cors.WithAllowedOrigins("https://app.example.com")))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin should not get ACAO, got %q", got)
	}
	// Upstream still runs (browser will block the response client-side).
	if resp.StatusCode != http.StatusOK {
		t.Errorf("upstream should still run, status %d", resp.StatusCode)
	}
}

func TestCORS_WildcardOriginNoCredentials(t *testing.T) {
	srv := newServer(cors.New(cors.WithAllowedOrigins("*")))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Origin", "https://anywhere.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("ACAO with wildcard: want *, got %q", got)
	}
}

func TestCORS_WildcardWithCredentialsEchoesOrigin(t *testing.T) {
	// When credentials are required, "*" is illegal — we must echo the
	// concrete origin. Verify that combining the two reflects the request origin.
	srv := newServer(cors.New(
		cors.WithAllowedOrigins("*"),
		cors.WithAllowCredentials(true),
	))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Origin", "https://a.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://a.example.com" {
		t.Errorf("ACAO with credentials should echo origin, got %q", got)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	var upstreamRan bool
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRan = true
		w.WriteHeader(http.StatusOK)
	})
	filter := cors.New(
		cors.WithAllowedOrigins("https://app.example.com"),
		cors.WithAllowedMethods("GET", "POST"),
		cors.WithAllowedHeaders("Content-Type", "X-Trace-Id"),
	)
	srv := httptest.NewServer(filter(upstream))
	defer srv.Close()

	req, _ := http.NewRequest("OPTIONS", srv.URL, nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("preflight status: want 204, got %d", resp.StatusCode)
	}
	if upstreamRan {
		t.Error("upstream should not run for preflight")
	}
	for _, want := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
	} {
		if resp.Header.Get(want) == "" {
			t.Errorf("preflight missing header %s", want)
		}
	}
}

func TestCORS_OptionsWithoutACRMHeaderIsNotPreflight(t *testing.T) {
	// An OPTIONS request without Access-Control-Request-Method is just a
	// regular OPTIONS, not a CORS preflight. Should not short-circuit.
	var upstreamRan bool
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRan = true
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(cors.New(cors.WithAllowedOrigins("https://a.example.com"))(upstream))
	defer srv.Close()

	req, _ := http.NewRequest("OPTIONS", srv.URL, nil)
	req.Header.Set("Origin", "https://a.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if !upstreamRan {
		t.Error("plain OPTIONS without preflight headers should reach upstream")
	}
}
