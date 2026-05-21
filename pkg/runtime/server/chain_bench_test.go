package server

import (
	"context"
	"testing"

	"github.com/kris/go-infrastructure/pkg/testutil"

	"github.com/go-kratos/kratos/v2/middleware"
)

// BenchmarkDefaultChain_PassesThroughNoopHandler measures the per-request
// overhead of recovery -> tracing -> logid -> access -> metric around a
// handler that returns nil immediately. Baseline lets reviewers spot
// regressions when adding new middleware to the default chain.
//
// Run: go test -run=^$ -bench=BenchmarkDefaultChain -benchmem ./pkg/runtime/server/...
func BenchmarkDefaultChain_PassesThroughNoopHandler(b *testing.B) {
	logger, _ := testutil.NewMemoryLogger()
	mws := defaultChain(logger)
	chain := middleware.Chain(mws...)

	ft := testutil.NewFakeTransport()
	ctx := testutil.InjectServerContext(context.Background(), ft)

	noop := middleware.Handler(func(context.Context, any) (any, error) {
		return nil, nil
	})
	handler := chain(noop)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler(ctx, nil)
	}
}
