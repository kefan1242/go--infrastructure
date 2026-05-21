// Package metric maintains shared Prometheus metric definitions.
//
// Any observability component inside pkg should register its metrics here
// instead of maintaining its own registry. The /metrics endpoint is exposed
// by pkg/runtime/server on the sidecar HTTP listener.
package metric
