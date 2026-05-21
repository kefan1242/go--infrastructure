// Example: kris-gamma demonstrates a service that also dials a downstream
// gRPC service via pkg/client, plus wires readiness probes that check the
// downstream connection.
package main

import (
	"context"
	"flag"
	"os"
	"time"

	pkgclient "github.com/kris/go-infrastructure/pkg/client"
	pkglog "github.com/kris/go-infrastructure/pkg/log"
	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
	"github.com/kris/go-infrastructure/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/grpc/connectivity"
)

var (
	Name      = "kris-gamma"
	Version   = "dev"
	Commit    = ""
	BuildTime = ""

	grpcAddr  string
	otherAddr string
	upstream  string
)

func init() {
	flag.StringVar(&grpcAddr, "grpc", ":50053", "gRPC listen address")
	flag.StringVar(&otherAddr, "other", ":8085", "sidecar HTTP listen address")
	flag.StringVar(&upstream, "upstream", "kris-alpha:50051", "downstream gRPC endpoint to dial")
}

func main() {
	flag.Parse()

	id, _ := os.Hostname()
	logger := pkglog.New(Name, Version, id)
	helper := log.NewHelper(logger)

	// Dial downstream. If unreachable at boot, the example logs and continues
	// rather than failing — adapt to your own fail-fast policy.
	conn, cleanup, err := pkgclient.New(pkgclient.Config{
		Endpoint:    upstream,
		Timeout:     2 * time.Second,
		DialTimeout: 3 * time.Second,
	}, logger)
	if err != nil {
		helper.Warnf("dial upstream %q: %v (continuing without it)", upstream, err)
		cleanup = func() {}
	}
	defer cleanup()

	probes := &pkgserver.ReadinessProbes{}
	if conn != nil {
		probes.All = append(probes.All, pkgserver.Pinger{
			Name: "upstream",
			Ping: func(_ context.Context) error {
				if s := conn.GetState(); s != connectivity.Ready && s != connectivity.Idle {
					return errUpstreamNotReady
				}
				return nil
			},
		})
	}

	gs := pkgserver.NewGRPCServer(
		pkgserver.GRPCConfig{Network: "tcp", Addr: grpcAddr},
		logger,
		nil,
	)
	oh := pkgserver.NewOtherHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: otherAddr},
		logger,
		probes,
	)
	oh.S.HandleFunc("/version", version.Handler(version.Info{
		Name: Name, Version: Version, Commit: Commit, BuildTime: BuildTime,
	}))

	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(gs, oh.S),
	)
	if err := app.Run(); err != nil {
		helper.Fatalf("app run: %v", err)
	}
}

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errUpstreamNotReady = sentinelErr("upstream conn not ready")
