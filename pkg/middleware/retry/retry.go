// Package retry provides an opt-in client middleware that retries failed
// outbound calls with exponential backoff + jitter.
//
// **Idempotency warning.** Retries are only safe for idempotent operations
// (GET, PUT, DELETE — never POST without a domain-specific dedup key).
// This middleware does not enforce idempotency; the caller takes that risk
// when they enable it.
//
// Default RetryOn covers:
//   - net errors (DNS, connection refused, reset)
//   - kratos GatewayTimeout (504) / ServiceUnavailable (503) / RATE_LIMITED (429)
//   - gRPC UNAVAILABLE / DEADLINE_EXCEEDED (when carried as kratos errors)
//
// Backoff schedule:
//   - attempt 1: initial backoff = Backoff (default 100ms)
//   - attempt 2: 2× initial
//   - attempt N: min(initial × 2^(N-1), MaxBackoff)
//   - each step adds ±25% jitter so retrying clients don't thunder
package retry

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Default backoff parameters. Override via Option.
const (
	DefaultMaxAttempts    = 3
	DefaultInitialBackoff = 100 * time.Millisecond
	DefaultMaxBackoff     = 2 * time.Second
)

// Predicate decides whether `err` is worth retrying. Returns true to retry.
type Predicate func(err error) bool

type options struct {
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	retryOn        Predicate
}

// Option is a functional config setter.
type Option func(*options)

// WithMaxAttempts caps the total number of attempts (including the first).
// MaxAttempts <= 1 disables the middleware (it becomes a pass-through).
func WithMaxAttempts(n int) Option { return func(o *options) { o.maxAttempts = n } }

// WithInitialBackoff sets the first-retry sleep.
func WithInitialBackoff(d time.Duration) Option { return func(o *options) { o.initialBackoff = d } }

// WithMaxBackoff caps each backoff step.
func WithMaxBackoff(d time.Duration) Option { return func(o *options) { o.maxBackoff = d } }

// WithRetryOn overrides the default retryable-error predicate.
func WithRetryOn(p Predicate) Option { return func(o *options) { o.retryOn = p } }

// DefaultRetryOn returns the built-in retry predicate.
func DefaultRetryOn(err error) bool {
	if err == nil {
		return false
	}
	// Context cancellation / deadline propagated from caller: do NOT retry
	// (the caller asked us to stop).
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// Net-layer transient: dial failure, conn reset, timeout.
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	// Kratos-encoded server signals: 503 / 504 / 429.
	var ke *kerrors.Error
	if errors.As(err, &ke) {
		switch ke.Code {
		case 503, 504, 429:
			return true
		}
	}
	return false
}

// Client returns the retry middleware. Plug into the outbound chain BEFORE
// circuitbreaker so each retry counts against the breaker; or AFTER if you
// want the breaker to absorb retry storms.
func Client(opts ...Option) middleware.Middleware {
	o := &options{
		maxAttempts:    DefaultMaxAttempts,
		initialBackoff: DefaultInitialBackoff,
		maxBackoff:     DefaultMaxBackoff,
		retryOn:        DefaultRetryOn,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if o.maxAttempts <= 1 {
				return handler(ctx, req)
			}
			var lastErr error
			for attempt := 1; attempt <= o.maxAttempts; attempt++ {
				if err := ctx.Err(); err != nil {
					return nil, err // caller bailed
				}
				reply, err := handler(ctx, req)
				if err == nil {
					return reply, nil
				}
				lastErr = err
				if !o.retryOn(err) {
					return nil, err
				}
				if attempt == o.maxAttempts {
					break
				}
				if err := sleepWithCtx(ctx, backoffFor(attempt, o.initialBackoff, o.maxBackoff)); err != nil {
					return nil, err
				}
			}
			return nil, lastErr
		}
	}
}

// backoffFor returns the sleep before the (attempt+1)-th call, given that
// `attempt` (1-indexed) just failed. Exponential with ±25% jitter, clamped.
func backoffFor(attempt int, initial, maxBackoff time.Duration) time.Duration {
	d := initial << (attempt - 1) // initial * 2^(attempt-1)
	if d <= 0 || d > maxBackoff {
		d = maxBackoff
	}
	// ±25% jitter
	jitter := time.Duration(rand.Int63n(int64(d) / 2)) //nolint:gosec // non-cryptographic
	d = d - d/4 + jitter
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

// sleepWithCtx returns nil on a clean timer fire, or ctx.Err() on cancel.
func sleepWithCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
