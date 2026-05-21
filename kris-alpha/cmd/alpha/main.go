// Example: kris-alpha is a minimal service demonstrating the standard
// gRPC + HTTP + sidecar wiring with only the default middleware chain,
// plus the recommended automaxprocs blank import and /version endpoint.
package main

import (
	"flag"
	"net/http"
	"os"

	pkglog "github.com/kris/go-infrastructure/pkg/log"
	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
	"github.com/kris/go-infrastructure/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	_ "go.uber.org/automaxprocs"
)

// Build-time identity. Override via:
//
//	go build -ldflags "-X main.Name=... -X main.Version=... \
//	                   -X main.Commit=... -X main.BuildTime=..."
var (
	Name      = "kris-alpha"
	Version   = "dev"
	Commit    = ""
	BuildTime = ""

	grpcAddr  string
	httpAddr  string
	otherAddr string
)

func init() {
	flag.StringVar(&grpcAddr, "grpc", ":50051", "gRPC listen address")
	flag.StringVar(&httpAddr, "http", ":8080", "business HTTP listen address")
	flag.StringVar(&otherAddr, "other", ":8081", "sidecar HTTP listen address (healthz/metrics/pprof)")
}

func main() {
	flag.Parse()

	id, _ := os.Hostname()
	logger := pkglog.New(Name, Version, id)

	app, cleanup := newApp(logger, id)
	defer cleanup()

	if err := app.Run(); err != nil {
		log.NewHelper(logger).Fatalf("app run: %v", err)
	}
}

func newApp(logger log.Logger, id string) (*kratos.App, func()) {
	gs := pkgserver.NewGRPCServer(
		pkgserver.GRPCConfig{Network: "tcp", Addr: grpcAddr},
		logger,
		// nil registrar: no business gRPC handlers in this example.
		nil,
	)

	hs := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: httpAddr},
		logger,
		func(s *pkgserver.BizHTTPServer) {
			// Use s.HandleFunc (not s.S.HandleFunc) so the default chain wraps
			// this handler — otherwise raw HandleFunc bypasses kratos middleware.
			s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("kris-alpha ok"))
			})
		},
	)

	oh := pkgserver.NewOtherHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: otherAddr},
		logger,
		nil, // no readiness probes wired in this example
	)
	oh.S.HandleFunc("/version", version.Handler(version.Info{
		Name:      Name,
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}))

	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(gs, hs.S, oh.S),
	)
	return app, func() {}
}
