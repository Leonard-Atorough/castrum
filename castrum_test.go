package castrum

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/timer"
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
		"jump":   {input.KeyInput{Key: input.KeySpace}},
		"move_x": {input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}},
		"dash": {input.Chord(
			input.KeyInput{Key: input.KeyB},
			input.KeyInput{Key: input.KeyC},
		)},
		"throttle": {input.PadAxisInput{Axis: input.PadRightTrigger, Deadzone: 0.05}},
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
	s.Keys[input.KeyB].Held, s.Keys[input.KeyC].Held = true, true
	s.Pads[0].Connected = true
	s.Pads[0].Axes[input.PadRightTrigger] = 0.1
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
	if !am.Pressed("dash") || !am.Held("dash") || !am.Pressed("throttle") || am.Axis("throttle") != 0.1 {
		t.Error("chord and per-binding axis should reach frame and tick action views")
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
	if !am.Released("dash") || !am.Released("throttle") || am.Axis("throttle") != 0 {
		t.Error("the second tick should deliver chord and axis releases")
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

func TestMainCamera(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	camera := g.MainCamera()
	if camera == nil {
		t.Fatal("MainCamera should return the engine camera")
	}
	cam, ok := camera.Component[render.Camera](g.World())
	if !ok || !cam.Primary {
		t.Fatalf("engine camera = %+v, ok %v, want primary", cam, ok)
	}
	transform, _ := camera.Component[core.Transform](g.World())
	if transform.Position != (geom.Vector2{}) {
		t.Fatalf("engine camera position = %v, want the origin", transform.Position)
	}
	if transform.Scale.X != 1 || transform.Scale.Y != 1 {
		t.Fatalf("engine camera scale = %v, want zoom 1", transform.Scale)
	}
}

func TestNewProvidesAssetLoader(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	loader := g.World().MustResource[*asset.Loader]()
	if loader == nil {
		t.Fatal("MustResource should return the engine-provided loader")
	}
	byErr, err := g.World().Resource[*asset.Loader]()
	if err != nil {
		t.Fatalf("Resource[*asset.Loader] right after New: %v", err)
	}
	if loader != byErr {
		t.Fatal("MustResource and Resource must be the same instance")
	}
}

func TestNewProvidesClipStore(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	store := g.World().MustResource[*animation.ClipStore]()
	if store == nil {
		t.Fatal("MustResource should return the engine-provided store")
	}
	byErr, err := g.World().Resource[*animation.ClipStore]()
	if err != nil {
		t.Fatalf("Resource[*animation.ClipStore] right after New: %v", err)
	}
	if store != byErr {
		t.Fatal("MustResource and Resource must be the same instance")
	}
}

func TestNewProvidesAudioMixer(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mixer := g.World().MustResource[*audio.Mixer]()
	if mixer == nil {
		t.Fatal("MustResource should return the engine-provided mixer")
	}
	byErr, err := g.World().Resource[*audio.Mixer]()
	if err != nil {
		t.Fatalf("Resource[*audio.Mixer] right after New: %v", err)
	}
	if mixer != byErr {
		t.Fatal("MustResource and Resource must be the same instance")
	}
}

func TestNewProvidesSignalHub(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	hub := g.World().MustResource[*signal.Hub]()
	if hub == nil {
		t.Fatal("MustResource should return the engine-provided hub")
	}
	byErr, err := g.World().Resource[*signal.Hub]()
	if err != nil {
		t.Fatalf("Resource[*signal.Hub] right after New: %v", err)
	}
	if hub != byErr {
		t.Fatal("MustResource and Resource must be the same instance")
	}
}

func TestOptionalTimerSystems(t *testing.T) {
	spawn := func(g *Game) {
		if _, err := g.World().NewEntity(timer.NewTimer(2*g.Options().FixedDT(), false)); err != nil {
			t.Fatalf("spawn timer: %v", err)
		}
	}
	advanceTwoTicks := func(g *Game) {
		if err := g.Advance(2 * g.Options().FixedDT()); err != nil {
			t.Fatalf("Advance: %v", err)
		}
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spawn(g)
	advanceTwoTicks(g)
	for e := range core.NewQuery(g.World()).With(timer.Timer{}).Execute() {
		if tm, _ := e.Component[timer.Timer](); tm.CompletedOn != 0 {
			t.Errorf("timer completed without WithTimer: %+v", tm)
		}
	}

	g, err = New(WithTimer())
	if err != nil {
		t.Fatalf("New(WithTimer): %v", err)
	}
	spawn(g)
	advanceTwoTicks(g)
	for e := range core.NewQuery(g.World()).With(timer.Timer{}).Execute() {
		if tm, _ := e.Component[timer.Timer](); tm.CompletedOn == 0 {
			t.Errorf("timer did not complete with WithTimer: %+v", tm)
		}
	}
}

func TestOptionalCollisionSystem(t *testing.T) {
	spawnPair := func(g *Game) {
		for _, x := range []float64{0, 1} {
			collider, err := collision.NewCollider(collision.Circle{Radius: 8})
			if err != nil {
				t.Fatalf("NewCollider: %v", err)
			}
			if _, err := g.World().NewEntity(collider, core.Transform{Position: geom.Vector2{X: x, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
				t.Fatalf("spawn collider: %v", err)
			}
		}
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spawnPair(g)
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(collision.Collider{}).Execute() {
		if _, ok := core.NewEntity(e.ID()).Component[collision.Contacts](g.World()); ok {
			t.Error("contacts produced without WithCollision")
		}
	}

	g, err = New(WithCollision())
	if err != nil {
		t.Fatalf("New(WithCollision): %v", err)
	}
	spawnPair(g)
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(collision.Collider{}).Execute() {
		contacts, ok := core.NewEntity(e.ID()).Component[collision.Contacts](g.World())
		if !ok || len(contacts.Current) != 1 {
			t.Errorf("contacts with WithCollision = %+v, ok %v, want one contact", contacts, ok)
		}
	}
}

func TestOptionalAnimationSystem(t *testing.T) {
	spawnAnimated := func(g *Game) {
		if err := g.World().MustResource[*animation.ClipStore]().Add("clip", animation.Clip{
			Source: "sprites",
			Frames: []string{"frame_0", "frame_1"},
			FPS:    60,
		}); err != nil {
			t.Fatalf("Add clip: %v", err)
		}
		if _, err := g.World().NewEntity(animation.Animation{Clip: "clip"}, render.Sprite{}, core.Transform{Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
			t.Fatalf("spawn animated sprite: %v", err)
		}
	}

	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spawnAnimated(g)
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(animation.Animation{}).Execute() {
		if s, _ := core.NewEntity(e.ID()).Component[render.Sprite](g.World()); s.Drawable != nil {
			t.Errorf("sprite drawable written without WithAnimation: %+v", s.Drawable)
		}
	}

	g, err = New(WithAnimation())
	if err != nil {
		t.Fatalf("New(WithAnimation): %v", err)
	}
	spawnAnimated(g)
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(animation.Animation{}).Execute() {
		s, _ := core.NewEntity(e.ID()).Component[render.Sprite](g.World())
		if _, ok := s.Drawable.(render.AtlasSource); !ok {
			t.Errorf("sprite drawable with WithAnimation = %#v, want an atlas source", s.Drawable)
		}
	}
}

func TestWithDefaultSystemsIdempotent(t *testing.T) {
	g, err := New(WithDefaultSystems(), WithTimer())
	if err != nil {
		t.Fatalf("New(WithDefaultSystems, WithTimer): %v", err)
	}
	if _, err := g.World().NewEntity(timer.NewTimer(2*g.Options().FixedDT(), false)); err != nil {
		t.Fatalf("spawn timer: %v", err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(timer.Timer{}).Execute() {
		if tm, _ := e.Component[timer.Timer](); tm.CompletedOn != 0 {
			t.Fatalf("timer completed after one tick: %+v, want a single registration", tm)
		}
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	for e := range core.NewQuery(g.World()).With(timer.Timer{}).Execute() {
		if tm, _ := e.Component[timer.Timer](); tm.CompletedOn == 0 {
			t.Fatalf("timer did not complete on its second tick: %+v", tm)
		}
	}
}

func TestWithTimerProvidesGroups(t *testing.T) {
	g, err := New(WithTimer())
	if err != nil {
		t.Fatalf("New(WithTimer): %v", err)
	}
	if _, err := g.World().Resource[*timer.Groups](); err != nil {
		t.Errorf("WithTimer should provide timer groups: %v", err)
	}
}

func TestSetGroupEnabledAcrossPhasesAndRegistrations(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var fixed, frame, late, other, ungrouped int
	if err := g.AddSystemToGroup(core.PhaseFixed, "mission", "first fixed", core.SystemFunc(func(*core.Context) error {
		fixed++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFixed, "other", "other fixed", core.SystemFunc(func(*core.Context) error {
		other++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "ungrouped fixed", core.SystemFunc(func(*core.Context) error {
		ungrouped++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.SetGroupEnabled("mission", false); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFrame, "mission", "late frame", core.SystemFunc(func(*core.Context) error {
		frame++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFixed, "mission", "late fixed", core.SystemFunc(func(*core.Context) error {
		late++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance inactive: %v", err)
	}
	if fixed != 0 || frame != 0 || late != 0 {
		t.Fatalf("inactive updates: fixed=%d frame=%d late=%d", fixed, frame, late)
	}
	if other != 1 || ungrouped != 1 {
		t.Fatalf("unrelated updates: other=%d ungrouped=%d, want 1 each", other, ungrouped)
	}
	if err := g.SetGroupEnabled("mission", true); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if fixed != 1 || frame != 1 || late != 1 {
		t.Fatalf("active updates: fixed=%d frame=%d late=%d, want 1 each", fixed, frame, late)
	}
}

func TestSetGroupEnabledRejectsUnknownGroups(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "missing"} {
		if err := g.SetGroupEnabled(name, false); err == nil {
			t.Errorf("SetGroupEnabled(%q) accepted an invalid group", name)
		}
	}
	var updates int
	if err := g.AddSystemToGroup(core.PhaseFixed, "missing", "later", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if updates != 1 {
		t.Fatalf("rejected group change affected later registration: updates=%d", updates)
	}
}

func TestSetGroupEnabledAffectsLaterSystemsInCurrentPhase(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	var updates int
	if err := g.AddSystem(core.PhaseFrame, "disable", core.SystemFunc(func(*core.Context) error {
		return g.SetGroupEnabled("mission", false)
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystemToGroup(core.PhaseFrame, "mission", "later", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if updates != 0 {
		t.Fatalf("disabled system ran in current phase: updates=%d", updates)
	}
}

func TestGroupedSystemsRemainEnabledWithoutScenes(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var updates int
	if err := g.AddSystemToGroup(core.PhaseFixed, "custom", "custom.update", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatalf("AddSystemToGroup: %v", err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if updates != 1 {
		t.Fatalf("group updates: got %d, want 1", updates)
	}
}

func TestTimerEmitsCompletedSignal(t *testing.T) {
	// FixedTPS 4 => fixedDT 250ms; a 300ms one-shot completes on tick 2.
	g, err := New(WithTimer(), WithFixedTPS(4))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	hub := g.World().MustResource[*signal.Hub]()
	reader := hub.NewReader[timer.Completed]()

	entity, err := g.World().NewEntity(timer.NewTimer(300*time.Millisecond, false))
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	var got []timer.Completed
	g.AddSystem(core.PhaseFrame, "signal collector", core.SystemFunc(func(ctx *core.Context) error {
		return reader.Drain(func(c timer.Completed) error {
			got = append(got, c)
			return nil
		})
	}))

	step := 100 * time.Millisecond
	for range 6 {
		if err := g.Advance(step); err != nil {
			t.Fatalf("Advance: %v", err)
		}
	}
	if len(got) != 1 || got[0] != (timer.Completed{Entity: entity.ID(), Tick: 2}) {
		t.Fatalf("signals = %v, want one completion of entity %d on tick 2", got, entity.ID())
	}
}

func TestCollisionEmitsEnteredSignal(t *testing.T) {
	g, err := New(WithCollision())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	hub := g.World().MustResource[*signal.Hub]()

	shape, err := collision.NewCollider(collision.Box{
		Min: geom.Vector2{X: -1, Y: -1},
		Max: geom.Vector2{X: 1, Y: 1},
	})
	if err != nil {
		t.Fatalf("collider: %v", err)
	}
	left, err := g.World().NewEntity(core.Transform{}, shape)
	if err != nil {
		t.Fatalf("spawn left: %v", err)
	}
	_, err = g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: 1, Y: 0}}, shape)
	if err != nil {
		t.Fatalf("spawn right: %v", err)
	}

	reader := hub.NewReader[collision.Entered]()
	var got []collision.Entered
	g.AddSystem(core.PhaseFrame, "signal collector", core.SystemFunc(func(ctx *core.Context) error {
		return reader.Drain(func(e collision.Entered) error {
			got = append(got, e)
			return nil
		})
	}))

	step := time.Second / 60
	for range 3 {
		if err := g.Advance(step); err != nil {
			t.Fatalf("Advance: %v", err)
		}
	}

	if len(got) != 2 {
		t.Fatalf("entered signals = %v, want two (one per perspective)", got)
	}
	if got[0].Entity != left.ID() && got[1].Entity != left.ID() {
		t.Fatalf("no entered signal for entity %d: %v", left.ID(), got)
	}
}

func TestNewProvidesDebugPanel(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	panel := g.World().MustResource[*debug.Panel]()
	if panel == nil {
		t.Fatal("MustResource should return the engine-provided panel")
	}
	if !panel.Enabled(debug.FPS) || !panel.Enabled(debug.TPS) {
		t.Fatal("the engine's panel should start with every builtin enabled")
	}
}
