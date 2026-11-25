package castrum

// These tests drive a fully composed Game through its public surface and
// assert on behaviour that only emerges when several subsystems run together:
// phase ordering, signal visibility across phases, input edges across fixed
// ticks, scene and state transitions, and pause or time-scale interactions.
// Single-package rules belong in that package's own tests.

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/state"
	"github.com/Leonard-Atorough/castrum/tween"
)

type integrationAudioController struct {
	playbacks []audio.Playback
	entity    core.EntityID
	source    asset.ID
	finished  bool
	err       error
}

func (c *integrationAudioController) Sync(entity core.EntityID, source asset.ID, want audio.Playback) (bool, error) {
	c.playbacks = append(c.playbacks, want)
	c.entity, c.source = entity, source
	return !c.finished, c.err
}

func (c *integrationAudioController) Sweep() error { return nil }

func newTestAudioEntity(t *testing.T, g *Game, settings audio.Settings, components ...any) core.Entity {
	t.Helper()
	player, err := audio.NewPlayer(audio.Track("sound"), settings)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	entity, err := g.World().NewEntity(append([]any{player}, components...)...)
	if err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	return entity
}

// rootScene builds a one-node tree. The props string names the scene in
// lifecycle records.
var rootScene = scene.Of(func(string) scene.Node {
	return scene.Node{Key: "root"}
})

// recordEntry appends "scene:<name>" to log when the scene is entered.
func recordEntry(log *[]string, name string) scene.Option {
	return scene.WithHooks(scene.Hooks{
		Enter: func(*core.Context, *scene.Handle) error {
			*log = append(*log, "scene:"+name)
			return nil
		},
	})
}

func TestEngineDefaultsRunWithoutOptions(t *testing.T) {
	g := newTestGame(t)
	fixedDT := g.Options().FixedDT()

	timer, err := g.World().NewEntity(gametime.NewTimer(fixedDT, false))
	if err != nil {
		t.Fatalf("spawn timer: %v", err)
	}
	var colliders []core.Entity
	for _, x := range []float64{0, 1} {
		shape, err := collision.NewCollider(collision.Circle{Radius: 8})
		if err != nil {
			t.Fatalf("NewCollider: %v", err)
		}
		entity, err := g.World().NewEntity(shape, core.Transform{Position: geom.Vector2{X: x}})
		if err != nil {
			t.Fatalf("NewEntity: %v", err)
		}
		colliders = append(colliders, entity)
	}

	advanceTestGame(t, g, fixedDT)

	if tm, ok := timer.Component[gametime.Timer](g.World()); !ok || tm.CompletedOn == 0 {
		t.Errorf("timer = %+v, found %t; want completion without any option", tm, ok)
	}
	for _, entity := range colliders {
		contacts, ok := entity.Component[collision.Contacts](g.World())
		if !ok || len(contacts.Current) != 1 {
			t.Errorf("entity %d contacts = %+v, found %t; want one contact without any option", entity.ID(), contacts, ok)
		}
	}
}

func TestEngineSystemsRejectRelativeInsertion(t *testing.T) {
	g := newTestGame(t, WithScenes(), WithState(state.New[string]().WithInitial("boot")), WithBindings(input.Bindings{
		"jump": {input.KeyInput{Key: input.KeySpace}.Binding()},
	}))
	system := func(*core.Context) error { return nil }
	for _, entry := range []struct {
		phase core.Phase
		name  string
	}{
		{core.PhasePreStartup, state.SystemName}, {core.PhaseStartup, scene.StartupSystemName},
		{core.PhasePreFrame, input.FrameSystemName}, {core.PhasePreFrame, state.SystemName},
		{core.PhaseFrame, scene.FrameSystemName}, {core.PhaseFrame, audio.SystemName},
		{core.PhasePreFixed, core.PrevTransformSystemName},
		{core.PhasePreFixed, input.TickSystemName}, {core.PhaseFixed, gametime.TimerSystemName},
		{core.PhaseFixed, animation.SystemName}, {core.PhaseFixed, collision.SystemName},
	} {
		for _, option := range []SystemOption{Before(entry.name), After(entry.name)} {
			if err := g.AddSystem(entry.phase, "relative", system, option); err == nil {
				t.Errorf("relative insertion accepted engine system %q", entry.name)
			}
		}
	}
}

