package ebitrun

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
)

// The option is explicit and off by default: the runner registers no
// draw callback without it and one with it.
func TestDebugOverlayOption(t *testing.T) {
	plain, err := castrum.New()
	if err != nil {
		t.Fatal(err)
	}
	runner, err := New(plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.draws) != 0 {
		t.Errorf("draws = %d, want none by default", len(runner.draws))
	}

	g, err := castrum.New()
	if err != nil {
		t.Fatal(err)
	}
	runner, err = New(g, WithDebugOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.draws) != 1 {
		t.Errorf("draws = %d, want the overlay's callback", len(runner.draws))
	}
}

// Rates stay zero until a full sample window has passed.
func TestDebugOverlayWaitsForWindow(t *testing.T) {
	o := newDebugOverlay()
	start := time.Now()
	for range 10 {
		o.sample(start.Add(10*time.Millisecond), 1)
	}
	if o.fps != 0 || o.tps != 0 {
		t.Errorf("fps %v tps %v, want zero before the window elapses", o.fps, o.tps)
	}
}

// The window measures display frames and engine ticks per second. A
// frame every 60th of a second reads 60 fps; a tick per frame reads
// 60 tps. The next window starts fresh, so slowing ticks to 20 per
// second reads 20 while the frame rate holds.
func TestDebugOverlayMeasuresRates(t *testing.T) {
	o := newDebugOverlay()
	start := time.Now()
	for i := range 31 {
		o.sample(start.Add(time.Duration(i)*time.Second/60), uint64(i))
	}
	if !almost(o.fps, 60, 0.1) {
		t.Errorf("fps = %v, want 60", o.fps)
	}
	if !almost(o.tps, 60, 0.1) {
		t.Errorf("tps = %v, want 60", o.tps)
	}

	for i := range 30 {
		o.sample(start.Add(time.Second/2+time.Duration(i+1)*time.Second/60), uint64(30+(i+1)/3))
	}
	if !almost(o.fps, 60, 0.1) {
		t.Errorf("fps = %v, want 60 in the second window", o.fps)
	}
	if !almost(o.tps, 20, 0.1) {
		t.Errorf("tps = %v, want 20 in the second window", o.tps)
	}
}

func almost(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}
