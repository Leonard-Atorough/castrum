package castrum

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/state"
	"github.com/Leonard-Atorough/castrum/timer"
	"github.com/Leonard-Atorough/castrum/tween"
)

func TestWithBindingsWiresInput(t *testing.T) {
	bindings := input.Bindings{
		"jump": {input.KeyInput{Key: input.KeySpace}},
	}
	g, err := New(WithBindings(bindings), WithFixedTPS(4))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	am, err := g.World().Resource[*input.ActionMap]()
	if err != nil {
		t.Fatalf("resolve action map: %v", err)
	}
	if g.Context().Actions != am {
		t.Fatal("Context.Actions should be the same map the resource resolves")
	}
	var framePresses int
	if err := g.AddSystem(core.PhaseFrame, "frame input", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Actions.JustPressed("jump") {
			framePresses++
		}
		return nil
	})); err != nil {
		t.Fatalf("AddSystem(frame): %v", err)
	}
	type edges struct{ pressed, released bool }
	var ticks []edges
	if err := g.AddSystem(core.PhaseFixed, "fixed input", core.SystemFunc(func(ctx *core.Context) error {
		ticks = append(ticks, edges{ctx.Actions.Pressed("jump"), ctx.Actions.Released("jump")})
		return nil
	})); err != nil {
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
		g.Context().Input = step.input
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
}

func TestOptionalTimerSystems(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []option
		enabled bool
	}{
		{"disabled", nil, false},
		{"enabled", []option{WithTimer()}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(tt.options...)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if tt.enabled {
				if _, err := g.World().Resource[*timer.Groups](); err != nil {
					t.Fatalf("resolve timer groups: %v", err)
				}
			}
			entity, err := g.World().NewEntity(timer.NewTimer(2*g.Options().FixedDT(), false))
			if err != nil {
				t.Fatalf("spawn timer: %v", err)
			}
			if err := g.Advance(2 * g.Options().FixedDT()); err != nil {
				t.Fatalf("Advance: %v", err)
			}
			tm, ok := entity.Component[timer.Timer](g.World())
			if !ok || (tm.CompletedOn != 0) != tt.enabled {
				t.Errorf("timer = %+v, found %t; want completed = %t", tm, ok, tt.enabled)
			}
		})
	}
}

func TestOptionalCollisionSystem(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []option
		enabled bool
	}{
		{"disabled", nil, false},
		{"enabled", []option{WithCollision()}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(tt.options...)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			var entities []*core.Entity
			for _, x := range []float64{0, 1} {
				collider, err := collision.NewCollider(collision.Circle{Radius: 8})
				if err != nil {
					t.Fatalf("NewCollider: %v", err)
				}
				entity, err := g.World().NewEntity(collider, core.Transform{Position: geom.Vector2{X: x}})
				if err != nil {
					t.Fatalf("spawn collider: %v", err)
				}
				entities = append(entities, entity)
			}
			if err := g.Advance(g.Options().FixedDT()); err != nil {
				t.Fatalf("Advance: %v", err)
			}
			for _, entity := range entities {
				contacts, ok := entity.Component[collision.Contacts](g.World())
				if ok != tt.enabled || (tt.enabled && len(contacts.Current) != 1) {
					t.Errorf("entity %d contacts = %+v, found %t; enabled %t", entity.ID(), contacts, ok, tt.enabled)
				}
			}
		})
	}
}

func TestOptionalAnimationSystem(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []option
		enabled bool
	}{
		{"disabled", nil, false},
		{"enabled", []option{WithAnimation()}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New(tt.options...)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if err := g.World().MustResource[*animation.ClipStore]().Add("clip", animation.Clip{
				Source: "sprites", Frames: []string{"frame_0", "frame_1"}, FPS: 60,
			}); err != nil {
				t.Fatalf("Add clip: %v", err)
			}
			entity, err := g.World().NewEntity(animation.Animation{Clip: "clip"}, render.Sprite{}, core.Transform{})
			if err != nil {
				t.Fatalf("spawn animated sprite: %v", err)
			}
			if err := g.Advance(g.Options().FixedDT()); err != nil {
				t.Fatalf("Advance: %v", err)
			}
			sprite, ok := entity.Component[render.Sprite](g.World())
			if !ok {
				t.Fatal("animated entity has no sprite")
			}
			if tt.enabled {
				if _, ok := sprite.Drawable.(render.AtlasSource); !ok {
					t.Errorf("drawable = %#v, want an atlas source", sprite.Drawable)
				}
			} else if sprite.Drawable != nil {
				t.Errorf("drawable = %#v, want nil", sprite.Drawable)
			}
		})
	}
}

