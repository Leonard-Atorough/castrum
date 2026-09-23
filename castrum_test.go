package castrum

import (
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	opts := g.Options()
	if opts.Title != "castrum" {
		t.Errorf("Title = %q, want %q", opts.Title, "castrum")
	}
	if opts.FixedTPS != 60 {
		t.Errorf("FixedTPS = %d, want 60", opts.FixedTPS)
	}
	if opts.MaxFrameTime != 250*time.Millisecond {
		t.Errorf("MaxFrameTime = %v, want 250ms", opts.MaxFrameTime)
	}
	if opts.MaxTicksPerFrame != 5 {
		t.Errorf("MaxTicksPerFrame = %d, want 5", opts.MaxTicksPerFrame)
	}
	if opts.FixedDT() != time.Second/60 {
		t.Errorf("FixedDT = %v, want %v", opts.FixedDT(), time.Second/60)
	}
}

func TestOptionOverrides(t *testing.T) {
	g, err := New(
		WithTitle("Demo"),
		WithFixedTPS(120),
		WithMaxFrameTime(time.Second),
		WithMaxTicksPerFrame(2),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	opts := g.Options()

	if opts.Title != "Demo" {
		t.Errorf("Title = %q, want %q", opts.Title, "Demo")
	}
	if opts.FixedTPS != 120 {
		t.Errorf("FixedTPS = %d, want 120", opts.FixedTPS)
	}
	if opts.MaxFrameTime != time.Second {
		t.Errorf("MaxFrameTime = %v, want 1s", opts.MaxFrameTime)
	}
	if opts.MaxTicksPerFrame != 2 {
		t.Errorf("MaxTicksPerFrame = %d, want 2", opts.MaxTicksPerFrame)
	}
	if opts.FixedDT() != time.Second/120 {
		t.Errorf("FixedDT = %v, want %v", opts.FixedDT(), time.Second/120)
	}
}

func TestInvalidOptionsError(t *testing.T) {
	cases := []struct {
		name string
		opts []option
	}{
		{"zero tps", []option{WithFixedTPS(0)}},
		{"negative tps", []option{WithFixedTPS(-1)}},
		{"zero max frame time", []option{WithMaxFrameTime(0)}},
		{"zero max ticks", []option{WithMaxTicksPerFrame(0)}},
		{"unrepresentable tick interval", []option{WithFixedTPS(2_000_000_000)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g, err := New(c.opts...)
			if err == nil {
				t.Errorf("%s: expected an error", c.name)
			}
			if g != nil {
				t.Errorf("%s: New must not return a half-built game", c.name)
			}
		})
	}
}
