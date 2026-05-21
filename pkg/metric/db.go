package metric

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// SQL pool descriptors. Each metric is keyed by `name` so a process with
// multiple connection pools can disambiguate them.
var (
	descSQLOpen = prometheus.NewDesc(
		"kris_db_pool_open_connections",
		"Open connections to the database.",
		[]string{"name"}, nil,
	)
	descSQLInUse = prometheus.NewDesc(
		"kris_db_pool_in_use",
		"Connections currently in use.",
		[]string{"name"}, nil,
	)
	descSQLIdle = prometheus.NewDesc(
		"kris_db_pool_idle",
		"Idle connections in the pool.",
		[]string{"name"}, nil,
	)
	descSQLWaitCount = prometheus.NewDesc(
		"kris_db_pool_wait_count_total",
		"Total number of connections waited for (monotonic).",
		[]string{"name"}, nil,
	)
	descSQLWaitSeconds = prometheus.NewDesc(
		"kris_db_pool_wait_seconds_total",
		"Total time blocked waiting for a connection (monotonic).",
		[]string{"name"}, nil,
	)
)

// SQLCollector is a prometheus.Collector reporting *sql.DB pool stats.
// Stats are sampled at scrape time — no background goroutine.
type SQLCollector struct {
	db   *sql.DB
	name string
}

// NewSQLCollector returns a Collector. Register it with prometheus.Register or
// the default registry; unregister on shutdown.
func NewSQLCollector(db *sql.DB, name string) *SQLCollector {
	if name == "" {
		name = "default"
	}
	return &SQLCollector{db: db, name: name}
}

// Describe implements prometheus.Collector.
func (c *SQLCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descSQLOpen
	ch <- descSQLInUse
	ch <- descSQLIdle
	ch <- descSQLWaitCount
	ch <- descSQLWaitSeconds
}

// Collect implements prometheus.Collector.
func (c *SQLCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.db.Stats()
	ch <- prometheus.MustNewConstMetric(descSQLOpen, prometheus.GaugeValue, float64(s.OpenConnections), c.name)
	ch <- prometheus.MustNewConstMetric(descSQLInUse, prometheus.GaugeValue, float64(s.InUse), c.name)
	ch <- prometheus.MustNewConstMetric(descSQLIdle, prometheus.GaugeValue, float64(s.Idle), c.name)
	ch <- prometheus.MustNewConstMetric(descSQLWaitCount, prometheus.CounterValue, float64(s.WaitCount), c.name)
	ch <- prometheus.MustNewConstMetric(descSQLWaitSeconds, prometheus.CounterValue, s.WaitDuration.Seconds(), c.name)
}

// Redis pool descriptors.
var (
	descRedisTotal = prometheus.NewDesc(
		"kris_redis_pool_total_connections",
		"Total connections in the pool.",
		[]string{"name"}, nil,
	)
	descRedisIdle = prometheus.NewDesc(
		"kris_redis_pool_idle_connections",
		"Idle connections in the pool.",
		[]string{"name"}, nil,
	)
	descRedisStale = prometheus.NewDesc(
		"kris_redis_pool_stale_connections",
		"Stale connections removed.",
		[]string{"name"}, nil,
	)
	descRedisHits = prometheus.NewDesc(
		"kris_redis_pool_hits_total",
		"Pool hits (connection reused).",
		[]string{"name"}, nil,
	)
	descRedisMisses = prometheus.NewDesc(
		"kris_redis_pool_misses_total",
		"Pool misses (new connection created).",
		[]string{"name"}, nil,
	)
	descRedisTimeouts = prometheus.NewDesc(
		"kris_redis_pool_timeouts_total",
		"Pool timeouts waiting for a connection.",
		[]string{"name"}, nil,
	)
)

// RedisCollector reports *redis.Client pool stats.
type RedisCollector struct {
	cli  *redis.Client
	name string
}

// NewRedisCollector returns a Collector.
func NewRedisCollector(cli *redis.Client, name string) *RedisCollector {
	if name == "" {
		name = "default"
	}
	return &RedisCollector{cli: cli, name: name}
}

// Describe implements prometheus.Collector.
func (c *RedisCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descRedisTotal
	ch <- descRedisIdle
	ch <- descRedisStale
	ch <- descRedisHits
	ch <- descRedisMisses
	ch <- descRedisTimeouts
}

// Collect implements prometheus.Collector.
func (c *RedisCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.cli.PoolStats()
	ch <- prometheus.MustNewConstMetric(descRedisTotal, prometheus.GaugeValue, float64(s.TotalConns), c.name)
	ch <- prometheus.MustNewConstMetric(descRedisIdle, prometheus.GaugeValue, float64(s.IdleConns), c.name)
	ch <- prometheus.MustNewConstMetric(descRedisStale, prometheus.GaugeValue, float64(s.StaleConns), c.name)
	ch <- prometheus.MustNewConstMetric(descRedisHits, prometheus.CounterValue, float64(s.Hits), c.name)
	ch <- prometheus.MustNewConstMetric(descRedisMisses, prometheus.CounterValue, float64(s.Misses), c.name)
	ch <- prometheus.MustNewConstMetric(descRedisTimeouts, prometheus.CounterValue, float64(s.Timeouts), c.name)
}
