package data

import (
	"context"
	"fmt"
	"time"

	"github.com/kris/go-infrastructure/pkg/metric"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// RedisConfig is the cross-service Redis configuration.
type RedisConfig struct {
	Name         string        // metric label; default "default"
	Addr         string        // host:port
	Username     string        // optional
	Password     string        // optional
	DB           int           // default 0
	DialTimeout  time.Duration // default 3s
	ReadTimeout  time.Duration // default 1s
	WriteTimeout time.Duration // default 1s
	PingTimeout  time.Duration // default 3s
}

// NewRedis returns a *redis.Client plus a cleanup function. It pings at
// startup to fail fast on connectivity issues.
func NewRedis(cfg RedisConfig, logger log.Logger) (*redis.Client, func(), error) {
	if cfg.Addr == "" {
		return nil, func() {}, fmt.Errorf("redis: empty Addr")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/data/redis"))

	cli := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  defaultDuration(cfg.DialTimeout, 3*time.Second),
		ReadTimeout:  defaultDuration(cfg.ReadTimeout, time.Second),
		WriteTimeout: defaultDuration(cfg.WriteTimeout, time.Second),
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), defaultDuration(cfg.PingTimeout, 3*time.Second))
	defer cancel()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		_ = cli.Close()
		return nil, func() {}, fmt.Errorf("redis: ping: %w", err)
	}
	helper.Info("redis connected")

	collector := metric.NewRedisCollector(cli, cfg.Name)
	if err := prometheus.Register(collector); err != nil {
		helper.Warnf("redis pool collector register: %v", err)
	}

	cleanup := func() {
		prometheus.Unregister(collector)
		if err := cli.Close(); err != nil {
			helper.Errorf("redis close: %v", err)
			return
		}
		helper.Info("redis closed")
	}
	return cli, cleanup, nil
}
