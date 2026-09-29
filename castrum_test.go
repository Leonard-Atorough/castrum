package castrum

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
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

func TestWithBindingsWiresInput(t *testing.T) {
	bindings := input.Bindings{
		"jump":   []input.Input{input.KeyInput{Key: input.KeySpace}},
		"move_x": []input.Input{input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}},
	}
	g, err := New(WithBindings(bindings))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}

	s := &input.Snapshot{}
	s.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	s.Keys[input.KeyD] = input.State{Held: true}
	g.Context().Input = s

	tick := time.Second / 60
	if err := g.Advance(tick); err != nil {
		t.Fatalf("Advance: %v", err)
	}

	am, err := g.World().Resource[*input.ActionMap]()
	if err != nil {
		t.Fatalf("resolve action map: %v", err)
	}
	if !am.Pressed("jump") {
		t.Error("jump should be latched for the tick that consumed the press")
	}
	if !am.JustPressed("jump") || !am.Held("jump") {
		t.Error("jump should be pressed and held in the frame view")
	}
	if am.Axis("move_x") != 1 {
		t.Errorf("move_x axis = %v, want 1", am.Axis("move_x"))
	}

	// The context publishes the same map: the discoverable face and
	// the resource are one instance.
	if g.Context().Actions != am {
		t.Error("Context.Actions should be the same map the resource resolves")
	}
	if !g.Context().Actions.Pressed("jump") {
		t.Error("Context.Actions should read the same tick view")
	}

	// The next frame with no input consumes the edge: the tick view
	// clears, and duration resets.
	g.Context().Input = &input.Snapshot{}
	if err := g.Advance(tick); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if am.Pressed("jump") || am.Held("jump") || am.Duration("jump") != 0 {
		t.Error("the second tick should see no jump input")
	}
}

func TestNoBindingsNoActionMap(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := g.World().Resource[*input.ActionMap](); err == nil {
		t.Error("an ActionMap resource should not exist without WithBindings")
	}
	if g.Context().Actions != nil {
		t.Error("Context.Actions should be nil without WithBindings")
	}
	if g.Context().Actions.Pressed("jump") || g.Context().Actions.Held("jump") ||
		g.Context().Actions.Axis("move_x") != 0 {
		t.Error("queries on the nil Actions should read zero")
	}
}
