package castrum

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
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

// MainCamera: New spawns the default camera - primary, zoom 1 at
// the origin - and the accessor hands it out for follow systems.
// The preference rule (user primaries win) is proven in core's
// TestEngineCameraPreference.
func TestMainCamera(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	camera := g.MainCamera()
	if camera == nil {
		t.Fatal("MainCamera should return the engine camera")
	}
	cam, ok := camera.Component[core.Camera](g.World())
	if !ok || !cam.Primary || cam.Zoom != 1 {
		t.Fatalf("engine camera = %+v, ok %v, want primary zoom 1", cam, ok)
	}
	transform, _ := camera.Component[core.Transform](g.World())
	if transform.Position != (geom.Vector2{}) {
		t.Fatalf("engine camera position = %v, want the origin", transform.Position)
	}
}

// The asset server is provided by New, before any runner exists, so
// games register atlases at setup time.
func TestNewProvidesAssetServer(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if g.AssetServer() == nil {
		t.Fatal("AssetServer should return the engine-provided server")
	}
	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("Resource[*asset.Server] right after New: %v", err)
	}
	if g.AssetServer() != server {
		t.Fatal("AssetServer and the resource must be the same instance")
	}
}

// The clip store is provided by New, before any runner exists, so
// games Add clips at setup time.
func TestNewProvidesClipStore(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if g.Clips() == nil {
		t.Fatal("Clips should return the engine-provided store")
	}
	store, err := g.World().Resource[*animation.ClipStore]()
	if err != nil {
		t.Fatalf("Resource[*animation.ClipStore] right after New: %v", err)
	}
	if g.Clips() != store {
		t.Fatal("Clips and the resource must be the same instance")
	}
}

func TestNewProvidesAudioMixer(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if g.Mixer() == nil {
		t.Fatal("Mixer should return the engine-provided mixer")
	}
	mixer, err := g.World().Resource[*audio.Mixer]()
	if err != nil {
		t.Fatalf("Resource[*audio.Mixer] right after New: %v", err)
	}
	if g.Mixer() != mixer {
		t.Fatal("Mixer and the resource must be the same instance")
	}
}