func TestFixedPhaseSignalsAndMovementShareTick(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4))
	hub := g.World().MustResource[*signal.Hub]()
	timerReader := hub.NewReader[gametime.Completed]()
	animationReader := hub.NewReader[animation.Completed]()
	collisionReader := hub.NewReader[collision.Entered]()
	if _, err := g.World().NewEntity(gametime.NewTimer(250*time.Millisecond, false)); err != nil {
		t.Fatal(err)
	}
	sets, err := animation.NewSets(map[string]animation.Set{
		"test": {Name: "test", States: map[string]animation.State{
			"run": {Name: "run", Tracks: []animation.Track{
				{Name: "body", Source: "sprites.json", Frames: []string{"frame_0", "frame_1"}, FPS: 60},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.World().SetResource(sets)
	animator, err := animation.NewAnimator("test", "run")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.World().NewEntity(animator, core.Transform{}); err != nil {
		t.Fatal(err)
	}
	shape, err := collision.NewCollider(collision.Circle{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	moving, err := g.World().NewEntity(core.Transform{Position: geom.Vector2{X: 10}}, shape)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.World().NewEntity(core.Transform{}, shape); err != nil {
		t.Fatal(err)
	}
	addTestSystem(t, g, core.PhasePreFixed, "move", func(ctx *core.Context) error {
		return moving.Update(ctx.World, func(tr *core.Transform) { tr.Position.X = 1 })
	})
	addTestSystem(t, g, core.PhaseFixed, "observe", func(*core.Context) error {
		if _, ok := timerReader.Next(); !ok {
			t.Error("timer signal not visible in same tick")
		}
		if _, ok := animationReader.Next(); !ok {
			t.Error("animation signal not visible in same tick")
		}
		if _, ok := collisionReader.Next(); !ok {
			t.Error("pre-fixed movement not detected in same tick")
		}
		return nil
	})
	advanceTestGame(t, g, 250*time.Millisecond)
}

func TestCollisionSignalsReachFrameAfterFixedTicks(t *testing.T) {
	g := New()
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
	if _, err := g.World().NewEntity(core.Transform{Position: geom.Vector2{X: 1, Y: 0}}, shape); err != nil {
		t.Fatalf("spawn right: %v", err)
	}

	reader := hub.NewReader[collision.Entered]()
	var got []collision.Entered
	addTestSystem(t, g, core.PhaseFrame, "signal collector", func(*core.Context) error {
		for event, ok := reader.Next(); ok; event, ok = reader.Next() {
			got = append(got, event)
		}
		return nil
	})

	step := time.Second / 60
	for range 3 {
		advanceTestGame(t, g, step)
	}

	if len(got) != 2 {
		t.Fatalf("entered signals = %v, want two (one per perspective)", got)
	}
	if got[0].Entity != left && got[1].Entity != left {
		t.Fatalf("no entered signal for entity %d: %v", left.ID(), got)
	}
}

func TestBindingEdgesReachEveryPhase(t *testing.T) {
	bindings := input.Bindings{
		"jump": {input.KeyInput{Key: input.KeySpace}.Binding()},
	}
	g := New(WithBindings(bindings), WithFixedTPS(4))
	am, err := g.World().Resource[input.Actions]()
	if err != nil {
		t.Fatalf("resolve action map: %v", err)
	}
	var framePresses int
	if err := g.AddSystem(core.PhaseFrame, "frame input", func(*core.Context) error {
		if am.JustPressed("jump") {
			framePresses++
		}
		return nil
	}); err != nil {
		t.Fatalf("AddSystem(frame): %v", err)
	}
	type edges struct{ pressed, released bool }
	var ticks, preTicks []edges
	if err := g.AddSystem(core.PhasePreFixed, "pre-fixed input", func(*core.Context) error {
		preTicks = append(preTicks, edges{am.Pressed("jump"), am.Released("jump")})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "fixed input", func(*core.Context) error {
		ticks = append(ticks, edges{am.Pressed("jump"), am.Released("jump")})
		return nil
	}); err != nil {
		t.Fatalf("AddSystem(fixed): %v", err)
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}
	press := &input.Snapshot{}
	press.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	release := &input.Snapshot{}
	release.Keys[input.KeySpace] = input.State{Released: true}
	for _, step := range []struct {
		input *input.Snapshot
		delta time.Duration
	}{
		{press, 100 * time.Millisecond},
		{release, 100 * time.Millisecond},
		{&input.Snapshot{}, 50 * time.Millisecond},
		{&input.Snapshot{}, 250 * time.Millisecond},
	} {
		current, _ := g.World().Resource[*input.Snapshot]()
		*current = *step.input
		if err := g.Advance(step.delta); err != nil {
			t.Fatalf("Advance: %v", err)
		}
	}
	if framePresses != 1 {
		t.Errorf("frame presses = %d, want 1", framePresses)
	}
	if len(ticks) != 2 || ticks[0] != (edges{true, true}) || ticks[1] != (edges{}) {
		t.Errorf("tick edges = %v, want press and release on first tick only", ticks)
	}
	if !slices.Equal(preTicks, ticks) {
		t.Errorf("pre-fixed edges = %v, fixed edges = %v; want identical", preTicks, ticks)
	}
}

func TestPressEdgeIsSeenOnceAcrossCatchUpTicks(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4), WithMaxFrameTime(time.Second), WithBindings(input.Bindings{
		"jump": {input.KeyInput{Key: input.KeySpace}.Binding()},
	}))
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	current, _ := g.World().Resource[*input.Snapshot]()
	*current = *snapshot
	var got []bool
	addTestSystem(t, g, core.PhasePreFixed, "input", func(ctx *core.Context) error {
		am, _ := ctx.World.Resource[input.Actions]()
		got = append(got, am.Pressed("jump"))
		return nil
	})
	advanceTestGame(t, g, 750*time.Millisecond)
	if want := []bool{true, false, false}; !slices.Equal(got, want) {
		t.Fatalf("catch-up tick edges = %v, want %v", got, want)
	}
}

func TestSpatialAudioFollowsFixedMovementDuringPause(t *testing.T) {
	g := newTestGame(t, WithFixedTPS(4))
	listener := g.World().MustResource[*audio.Listener]()
	listener.Enabled, listener.Mode, listener.Range = true, audio.FixedPosition, 100
	settings := audio.DefaultSettings()
	settings.Spatial = true
	entity := newTestAudioEntity(t, g, settings, core.Transform{})
	controller := &integrationAudioController{}
	if err := g.World().MustResource[*audio.Backend]().Attach(controller); err != nil {
		t.Fatal(err)
	}
	addTestSystem(t, g, core.PhaseFixed, "move", func(ctx *core.Context) error {
		return entity.Update(ctx.World, func(tr *core.Transform) { tr.Position.X += 50 })
	})
	advanceTestGame(t, g, 250*time.Millisecond)
	g.Pause()
	advanceTestGame(t, g, 250*time.Millisecond)
	if len(controller.playbacks) != 2 || controller.playbacks[0].Volume != 1 ||
		controller.playbacks[1].Volume != 0.5 || !controller.playbacks[1].Playing || controller.playbacks[1].Restart {
		t.Fatalf("frame playbacks = %+v, want initial position then fixed movement during game pause", controller.playbacks)
	}
}

func TestAudioFinishedSignalReachesPostFrameOnly(t *testing.T) {
	g := newTestGame(t)
	controller := &integrationAudioController{finished: true}
	entity := newTestAudioEntity(t, g, audio.DefaultSettings())
	hub := g.World().MustResource[*signal.Hub]()
	post := hub.NewReader[audio.Finished]()
	fixed := hub.NewReader[audio.Finished]()
	nextFrame := hub.NewReader[audio.Finished]()
	addTestSystem(t, g, core.PhaseFrame, "configure", func(*core.Context) error {
		return g.World().MustResource[*audio.Mixer]().SetMaster(0.5)
	})
	if err := g.World().MustResource[*audio.Backend]().Attach(controller); err != nil {
		t.Fatal(err)
	}
	var finished audio.Finished
	var received bool
	addTestSystem(t, g, core.PhasePostFrame, "finished", func(*core.Context) error {
		if signal, ok := post.Next(); ok {
			finished, received = signal, true
		}
		return nil
	})
	var fixedTicks int
	addTestSystem(t, g, core.PhasePreFixed, "check expiration", func(*core.Context) error {
		fixedTicks++
		if _, ok := fixed.Next(); ok {
			t.Fatal("frame signal survived into fixed tick")
		}
		return nil
	})
	advanceTestGame(t, g, g.Options().FixedDT())
	if !received || finished.Entity != entity || finished.Audio != "sound" {
		t.Fatalf("finished = %+v, received = %t; want entity %d and audio sound", finished, received, entity.ID())
	}
	if fixedTicks != 1 || len(controller.playbacks) != 1 || controller.playbacks[0].Volume != 0.5 {
		t.Fatalf("fixed ticks = %d, playbacks = %+v", fixedTicks, controller.playbacks)
	}
	if controller.entity != entity.ID() || controller.source != "sound" {
		t.Fatalf("controller synced entity %d and audio %q; want entity %d and audio sound", controller.entity, controller.source, entity.ID())
	}
	checkedNextFrame := false
	addTestSystem(t, g, core.PhasePreFrame, "check next frame", func(*core.Context) error {
		checkedNextFrame = true
		if _, ok := nextFrame.Next(); ok {
			t.Fatal("frame signal survived into the next frame")
		}
		return nil
	})
	advanceTestGame(t, g, time.Millisecond)
	if !checkedNextFrame {
		t.Fatal("next-frame signal check did not run")
	}
}

func TestAudioBackendFailureSurfacesThroughAdvance(t *testing.T) {
	g := newTestGame(t)
	newTestAudioEntity(t, g, audio.DefaultSettings())
	advanceTestGame(t, g, time.Millisecond)

	wantErr := errors.New("backend failed")
	controller := &integrationAudioController{err: wantErr}
	if err := g.World().MustResource[*audio.Backend]().Attach(controller); err != nil {
		t.Fatal(err)
	}
	err := g.Advance(time.Millisecond)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), audio.SystemName) {
		t.Fatalf("Advance error = %v, want backend failure from %s", err, audio.SystemName)
	}
	controller.err = nil
	advanceTestGame(t, g, time.Millisecond)
	if len(controller.playbacks) != 2 || !controller.playbacks[0].Restart || !controller.playbacks[1].Restart {
		t.Fatalf("playbacks = %+v; headless and failed syncs must not consume the initial restart", controller.playbacks)
	}
}

func TestStateTransitionsOrderAgainstScenesAndFrameWork(t *testing.T) {
	machine := state.New[string]().WithInitial("boot")
	g := newTestGame(t, WithScenes(), WithState(machine), WithBindings(input.Bindings{
		"start": {input.KeyInput{Key: input.KeySpace}.Binding()},
	}))
	director := g.World().MustResource[*scene.Director]()
	hub := g.World().MustResource[*signal.Hub]()
	startupSignal := hub.NewReader[state.Initialized[string]]()
	frameSignal := hub.NewReader[state.Initialized[string]]()
	changed := hub.NewReader[state.Changed[string]]()
	fixedSignal := hub.NewReader[state.Changed[string]]()
	var got []string

	// Superseded below by the same-batch Switch from the boot hook, so it never enters.
	if _, err := director.Switch(rootScene, "setup", recordEntry(&got, "setup")); err != nil {
		t.Fatalf("queue setup scene: %v", err)
	}
	machine.OnEnter("boot", func(*core.Context) error {
		got = append(got, "state:boot")
		_, err := director.Switch(rootScene, "startup", recordEntry(&got, "startup"))
		return err
	})
	machine.OnExit("boot", func(*core.Context) error {
		got = append(got, "state:exit-boot")
		return nil
	})
	machine.OnEnter("play", func(ctx *core.Context) error {
		am, _ := ctx.World.Resource[input.Actions]()
		if !am.JustPressed("start") {
			t.Fatal("state hook ran before frame input updated")
		}
		got = append(got, "state:play")
		_, err := director.Switch(rootScene, "frame", recordEntry(&got, "frame"))
		return err
	})
	loaded := false
	addTestSystem(t, g, core.PhasePreFrame, "user.pre-frame", func(*core.Context) error {
		got = append(got, "user:pre-frame")
		if loaded {
			return nil
		}
		loaded = true
		_, err := director.Load(rootScene, "pre-frame", recordEntry(&got, "pre-frame"))
		return err
	})
	addTestSystem(t, g, core.PhaseStartup, "user.startup", func(*core.Context) error {
		if event, ok := startupSignal.Next(); !ok || event.State != "boot" {
			t.Fatalf("startup signal = %+v, %t", event, ok)
		}
		got = append(got, "user:startup")
		return nil
	})
	frames := 0
	addTestSystem(t, g, core.PhaseFrame, "user.frame", func(*core.Context) error {
		frames++
		if frames == 1 {
			if event, ok := frameSignal.Next(); !ok || event.State != "boot" {
				t.Fatalf("first-frame initialization signal = %+v, %t", event, ok)
			}
			if event, ok := changed.Next(); !ok || event != (state.Changed[string]{From: "boot", To: "play"}) {
				t.Fatalf("frame change signal = %+v, %t", event, ok)
			}
		}
		got = append(got, "user:frame")
		return nil
	})
	addTestSystem(t, g, core.PhaseFixed, "user.fixed", func(*core.Context) error {
		if _, ok := fixedSignal.Next(); ok {
			t.Fatal("state signal did not expire before fixed ticks")
		}
		got = append(got, "user:fixed")
		return nil
	})

	if err := g.Startup(); err != nil {
		t.Fatalf("Startup(): %v", err)
	}
	wantStartup := []string{"state:boot", "scene:startup", "user:startup"}
	if !slices.Equal(got, wantStartup) {
		t.Fatalf("startup order = %v, want %v", got, wantStartup)
	}

	if err := machine.Request("play"); err != nil {
		t.Fatalf("Request(play): %v", err)
	}
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	current, _ := g.World().Resource[*input.Snapshot]()
	*current = *snapshot
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance(): %v", err)
	}
	// Scenes drain before the state system within PreFrame, so the scene
	// requests queued by the state and user hooks apply on the next Advance.
	wantFirst := []string{
		"state:boot", "scene:startup", "user:startup",
		"state:exit-boot", "state:play", "user:pre-frame", "user:frame", "user:fixed",
	}
	if !slices.Equal(got, wantFirst) {
		t.Fatalf("first frame order = %v, want %v", got, wantFirst)
	}

	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("second Advance(): %v", err)
	}
	wantSecond := append(slices.Clone(wantFirst),
		"scene:frame", "scene:pre-frame", "user:pre-frame", "user:frame", "user:fixed")
	if !slices.Equal(got, wantSecond) {
		t.Fatalf("second frame order = %v, want %v", got, wantSecond)
	}
}

func TestStateMachinesCoordinateInOrder(t *testing.T) {
	first := state.New[int]().WithInitial(1)
	last := state.New[string]().WithInitial("boot")
	g := newTestGame(t, WithState(first), WithState(last))
	var got []string
	first.OnEnter(1, func(*core.Context) error { got = append(got, "first"); return nil })
	last.OnEnter("boot", func(*core.Context) error { got = append(got, "last"); return nil })
	first.OnEnter(2, func(*core.Context) error { return last.Request("play") })
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"first", "last"}) {
		t.Fatalf("initial hook order = %v", got)
	}
	if err := first.Request(2); err != nil {
		t.Fatal(err)
	}
	advanceTestGame(t, g, 0)
	if !first.Is(2) || !last.Is("play") {
		t.Fatal("later machine did not process the earlier machine's hook request")
	}

	boom := errors.New("first machine failed")
	first.OnEnter(3, func(*core.Context) error { return boom })
	if err := first.Request(3); err != nil {
		t.Fatal(err)
	}
	if err := last.Request("done"); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(0); !errors.Is(err, boom) || !strings.Contains(err.Error(), "*state.Machine[int]") {
		t.Fatalf("machine failure = %v, want typed wrapped hook error", err)
	}
	if pending, ok := last.Pending(); !last.Is("play") || !ok || pending != "done" {
		t.Fatal("earlier machine failure consumed a later machine's request")
	}
	advanceTestGame(t, g, 0)
	if !last.Is("done") {
		t.Fatal("later machine's preserved request was not applied")
	}
}

