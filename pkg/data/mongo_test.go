package data

import (
	"testing"

	"github.com/kris/go-infrastructure/pkg/metric"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.mongodb.org/mongo-driver/event"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		t.Fatalf("write: %v", err)
	}
	return m.GetCounter().GetValue()
}

func TestMongoEventHandler_Maps(t *testing.T) {
	h := mongoEventHandler("unittest")

	before := map[string]float64{
		"created":   counterValue(t, metric.MongoConnectionsCreated.WithLabelValues("unittest")),
		"closed":    counterValue(t, metric.MongoConnectionsClosed.WithLabelValues("unittest", "stale")),
		"started":   counterValue(t, metric.MongoCheckoutsStarted.WithLabelValues("unittest")),
		"failed":    counterValue(t, metric.MongoCheckoutsFailed.WithLabelValues("unittest", "timeout")),
		"succeeded": counterValue(t, metric.MongoCheckoutsSucceeded.WithLabelValues("unittest")),
		"checkins":  counterValue(t, metric.MongoCheckins.WithLabelValues("unittest")),
	}

	h(&event.PoolEvent{Type: event.ConnectionCreated})
	h(&event.PoolEvent{Type: event.ConnectionClosed, Reason: "stale"})
	h(&event.PoolEvent{Type: event.GetStarted})
	h(&event.PoolEvent{Type: event.GetFailed, Reason: "timeout"})
	h(&event.PoolEvent{Type: event.GetSucceeded})
	h(&event.PoolEvent{Type: event.ConnectionReturned})

	after := map[string]float64{
		"created":   counterValue(t, metric.MongoConnectionsCreated.WithLabelValues("unittest")),
		"closed":    counterValue(t, metric.MongoConnectionsClosed.WithLabelValues("unittest", "stale")),
		"started":   counterValue(t, metric.MongoCheckoutsStarted.WithLabelValues("unittest")),
		"failed":    counterValue(t, metric.MongoCheckoutsFailed.WithLabelValues("unittest", "timeout")),
		"succeeded": counterValue(t, metric.MongoCheckoutsSucceeded.WithLabelValues("unittest")),
		"checkins":  counterValue(t, metric.MongoCheckins.WithLabelValues("unittest")),
	}

	for name := range before {
		if delta := after[name] - before[name]; delta != 1 {
			t.Errorf("%s: want +1, got +%v", name, delta)
		}
	}
}

func TestMongoEventHandler_DefaultsEmptyReason(t *testing.T) {
	h := mongoEventHandler("def")
	before := counterValue(t, metric.MongoConnectionsClosed.WithLabelValues("def", "unknown"))
	h(&event.PoolEvent{Type: event.ConnectionClosed}) // no Reason set
	after := counterValue(t, metric.MongoConnectionsClosed.WithLabelValues("def", "unknown"))
	if after-before != 1 {
		t.Errorf("expected reason=unknown bucket to increment; before=%v after=%v", before, after)
	}
}
