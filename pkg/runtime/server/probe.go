package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Pinger is a downstream-dependency probe.
type Pinger struct {
	Name string
	Ping func(ctx context.Context) error
}

// ReadinessProbes bundles the set of downstream pingers for /readyz.
// Pass nil to disable (then /readyz always returns 200).
type ReadinessProbes struct {
	All []Pinger
}

// healthHandler is the /healthz process-level liveness handler.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// readyHandler is the /readyz business-level readiness handler.
// Any failing probe returns 503; the body lists each probe's result.
func readyHandler(probes *ReadinessProbes) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := map[string]string{}
		allOK := true

		if probes != nil {
			for _, p := range probes.All {
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				err := p.Ping(ctx)
				cancel()
				if err != nil {
					result[p.Name] = "fail: " + err.Error()
					allOK = false
				} else {
					result[p.Name] = "ok"
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if !allOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ready":  allOK,
			"checks": result,
		})
	}
}

// metricsHandler exposes /metrics for Prometheus scraping.
// Uses the default registry; pkg/metric MustRegisters into it.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

// registerPprof mounts net/http/pprof handlers onto the mux.
// pprof lives on the sidecar listener, separate from the business port —
// gate it at the network layer (allowlist) in production.
func registerPprof(mux interface {
	HandleFunc(string, http.HandlerFunc)
},
) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