func TestWithDefaultSystemsIdempotent(t *testing.T) {
	g, err := New(WithDefaultSystems(), WithTimer())
	if err != nil {
		t.Fatalf("New(WithDefaultSystems, WithTimer): %v", err)
	}
	entity, err := g.World().NewEntity(timer.NewTimer(2*g.Options().FixedDT(), false))
	if err != nil {
		t.Fatalf("spawn timer: %v", err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	tm, ok := entity.Component[timer.Timer](g.World())
	if !ok || tm.CompletedOn != 0 {
		t.Fatalf("timer after one tick = %+v, found %t; want a single registration", tm, ok)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	tm, ok = entity.Component[timer.Timer](g.World())
	if !ok || tm.CompletedOn == 0 {
		t.Fatalf("timer after two ticks = %+v, found %t; want completion", tm, ok)
	}
}

func TestWithScenesDoesNotControlGroups(t *testing.T) {
	g := newTestGame(t, WithScenes())
	director, err := g.World().Resource[*scene.Director]()
	if err != nil {
		t.Fatalf("resolve scene director: %v", err)
	}

	var updates int
	if err := g.AddSystemToGroup(core.PhaseFixed, "mission", "mission", core.SystemFunc(func(*core.Context) error {
		updates++
		return nil
	})); err != nil {
		t.Fatalf("AddSystemToGroup: %v", err)
	}
	blueprint := func(name string) *scene.Blueprint {
		return &scene.Blueprint{Name: name, EntityList: []scene.EntitySpec{{Name: "root"}}}
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

	id, err := director.Load(blueprint("mission"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	check("after load", 2)
	if err := g.SetGroupEnabled("mission", false); err != nil {
		t.Fatal(err)
	}
	if _, err := director.Load(blueprint("overlay")); err != nil {
		t.Fatalf("Load overlay: %v", err)
	}
	check("load while disabled", 2)
	if err := director.Update(id, blueprint("new mission")); err != nil {
		t.Fatalf("Update: %v", err)
	}
	check("update while disabled", 2)
	id, err = director.Switch(blueprint("replacement"))
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
	id, err = director.Load(blueprint("another"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	check("load while enabled", 4)
	if err := director.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	check("remove while enabled", 5)
}

func TestWithoutStateHasNoMachine(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts []option
	}{
		{"default", nil},
		{"default systems", []option{WithDefaultSystems()}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := newTestGame(t, tt.opts...)
			if _, err := g.World().Resource[*state.Machine[string]](); !errors.Is(err, core.ErrResourceNotRegistered) {
				t.Fatalf("state resource error = %v, want ErrResourceNotRegistered", err)
			}
		})
	}
}

func TestWithStateLastOptionWins(t *testing.T) {
	first := state.New(1)
	last := state.New("boot")
	g := newTestGame(t, WithState(first), WithState(last), WithState(last))
	if _, err := g.World().Resource[*state.Machine[int]](); !errors.Is(err, core.ErrResourceNotRegistered) {
		t.Fatalf("replaced resource error = %v, want ErrResourceNotRegistered", err)
	}
	if got := g.World().MustResource[*state.Machine[string]](); got != last {
		t.Fatal("last machine was not provided")
	}
}

func TestWithStateRejectsNilMachine(t *testing.T) {
	if g, err := New(WithState[string](nil)); err == nil || g != nil {
		t.Fatalf("New with nil machine = %v, %v, want nil game and an error", g, err)
	}
}

func TestWithStateRunsWhilePaused(t *testing.T) {
	machine := state.New("boot")
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

func TestStateTransitionsRunBeforeSceneOperationsAndFrameSystems(t *testing.T) {
	machine := state.New("boot")
	g := newTestGame(t, WithScenes(), WithState(machine))
	if got := g.World().MustResource[*state.Machine[string]](); got != machine {
		t.Fatal("state resource is not the configured machine")
	}
	director := g.World().MustResource[*scene.Director]()
	var got []string
	blueprint := func(name string) *scene.Blueprint {
		return &scene.Blueprint{
			Name:       name,
			EntityList: []scene.EntitySpec{{Name: "root"}},
			Hooks: scene.Hooks{Enter: func(*core.Context, *scene.Instance) error {
				got = append(got, "scene:"+name)
				return nil
			}},
		}
	}
	if _, err := director.Switch(blueprint("setup")); err != nil {
		t.Fatalf("queue setup scene: %v", err)
	}

	machine.OnEnter("boot", core.SystemFunc(func(*core.Context) error {
		got = append(got, "state:boot")
		_, err := director.Switch(blueprint("startup"))
		return err
	}))
	machine.OnExit("boot", core.SystemFunc(func(*core.Context) error {
		got = append(got, "state:exit-boot")
		return nil
	}))
	machine.OnEnter("play", core.SystemFunc(func(*core.Context) error {
		got = append(got, "state:play")
		_, err := director.Switch(blueprint("frame"))
		return err
	}))
	addTestSystem(t, g, core.PhaseStartup, "user.startup", core.SystemFunc(func(*core.Context) error {
		got = append(got, "user:startup")
		return nil
	}))
	addTestSystem(t, g, core.PhaseFrame, "user.frame", core.SystemFunc(func(*core.Context) error {
		got = append(got, "user:frame")
		return nil
	}))
	addTestSystem(t, g, core.PhaseFixed, "user.fixed", core.SystemFunc(func(*core.Context) error {
		got = append(got, "user:fixed")
		return nil
	}))

	if err := g.Startup(); err != nil {
		t.Fatalf("Startup(): %v", err)
	}
	wantStartup := []string{"state:boot", "scene:setup", "scene:startup", "user:startup"}
	if !slices.Equal(got, wantStartup) {
		t.Fatalf("startup order = %v, want %v", got, wantStartup)
	}

	if err := machine.Request("play"); err != nil {
		t.Fatalf("Request(play): %v", err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance(): %v", err)
	}
	want := []string{
		"state:boot", "scene:setup", "scene:startup", "user:startup",
		"state:exit-boot", "state:play", "scene:frame", "user:frame", "user:fixed",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("execution order = %v, want %v", got, want)
	}
}

func TestWithScenesProcessesStartupRequests(t *testing.T) {
	g, err := New(WithScenes())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	director := g.World().MustResource[*scene.Director]()
	id, err := director.Load(&scene.Blueprint{Name: "startup", EntityList: []scene.EntitySpec{{Name: "root"}}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var entered bool
	if err := g.AddSystem(core.PhaseStartup, "check scene", core.SystemFunc(func(*core.Context) error {
		for entry := range core.NewQuery(g.World()).With(scene.OwnershipRecord{}).Execute() {
			owner, _ := entry.Component[scene.OwnershipRecord]()
			entered = entered || owner.Instance == id
		}
		return nil
	})); err != nil {
		t.Fatalf("AddSystem: %v", err)
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}
	if !entered {
		t.Fatal("startup system did not see queued scene entities")
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
	if err := g.AddSystem(core.PhaseFrame, "signal collector", core.SystemFunc(func(ctx *core.Context) error {
		for event, ok := reader.Next(); ok; event, ok = reader.Next() {
			got = append(got, event)
		}
		return nil
	})); err != nil {
		t.Fatalf("AddSystem: %v", err)
	}

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
	if err := g.AddSystem(core.PhaseFrame, "signal collector", core.SystemFunc(func(ctx *core.Context) error {
		for event, ok := reader.Next(); ok; event, ok = reader.Next() {
			got = append(got, event)
		}
		return nil
	})); err != nil {
		t.Fatalf("AddSystem: %v", err)
	}

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

func TestTweenSchedulerClocks(t *testing.T) {
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

			g, err := New(WithFixedTPS(4))
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			fixed, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("fixed NewTween() error: %v", err)
			}
			frame, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("frame NewTween() error: %v", err)
			}
			var fixedCalls, frameCalls int
			if err := g.AddSystem(core.PhaseFixed, "tween.fixed", core.SystemFunc(func(ctx *core.Context) error {
				fixedCalls++
				if ctx.DeltaTime != 250*time.Millisecond {
					t.Errorf("fixed DeltaTime = %v, want 250ms", ctx.DeltaTime)
				}
				_, err := fixed.Update(ctx.DeltaTime)
				return err
			})); err != nil {
				t.Fatalf("AddSystem(fixed) error: %v", err)
			}
			if err := g.AddSystem(core.PhaseFrame, "tween.frame", core.SystemFunc(func(ctx *core.Context) error {
				frameCalls++
				_, err := frame.Update(ctx.DeltaTime)
				return err
			})); err != nil {
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
					if err := g.SetTimeScale(step.scale); err != nil {
						t.Fatalf("step %d: SetTimeScale() error: %v", i, err)
					}
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
