package server

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

// ErrDraining is returned by the drain Pinger once shutdown has begun.
var ErrDraining = errors.New("draining")

// Drainer coordinates a graceful-shutdown grace period:
//
//  1. On SIGTERM, kratos invokes BeforeStop hooks (registered via
//     `kratos.BeforeStop(drainer.BeforeStop)`).
//  2. BeforeStop flips an internal `draining` flag — wired into /readyz via
//     Drainer.ReadinessPinger(), the probe now fails so the load balancer
//     (kube-proxy, ingress, etc.) drops the pod from the rotation.
//  3. BeforeStop sleeps for `delay` (typical: 10s) before returning, so
//     in-flight requests can finish and the LB has time to de-register the
//     pod **before** the actual servers stop.
//  4. kratos proceeds with server shutdown.
//
// Without this delay, k8s rolling restarts drop new requests on the
// terminating pod between SIGTERM and the time kube-proxy sync removes
// it from endpoints — a common source of "random 502s during deploy".
//
// Usage:
//
//	drainer := server.NewDrainer(10 * time.Second)
//	probes := &server.ReadinessProbes{
//	    All: []server.Pinger{
//	        drainer.ReadinessPinger(),
//	        // ... your downstream probes
//	    },
//	}
//	oh := server.NewOtherHTTPServer(otherCfg, logger, probes)
//
//	app := kratos.New(
//	    kratos.Server(gs, hs.S, oh.S),
//	    kratos.BeforeStop(drainer.BeforeStop),
//	)
type Drainer struct {
	delay    time.Duration
	draining atomic.Bool
}

// NewDrainer returns a Drainer with the given grace period. Pass <=0 to
// skip the sleep (still flips the readiness flag for tests / immediate-stop).
func NewDrainer(delay time.Duration) *Drainer {
	return &Drainer{delay: delay}
}

// Draining reports whether the process has begun shutting down.
func (d *Drainer) Draining() bool { return d.draining.Load() }

// BeforeStop is the kratos `BeforeStop` hook. Flips draining and sleeps for
// the configured delay (or until ctx is cancelled).
func (d *Drainer) BeforeStop(ctx context.Context) error {
	d.draining.Store(true)
	if d.delay <= 0 {
		return nil
	}
	timer := time.NewTimer(d.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
	return nil
}

// ReadinessPinger returns a probe that fails (ErrDraining) once Draining()
// is true. Append to your ReadinessProbes.All slice.
func (d *Drainer) ReadinessPinger() Pinger {
	return Pinger{
		Name: "drain",
		Ping: func(context.Context) error {
			if d.draining.Load() {
				return ErrDraining
			}
			return nil
		},
	}
}