func TestStateMachinesAdvanceWhilePaused(t *testing.T) {
	machine := state.New[string]().WithInitial("boot")
	g := newTestGame(t, WithState(machine))
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	g.Pause()
	if err := machine.Request("play"); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatal(err)
	}
	if !machine.Is("play") {
		t.Fatal("paused game did not process the state transition")
	}
}

func TestSceneLifecycleLeavesGroupStateToGameCode(t *testing.T) {
	g := newTestGame(t, WithScenes())
	director := g.World().MustResource[*scene.Director]()

	var updates int
	if err := g.AddSystem(core.PhaseFixed, "mission", func(*core.Context) error {
		updates++
		return nil
	}, InGroup("mission")); err != nil {
		t.Fatalf("AddSystem: %v", err)
	}
	check := func(action string, want int) {
		t.Helper()
		advanceTestGame(t, g, g.Options().FixedDT())
		if updates != want {
			t.Fatalf("%s: group updates = %d, want %d", action, updates, want)
		}
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}
	check("before load", 1)

	id, err := director.Load(rootScene, "mission")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	check("after load", 2)
	if err := g.SetGroupEnabled("mission", false); err != nil {
		t.Fatal(err)
	}
	if _, err := director.Load(rootScene, "overlay"); err != nil {
		t.Fatalf("Load overlay: %v", err)
	}
	check("load while disabled", 2)
	if err := director.Update(id, rootScene, "new mission"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	check("update while disabled", 2)
	id, err = director.Switch(rootScene, "replacement")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}
	check("switch while disabled", 2)
	if err := director.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	check("remove while disabled", 2)
	if err := g.SetGroupEnabled("mission", true); err != nil {
		t.Fatal(err)
	}
	check("explicit enable", 3)
	id, err = director.Load(rootScene, "another")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	check("load while enabled", 4)
	if err := director.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	check("remove while enabled", 5)
}

