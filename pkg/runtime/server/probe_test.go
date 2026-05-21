package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgmetric "github.com/kris/go-infrastructure/pkg/metric"
)

func TestHealthHandler_ReturnsOK(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: want application/json, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("body: %s", w.Body.String())
	}
}

func TestReadyHandler_NilProbesReturns200(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyHandler(nil)(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	var body struct {
		Ready  bool              `json:"ready"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Ready {
		t.Errorf("ready: want true, got false (%v)", body)
	}
}

func TestReadyHandler_AllPassReturns200(t *testing.T) {
	probes := &ReadinessProbes{All: []Pinger{
		{Name: "a", Ping: func(context.Context) error { return nil }},
		{Name: "b", Ping: func(context.Context) error { return nil }},
	}}
	w := httptest.NewRecorder()
	readyHandler(probes)(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var body struct {
		Ready  bool              `json:"ready"`
		Checks map[string]string `json:"checks"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Checks["a"] != "ok" || body.Checks["b"] != "ok" {
		t.Errorf("checks: want both ok, got %+v", body.Checks)
	}
}

func TestReadyHandler_OneFailReturns503(t *testing.T) {
	probes := &ReadinessProbes{All: []Pinger{
		{Name: "good", Ping: func(context.Context) error { return nil }},
		{Name: "bad", Ping: func(context.Context) error { return errors.New("boom") }},
	}}
	w := httptest.NewRecorder()
	readyHandler(probes)(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	var body struct {
		Ready  bool              `json:"ready"`
		Checks map[string]string `json:"checks"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Ready {
		t.Errorf("ready: want false")
	}
	if !strings.Contains(body.Checks["bad"], "boom") {
		t.Errorf("checks[bad]: want fail with boom, got %q", body.Checks["bad"])
	}
}

func TestMetricsHandler_ServesPrometheusFormat(t *testing.T) {
	// Emit at least one observation so the metric shows up in the scrape.
	pkgmetric.RequestsTotal.WithLabelValues("grpc", "/probe-test", "ok").Inc()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "kris_requests_total") {
		t.Errorf("expected kris_requests_total in /metrics output, got:\n%s", body[:min(len(body), 400)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
