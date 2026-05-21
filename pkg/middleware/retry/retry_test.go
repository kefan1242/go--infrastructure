package retry_test

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/middleware/retry"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

func TestClient_FirstSuccessNoRetry(t *testing.T) {
	var calls atomic.Int32
	mw := retry.Client(retry.WithMaxAttempts(3))
	handler := mw(func(context.Context, any) (any, error) {
		calls.Add(1)
		return "ok", nil
	})
	reply, err := handler(context.Background(), nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply: want ok, got %v", reply)
	}
	if calls.Load() != 1 {
		t.Errorf("calls: want 1 (no retry on success), got %d", calls.Load())
	}
}

func TestClient_RetriableErrorEventuallySucceeds(t *testing.T) {
	var calls atomic.Int32
	mw := retry.Client(
		retry.WithMaxAttempts(5),
		retry.WithInitialBackoff(time.Millisecond), // fast for test
		retry.WithMaxBackoff(5*time.Millisecond),
	)
	handler := mw(func(context.Context, any) (any, error) {
		if calls.Add(1) < 3 {
			return nil, kerrors.New(503, "UNAVAILABLE", "try again")
		}
		return "ok", nil
	})

	start := time.Now()
	reply, err := handler(context.Background(), nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if reply != "ok" {
		t.Errorf("reply: want ok, got %v", reply)
	}
	if calls.Load() != 3 {
		t.Errorf("calls: want 3 (1 fail + 1 fail + 1 ok), got %d", calls.Load())
	}
	if elapsed < 2*time.Millisecond {
		t.Errorf("elapsed: backoff should have slept, only %v passed", elapsed)
	}
}

func TestClient_NonRetriableReturnsImmediately(t *testing.T) {
	var calls atomic.Int32
	mw := retry.Client(retry.WithMaxAttempts(5), retry.WithInitialBackoff(time.Millisecond))
	want := kerrors.BadRequest("BAD_REQUEST", "client error")
	handler := mw(func(context.Context, any) (any, error) {
		calls.Add(1)
		return nil, want
	})

	_, err := handler(context.Background(), nil)
	if !errors.Is(err, want) {
		t.Errorf("err: want %v, got %v", want, err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls: want 1 (non-retriable shouldn't retry), got %d", calls.Load())
	}
}

func TestClient_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	mw := retry.Client(
		retry.WithMaxAttempts(3),
		retry.WithInitialBackoff(time.Millisecond),
		retry.WithMaxBackoff(5*time.Millisecond),
	)
	want := kerrors.New(503, "UNAVAILABLE", "down")
	handler := mw(func(context.Context, any) (any, error) {
		calls.Add(1)
		return nil, want
	})
	_, err := handler(context.Background(), nil)
	if !errors.Is(err, want) {
		t.Errorf("err: want last error %v, got %v", want, err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls: want 3, got %d", calls.Load())
	}
}

func TestClient_MaxAttemptsOneIsPassThrough(t *testing.T) {
	var calls atomic.Int32
	mw := retry.Client(retry.WithMaxAttempts(1))
	handler := mw(func(context.Context, any) (any, error) {
		calls.Add(1)
		return nil, kerrors.New(503, "UNAVAILABLE", "x")
	})
	_, _ = handler(context.Background(), nil)
	if calls.Load() != 1 {
		t.Errorf("max=1 should not retry, got %d calls", calls.Load())
	}
}

func TestClient_ContextCancelAbortsRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mw := retry.Client(
		retry.WithMaxAttempts(10),
		retry.WithInitialBackoff(200*time.Millisecond),
	)
	handler := mw(func(context.Context, any) (any, error) {
		return nil, kerrors.New(503, "UNAVAILABLE", "x")
	})

	// Cancel after first failure but before second attempt's backoff finishes.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := handler(ctx, nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err: want context.Canceled, got %v", err)
	}
	if elapsed > 150*time.Millisecond {
		t.Errorf("ctx-cancel should abort retry quickly, took %v", elapsed)
	}
}

func TestDefaultRetryOn_NilNotRetried(t *testing.T) {
	if retry.DefaultRetryOn(nil) {
		t.Error("nil error should never retry")
	}
}

func TestDefaultRetryOn_CtxCanceledNotRetried(t *testing.T) {
	if retry.DefaultRetryOn(context.Canceled) {
		t.Error("ctx canceled should not retry")
	}
	if retry.DefaultRetryOn(context.DeadlineExceeded) {
		t.Error("deadline exceeded should not retry — caller's budget is up")
	}
}

func TestDefaultRetryOn_KratosTransients(t *testing.T) {
	for _, code := range []int{503, 504, 429} {
		err := kerrors.New(code, "X", "x")
		if !retry.DefaultRetryOn(err) {
			t.Errorf("code %d should retry", code)
		}
	}
	if retry.DefaultRetryOn(kerrors.New(400, "BAD", "x")) {
		t.Error("400 should NOT retry")
	}
}

func TestDefaultRetryOn_NetOpError(t *testing.T) {
	// Synthesize a typical net.OpError (the kind net/http returns on dial fail).
	err := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("conn refused")}
	if !retry.DefaultRetryOn(err) {
		t.Error("net.OpError should retry")
	}
}

func TestWithRetryOn_CustomPredicateUsed(t *testing.T) {
	called := false
	mw := retry.Client(
		retry.WithMaxAttempts(2),
		retry.WithInitialBackoff(time.Microsecond),
		retry.WithRetryOn(func(error) bool { called = true; return false }),
	)
	_, _ = mw(func(context.Context, any) (any, error) {
		return nil, errors.New("anything")
	})(context.Background(), nil)
	if !called {
		t.Error("custom predicate should have been consulted")
	}
}
