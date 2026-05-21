package client

import (
	"context"
	"fmt"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HTTPConfig is the dial configuration for a downstream HTTP service.
//
// Unlike the gRPC factory, HTTP has no startup-time handshake, so there is no
// fail-fast Ping. Connection errors surface on the first RPC.
type HTTPConfig struct {
	Endpoint    string        // base URL or "host:port" (kratos discovery URIs also work)
	Timeout     time.Duration // per-request default timeout (0 = unset)
	DialTimeout time.Duration // construct-time timeout; default 5s
}

// NewHTTP returns a kratos HTTP client plus a cleanup function. It wires the
// standard recovery + tracing + logid client middlewares; additional client
// options can be passed in `extra`.
func NewHTTP(cfg HTTPConfig, logger log.Logger, extra ...khttp.ClientOption) (*khttp.Client, func(), error) {
	if cfg.Endpoint == "" {
		return nil, func() {}, fmt.Errorf("http client: empty Endpoint")
	}
	helper := log.NewHelper(log.With(logger, "module", "pkg/client/http", "endpoint", cfg.Endpoint))

	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	opts := []khttp.ClientOption{
		khttp.WithEndpoint(cfg.Endpoint),
		khttp.WithMiddleware(
			recovery.Recovery(),
			tracing.Client(),
			logid.Client(),
		),
	}
	if cfg.Timeout > 0 {
		opts = append(opts, khttp.WithTimeout(cfg.Timeout))
	}
	opts = append(opts, extra...)

	cli, err := khttp.NewClient(ctx, opts...)
	if err != nil {
		return nil, func() {}, fmt.Errorf("http client new %s: %w", cfg.Endpoint, err)
	}
	helper.Info("downstream http client ready")

	cleanup := func() {
		if err := cli.Close(); err != nil {
			helper.Errorf("http client close: %v", err)
			return
		}
		helper.Info("http client closed")
	}
	return cli, cleanup, nil
}
