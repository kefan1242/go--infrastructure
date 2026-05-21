package metric

import "github.com/prometheus/client_golang/prometheus"

// RequestsTotal is the inbound RPC request counter. Labels:
//
//	kind = grpc|http
//	op   = /pkg.svc.Method
//	code = ok|error|<kratos reason>
var RequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "kris_requests_total",
	Help: "Total number of inbound RPC requests handled by this service.",
}, []string{"kind", "op", "code"})

// RequestLatencySeconds is the inbound RPC handler latency histogram.
var RequestLatencySeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "kris_request_latency_seconds",
	Help:    "Inbound RPC handler latency in seconds.",
	Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
}, []string{"kind", "op"})

func init() {
	prometheus.MustRegister(RequestsTotal, RequestLatencySeconds)
}
