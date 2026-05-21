// Package ratelimit provides an inbound token-bucket rate limiter.
//
// Algorithm: x/time/rate TokenBucket, sharded by key. The default key is the
// gRPC / HTTP client IP; override via WithKey for user id / api key / etc.
//
// Usage:
//
//	mw := ratelimit.Server(
//	    ratelimit.WithRate(100, 200),
//	    ratelimit.WithSkip(func(op string) bool { ... }),
//	)
//
// Plug into pkg/runtime/server's `extra` middlewares.
//
// Note: implementation is **in-process**, so each replica enforces its own
// quota. For cluster-wide limits, swap to a Redis Lua impl or front the
// services with an Envoy rate-limit service.
package ratelimit

import (
	"context"
	"net"
	"sync"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"golang.org/x/time/rate"
)

// KeyFn maps a request to a bucket key (requests sharing a key share a limiter).
type KeyFn func(ctx context.Context, op string) string

// SkipFn returns true to bypass rate limiting for a given operation.
type SkipFn func(op string) bool

type options struct {
	rps    rate.Limit
	burst  int
	keyFn  KeyFn
	skipFn SkipFn
}

// Option is a functional config setter.
type Option func(*options)

// WithRate sets the steady-state rate (req/s) and burst size.
func WithRate(rps, burst int) Option {
	return func(o *options) { o.rps = rate.Limit(rps); o.burst = burst }
}

// WithKey overrides the default IP-based bucket key extractor.
func WithKey(fn KeyFn) Option { return func(o *options) { o.keyFn = fn } }

// WithSkip skips rate limiting for matching operations (health, metrics).
func WithSkip(fn SkipFn) Option { return func(o *options) { o.skipFn = fn } }

// ErrTooManyRequests is returned when the bucket is empty.
var ErrTooManyRequests = kerrors.New(429, "RATE_LIMITED", "too many requests")

// Server is the inbound rate-limit middleware.
func Server(opts ...Option) middleware.Middleware {
	o := &options{rps: 100, burst: 200}
	for _, opt := range opts {
		opt(o)
	}
	if o.keyFn == nil {
		o.keyFn = defaultKey
	}
	b := newBucket(o.rps, o.burst)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			op := ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				op = tr.Operation()
			}
			if o.skipFn != nil && o.skipFn(op) {
				return handler(ctx, req)
			}
			key := o.keyFn(ctx, op)
			if !b.allow(key) {
				return nil, ErrTooManyRequests
			}
			return handler(ctx, req)
		}
	}
}

// defaultKey extracts the client IP, falling back to "global".
func defaultKey(ctx context.Context, _ string) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if xff := tr.RequestHeader().Get("x-forwarded-for"); xff != "" {
			if i := indexComma(xff); i > 0 {
				return trim(xff[:i])
			}
			return trim(xff)
		}
		if r := tr.RequestHeader().Get("x-real-ip"); r != "" {
			return r
		}
	}
	if p, ok := remotePeer(ctx); ok {
		if host, _, err := net.SplitHostPort(p); err == nil {
			return host
		}
		return p
	}
	return "global"
}

// bucket lazy-creates a limiter per key.
type bucket struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newBucket(rps rate.Limit, burst int) *bucket {
	return &bucket{limiters: map[string]*rate.Limiter{}, rps: rps, burst: burst}
}

func (b *bucket) allow(key string) bool {
	b.mu.Lock()
	l, ok := b.limiters[key]
	if !ok {
		l = rate.NewLimiter(b.rps, b.burst)
		b.limiters[key] = l
	}
	b.mu.Unlock()
	return l.Allow()
}

func indexComma(s string) int {
	for i, c := range s {
		if c == ',' {
			return i
		}
	}
	return -1
}
func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
