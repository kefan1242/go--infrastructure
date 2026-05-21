// Example: kris-beta demonstrates plugging optional middlewares (auth +
// ratelimit) and an HTTP filter (cors) on top of the default chain.
// HTTP-only; no gRPC server.
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strings"

	pkglog "github.com/kris/go-infrastructure/pkg/log"
	"github.com/kris/go-infrastructure/pkg/middleware/auth"
	"github.com/kris/go-infrastructure/pkg/middleware/cors"
	"github.com/kris/go-infrastructure/pkg/middleware/ratelimit"
	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
	"github.com/kris/go-infrastructure/pkg/version"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	_ "go.uber.org/automaxprocs"
)

var (
	Name      = "kris-beta"
	Version   = "dev"
	Commit    = ""
	BuildTime = ""

	httpAddr  string
	otherAddr string
)

func init() {
	flag.StringVar(&httpAddr, "http", ":8082", "business HTTP listen address")
	flag.StringVar(&otherAddr, "other", ":8083", "sidecar HTTP listen address")
}

func main() {
	flag.Parse()

	id, _ := os.Hostname()
	logger := pkglog.New(Name, Version, id)

	// Demo validator: accept any token starting with "demo-" and treat the
	// remainder as the subject. Replace with a JWT decoder / IAM lookup.
	validator := func(_ context.Context, token string) (*auth.Claims, error) {
		if !strings.HasPrefix(token, "demo-") {
			return nil, auth.ErrUnauthorized
		}
		return &auth.Claims{Subject: strings.TrimPrefix(token, "demo-")}, nil
	}

	authMW := auth.Server(
		auth.WithValidator(validator),
		auth.WithSkip(func(op string) bool { return op == "/" }),
	)
	rlMW := ratelimit.Server(ratelimit.WithRate(50, 100))

	corsFilter := cors.New(
		cors.WithAllowedOrigins("*"),
		cors.WithAllowedMethods("GET", "POST", "OPTIONS"),
	)

	hs := pkgserver.NewBizHTTPServer(
		pkgserver.HTTPConfig{
			Network: "tcp",
			Addr:    httpAddr,
			Filters: []khttp.FilterFunc{corsFilter},
		},
		logger,
		func(s *khttp.Server) {
			s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("kris-beta public ok"))
			})
			s.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "no claims", http.StatusUnauthorized)
					return
				}
				_, _ = w.Write([]byte("hello, " + claims.Subject))
			})
		},
		authMW, rlMW,
	)

	oh := pkgserver.NewOtherHTTPServer(
		pkgserver.HTTPConfig{Network: "tcp", Addr: otherAddr},
		logger,
		nil,
	)
	oh.S.HandleFunc("/version", version.Handler(version.Info{
		Name: Name, Version: Version, Commit: Commit, BuildTime: BuildTime,
	}))

	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Logger(logger),
		kratos.Server(hs.S, oh.S),
	)
	if err := app.Run(); err != nil {
		log.NewHelper(logger).Fatalf("app run: %v", err)
	}
}
