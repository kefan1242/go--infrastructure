package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgserver "github.com/kris/go-infrastructure/pkg/runtime/server"
)

func TestDrainer_DrainingStartsFalse(t *testing.T) {
	d := pkgserver.NewDrainer(time.Second)
	if d.Draining() {
		t.Fatal("brand-new drainer should not be draining")
	}
}

func TestDrainer_ReadinessPingerFailsOnceDraining(t *testing.T) {
	d := pkgserver.NewDrainer(0)
	pinger := d.ReadinessPinger()

	if err := pinger.Ping(context.Background()); err != nil {
		t.Errorf("pre-drain ping should succeed, got %v", err)
	}
	// Trigger drain without sleeping (delay=0).
	_ = d.BeforeStop(context.Background())

	if !d.Draining() {
		t.Error("Draining() should be true after BeforeStop")
	}
	if err := pinger.Ping(context.Background()); !errors.Is(err, pkgserver.ErrDraining) {
		t.Errorf("post-drain ping: want ErrDraining, got %v", err)
	}
}

func TestDrainer_BeforeStopWaitsForDelay(t *testing.T) {
	d := pkgserver.NewDrainer(150 * time.Millisecond)
	start := time.Now()
	if err := d.BeforeStop(context.Background()); err != nil {
		t.Fatalf("BeforeStop: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Errorf("BeforeStop returned too quickly (%v); expected >= 150ms delay", elapsed)
	}
}

func TestDrainer_BeforeStopHonorsContextCancel(t *testing.T) {
	d := pkgserver.NewDrainer(5 * time.Second) // intentionally long
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- d.BeforeStop(ctx) }()

	// Cancel quickly — BeforeStop should return promptly.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("BeforeStop should return nil on ctx cancel, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("BeforeStop didn't respect ctx cancel within 1s")
	}
}

func TestDrainer_ZeroDelaySkipsSleep(t *testing.T) {
	d := pkgserver.NewDrainer(0)
	start := time.Now()
	_ = d.BeforeStop(context.Background())
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("zero-delay BeforeStop took %v; should be near-instant", elapsed)
	}
}
