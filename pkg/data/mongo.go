package data

import (
	"context"
	"fmt"
	"time"

	"github.com/kris/go-infrastructure/pkg/metric"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoConfig is the cross-service MongoDB configuration.
type MongoConfig struct {
	Name           string        // metric label; default "default"
	URI            string        // mongodb://user:pass@host:port/dbname
	Database       string        // logical database name used by the service
	ConnectTimeout time.Duration // default 10s
	PingTimeout    time.Duration // default 3s
}

// MongoBundle bundles the connected client with the resolved business
// database so callers don't have to call client.Database(name) again.
type MongoBundle struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongo returns a MongoBundle plus a cleanup function. The client is
// configured with a PoolMonitor that increments pkg/metric.Mongo* counters
// — no separate Collector to register.
func NewMongo(cfg MongoConfig, logger log.Logger) (*MongoBundle, func(), error) {
	if cfg.URI == "" {
		return nil, func() {}, fmt.Errorf("mongo: empty URI")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/data/mongo"))

	name := cfg.Name
	if name == "" {
		name = "default"
	}

	connCtx, cancel := context.WithTimeout(context.Background(), defaultDuration(cfg.ConnectTimeout, 10*time.Second))
	defer cancel()

	opts := options.Client().
		ApplyURI(cfg.URI).
		SetPoolMonitor(&event.PoolMonitor{Event: mongoEventHandler(name)})

	client, err := mongo.Connect(connCtx, opts)
	if err != nil {
		return nil, func() {}, fmt.Errorf("mongo: connect: %w", err)
	}

	pingCtx, pcancel := context.WithTimeout(context.Background(), defaultDuration(cfg.PingTimeout, 3*time.Second))
	defer pcancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, func() {}, fmt.Errorf("mongo: ping: %w", err)
	}
	helper.Info("mongo connected")

	bundle := &MongoBundle{Client: client, Database: client.Database(cfg.Database)}

	cleanup := func() {
		ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		if err := client.Disconnect(ctx); err != nil {
			helper.Errorf("mongo disconnect: %v", err)
			return
		}
		helper.Info("mongo disconnected")
	}
	return bundle, cleanup, nil
}

// mongoEventHandler returns the PoolMonitor.Event hook bound to `name`.
// Event-type strings come from go.mongodb.org/mongo-driver/event constants.
func mongoEventHandler(name string) func(*event.PoolEvent) {
	return func(e *event.PoolEvent) {
		switch e.Type {
		case event.ConnectionCreated:
			metric.MongoConnectionsCreated.WithLabelValues(name).Inc()
		case event.ConnectionClosed:
			reason := e.Reason
			if reason == "" {
				reason = "unknown"
			}
			metric.MongoConnectionsClosed.WithLabelValues(name, reason).Inc()
		case event.GetStarted:
			metric.MongoCheckoutsStarted.WithLabelValues(name).Inc()
		case event.GetFailed:
			reason := e.Reason
			if reason == "" {
				reason = "unknown"
			}
			metric.MongoCheckoutsFailed.WithLabelValues(name, reason).Inc()
		case event.GetSucceeded:
			metric.MongoCheckoutsSucceeded.WithLabelValues(name).Inc()
		case event.ConnectionReturned:
			metric.MongoCheckins.WithLabelValues(name).Inc()
		}
	}
}
