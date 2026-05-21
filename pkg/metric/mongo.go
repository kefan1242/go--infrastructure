package metric

import "github.com/prometheus/client_golang/prometheus"

// Mongo pool event counters.
//
// These are populated by pkg/data/mongo.go's PoolMonitor hook. They are
// driver-agnostic in shape — `pkg/metric` does not import the mongo driver.
var (
	MongoConnectionsCreated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_connections_created_total",
		Help: "Mongo connections created (driver pool event).",
	}, []string{"name"})

	MongoConnectionsClosed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_connections_closed_total",
		Help: "Mongo connections closed.",
	}, []string{"name", "reason"})

	MongoCheckoutsStarted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_checkouts_started_total",
		Help: "Mongo checkout attempts initiated.",
	}, []string{"name"})

	MongoCheckoutsFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_checkouts_failed_total",
		Help: "Mongo checkout attempts that failed.",
	}, []string{"name", "reason"})

	MongoCheckoutsSucceeded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_checkouts_succeeded_total",
		Help: "Mongo checkout attempts that succeeded.",
	}, []string{"name"})

	MongoCheckins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kris_mongo_pool_checkins_total",
		Help: "Mongo connections returned to the pool.",
	}, []string{"name"})
)

func init() {
	prometheus.MustRegister(
		MongoConnectionsCreated,
		MongoConnectionsClosed,
		MongoCheckoutsStarted,
		MongoCheckoutsFailed,
		MongoCheckoutsSucceeded,
		MongoCheckins,
	)
}
