//go:build integration

package data_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/data"
	"github.com/kris/go-infrastructure/pkg/metric"
	"github.com/kris/go-infrastructure/pkg/testutil"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.mongodb.org/mongo-driver/bson"
)

const defaultMongoURI = "mongodb://127.0.0.1:27017"

func mongoURI() string {
	if v := os.Getenv("KRIS_MONGO_URI"); v != "" {
		return v
	}
	return defaultMongoURI
}

func counterTotal(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := c.Write(m); err != nil {
		t.Fatalf("counter write: %v", err)
	}
	return m.GetCounter().GetValue()
}

func TestIntegrationMongo_ConnectPingRoundTripEmitsEvents(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	cfg := data.MongoConfig{
		Name:           "integration",
		URI:            mongoURI(),
		Database:       "kris_integration_test",
		ConnectTimeout: 5 * time.Second,
		PingTimeout:    2 * time.Second,
	}
	bundle, cleanup, err := data.NewMongo(cfg, logger)
	if err != nil {
		t.Fatalf("NewMongo: %v (is the dev mongo running on %s?)", err, cfg.URI)
	}
	defer cleanup()
	if bundle == nil || bundle.Database == nil {
		t.Fatal("bundle / database nil")
	}

	created0 := counterTotal(t, metric.MongoConnectionsCreated.WithLabelValues("integration"))
	checkouts0 := counterTotal(t, metric.MongoCheckoutsSucceeded.WithLabelValues("integration"))

	// Round-trip: insert, find, delete. Each command does a checkout/checkin
	// and (often) a fresh connection creation under PoolMonitor.
	coll := bundle.Database.Collection("probe")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := coll.InsertOne(ctx, bson.M{"k": "v"}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var got bson.M
	if err := coll.FindOne(ctx, bson.M{"k": "v"}).Decode(&got); err != nil {
		t.Fatalf("find: %v", err)
	}
	if _, err := coll.DeleteMany(ctx, bson.M{"k": "v"}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	created := counterTotal(t, metric.MongoConnectionsCreated.WithLabelValues("integration"))
	checkouts := counterTotal(t, metric.MongoCheckoutsSucceeded.WithLabelValues("integration"))

	// The pool may have already minted a connection at Connect time; the
	// round-trip can then reuse it without firing another `created`. So we
	// require checkouts to advance, but treat created as best-effort.
	if checkouts <= checkouts0 {
		t.Errorf("expected checkouts_succeeded to increase: before=%v after=%v", checkouts0, checkouts)
	}
	if created < created0 {
		t.Errorf("connections_created went backwards: before=%v after=%v", created0, created)
	}
}

func TestIntegrationMongo_RejectsEmptyURI(t *testing.T) {
	logger, _ := testutil.NewMemoryLogger()
	_, _, err := data.NewMongo(data.MongoConfig{}, logger)
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
}
