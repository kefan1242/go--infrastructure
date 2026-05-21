package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/kris/go-infrastructure/pkg/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus"
)

// MySQLConfig is the cross-service MySQL configuration.
// Intentionally minimal: DSN + pool + timeouts. Translate from your own
// service config struct on the caller side.
type MySQLConfig struct {
	Name            string        // metric label; default "default"
	DSN             string        // user:pass@tcp(host:port)/dbname?...
	MaxOpenConns    int           // default 10
	MaxIdleConns    int           // default 5
	ConnMaxLifetime time.Duration // default 30m
	PingTimeout     time.Duration // default 3s
}

// NewMySQL returns *sql.DB plus a cleanup function. It pings the database at
// startup so connectivity problems fail fast instead of surfacing on the first
// request.
func NewMySQL(cfg MySQLConfig, logger log.Logger) (*sql.DB, func(), error) {
	if cfg.DSN == "" {
		return nil, func() {}, fmt.Errorf("mysql: empty DSN")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/data/mysql"))

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, func() {}, fmt.Errorf("mysql: open: %w", err)
	}

	db.SetMaxOpenConns(defaultInt(cfg.MaxOpenConns, 10))
	db.SetMaxIdleConns(defaultInt(cfg.MaxIdleConns, 5))
	db.SetConnMaxLifetime(defaultDuration(cfg.ConnMaxLifetime, 30*time.Minute))

	pingCtx, cancel := context.WithTimeout(context.Background(), defaultDuration(cfg.PingTimeout, 3*time.Second))
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, func() {}, fmt.Errorf("mysql: ping: %w", err)
	}
	helper.Info("mysql connected")

	collector := metric.NewSQLCollector(db, cfg.Name)
	if err := prometheus.Register(collector); err != nil {
		// already registered (e.g. multiple pools with same name) — log but don't fail.
		helper.Warnf("mysql pool collector register: %v", err)
	}

	cleanup := func() {
		prometheus.Unregister(collector)
		if err := db.Close(); err != nil {
			helper.Errorf("mysql close: %v", err)
			return
		}
		helper.Info("mysql closed")
	}
	return db, cleanup, nil
}

func defaultInt(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}

func defaultDuration(v, d time.Duration) time.Duration {
	if v == 0 {
		return d
	}
	return v
}
