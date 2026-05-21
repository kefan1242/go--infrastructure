//go:build integration

package data_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/data"
	"github.com/kris/go-infrastructure/pkg/testutil"
)

func redisAddr() string {
	if v := os.Getenv("KRIS_REDIS_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:6379"
}

func TestIntegrationRedis_ConnectAndRoundTrip(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	cli, cleanup, err := data.NewRedis(data.RedisConfig{
		Name:        "integration",
		Addr:        redisAddr(),
		DialTimeout: 2 * time.Second,
		PingTimeout: 2 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("NewRedis: %v (is the dev redis running on %s?)", err, redisAddr())
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := "kris-integration-probe"

	if err := cli.Set(ctx, key, "hello", time.Minute).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := cli.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "hello" {
		t.Errorf("get: want hello, got %q", v)
	}
	if err := cli.Del(ctx, key).Err(); err != nil {
		t.Errorf("del: %v", err)
	}

	stats := cli.PoolStats()
	if stats.TotalConns == 0 && stats.Hits+stats.Misses == 0 {
		t.Errorf("pool stats look empty: %+v", stats)
	}
}

func TestIntegrationRedis_RejectsEmptyAddr(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	_, _, err := data.NewRedis(data.RedisConfig{}, logger)
	if err == nil {
		t.Fatal("expected error for empty addr")
	}
}
