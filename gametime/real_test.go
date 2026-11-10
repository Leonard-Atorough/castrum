package gametime

import (
	"testing"
	"time"
)

func TestNewRealDefaults(t *testing.T) {
	r := NewReal(250 * time.Millisecond)
	if r.Elapsed != 0 || r.Delta != 0 {
		t.Fatalf("real = %+v, want zero elapsed and delta", *r)
	}
	if r.MaxFrameTime != 250*time.Millisecond {
		t.Fatalf("MaxFrameTime = %v, want 250ms", r.MaxFrameTime)
	}
}

func TestRealSnapshotRestore(t *testing.T) {
	r := NewReal(time.Second)
	r.Elapsed = 42 * time.Second
	r.Delta = time.Second

	restored := &Real{Elapsed: r.Elapsed, Delta: r.Delta, MaxFrameTime: r.MaxFrameTime}
	if restored.Elapsed != 42*time.Second {
		t.Fatalf("restored Elapsed = %v, want 42s", restored.Elapsed)
	}
}
