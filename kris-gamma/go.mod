module github.com/kris/go-infrastructure/kris-gamma

go 1.25.0

require (
	github.com/go-kratos/kratos/v2 v2.8.0
	github.com/kris/go-infrastructure/pkg v0.0.0-00010101000000-000000000000
	go.uber.org/automaxprocs v1.6.0
	google.golang.org/grpc v1.80.0
)

replace github.com/kris/go-infrastructure/pkg => ../pkg
