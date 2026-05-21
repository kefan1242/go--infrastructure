package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoConfig is the cross-service MongoDB configuration.
type MongoConfig struct {
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

// NewMongo returns a MongoBundle plus a cleanup function.
func NewMongo(cfg MongoConfig, logger log.Logger) (*MongoBundle, func(), error) {
	if cfg.URI == "" {
		return nil, func() {}, fmt.Errorf("mongo: empty URI")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/data/mongo"))

	connCtx, cancel := context.WithTimeout(context.Background(), defaultDuration(cfg.ConnectTimeout, 10*time.Second))
	defer cancel()

	client, err := mongo.Connect(connCtx, options.Client().ApplyURI(cfg.URI))
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