func TestTweenClocksFollowPauseAndTimeScale(t *testing.T) {
	t.Parallel()

	type step struct {
		delta      time.Duration
		pause      bool
		resume     bool
		scale      float64
		fixedTime  float64
		frameTime  float64
		fixedCalls int
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "normal accumulation",
			steps: []step{
				{delta: 125 * time.Millisecond, frameTime: 0.125},
				{delta: 125 * time.Millisecond, fixedTime: 0.25, frameTime: 0.25, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 0.5, fixedCalls: 2},
			},
		},
		{
			name: "pause and resume preserve remainder",
			steps: []step{
				{delta: 125 * time.Millisecond, frameTime: 0.125},
				{delta: 250 * time.Millisecond, pause: true, frameTime: 0.375},
				{delta: 250 * time.Millisecond, frameTime: 0.625},
				{delta: 125 * time.Millisecond, resume: true, fixedTime: 0.25, frameTime: 0.75, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 1, fixedCalls: 2},
			},
		},
		{
			name: "half speed",
			steps: []step{
				{delta: 250 * time.Millisecond, scale: 0.5, frameTime: 0.25},
				{delta: 250 * time.Millisecond, fixedTime: 0.25, frameTime: 0.5, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.25, frameTime: 0.75, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 1, fixedCalls: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := New(WithFixedTPS(4))
			fixed, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("fixed NewTween() error: %v", err)
			}
			frame, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("frame NewTween() error: %v", err)
			}
			var fixedCalls, frameCalls int
			if err := g.AddSystem(core.PhaseFixed, "tween.fixed", func(ctx *core.Context) error {
				fixedCalls++
				if ctx.DeltaTime != 250*time.Millisecond {
					t.Errorf("fixed DeltaTime = %v, want 250ms", ctx.DeltaTime)
				}
				_, err := fixed.Update(ctx.DeltaTime)
				return err
			}); err != nil {
				t.Fatalf("AddSystem(fixed) error: %v", err)
			}
			if err := g.AddSystem(core.PhaseFrame, "tween.frame", func(ctx *core.Context) error {
				frameCalls++
				_, err := frame.Update(ctx.DeltaTime)
				return err
			}); err != nil {
				t.Fatalf("AddSystem(frame) error: %v", err)
			}
			if err := g.Startup(); err != nil {
				t.Fatalf("Startup() error: %v", err)
			}

			for i, step := range tt.steps {
				if step.pause {
					g.Pause()
				}
				if step.resume {
					g.Resume()
				}
				if step.scale > 0 {
					g.SetTimeScale(step.scale)
				}
				if err := g.Advance(step.delta); err != nil {
					t.Fatalf("step %d: Advance() error: %v", i, err)
				}
				if fixedCalls != step.fixedCalls || frameCalls != i+1 {
					t.Errorf("step %d: fixed/frame calls = %d/%d, want %d/%d",
						i, fixedCalls, frameCalls, step.fixedCalls, i+1)
				}
				for _, clock := range []struct {
					name  string
					tween *tween.Tween
					time  float64
				}{
					{"fixed", &fixed, step.fixedTime},
					{"frame", &frame, step.frameTime},
				} {
					want := clock.time * clock.time
					if got := clock.tween.Progress(); math.Abs(got-want) > 1e-12 {
						t.Errorf("step %d: %s Progress() = %v, want %v", i, clock.name, got, want)
					}
				}
			}
		})
	}
}

func TestSchedulerWiredIntoEngine(t *testing.T) {
	g := newTestGame(t)
	scheduler := g.World().MustResource[*gametime.Scheduler]()
	var ran int
	if err := scheduler.Every("wired", 1, func(*core.Context) error {
		ran++
		return nil
	}); err != nil {
		t.Fatalf("Every: %v", err)
	}
	advanceTestGame(t, g, 2*g.Options().FixedDT())
	if ran != 2 {
		t.Fatalf("scheduled entries ran %d times in two ticks, want 2", ran)
	}
}
