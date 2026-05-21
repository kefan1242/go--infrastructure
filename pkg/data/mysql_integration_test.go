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

func mysqlDSN() string {
	if v := os.Getenv("KRIS_MYSQL_DSN"); v != "" {
		return v
	}
	// No db in the DSN — `SELECT 1` doesn't need one. Avoids dependence on
	// scripts/dev/mysql-init.sql having run.
	return "root:@tcp(127.0.0.1:3306)/?parseTime=true"
}

func TestIntegrationMySQL_ConnectAndQuery(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	db, cleanup, err := data.NewMySQL(data.MySQLConfig{
		Name:        "integration",
		DSN:         mysqlDSN(),
		PingTimeout: 2 * time.Second,
	}, logger)
	if err != nil {
		t.Fatalf("NewMySQL: %v (is the dev mysql running on 127.0.0.1:3306 with db `kris`?)", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var n int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&n); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if n != 1 {
		t.Errorf("want 1, got %d", n)
	}

	stats := db.Stats()
	if stats.OpenConnections == 0 {
		t.Errorf("expected at least 1 open conn after query, got %+v", stats)
	}
}

func TestIntegrationMySQL_RejectsEmptyDSN(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	_, _, err := data.NewMySQL(data.MySQLConfig{}, logger)
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}
