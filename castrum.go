package castrum

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/serialization"

	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

// Game owns the world, configuration, and system schedules for a game.
// A runner drives the platform loop and supplies the window, draw surface,
// and input.
//
// Call Game methods and access its world and context only from the thread
// that runs its systems.
type Game struct {
	opts        Options
	world       *core.World
	ctx         core.Context
	schedules   map[core.Phase]*runtime.Schedule
	groupStates map[string]bool

	mainCamera core.Entity
	hub        *signal.Hub

	acc   time.Duration
	state lifecycle
	quit  bool

	sim  *gametime.Simulation
	real *gametime.Real
}

// lifecycle tracks startup and run separately so embedders can run startup
// phases without a runner.
type lifecycle uint8

const (
	lifecycleStartupBegun lifecycle = 1 << iota
	lifecycleRunBegun
)

func (l lifecycle) has(flag lifecycle) bool { return l&flag != 0 }

// New creates a [Game] with default options overridden by opts. It panics
// if the resulting configuration is invalid.
//
// The asset loader resolves engine-shipped assets under "builtin/" alongside
// the configured filesystem.
func New(opts ...option) *Game {
	g, err := newGame(opts...)
	if err != nil {
		panic(err)
	}
	return g
}

func newGame(opts ...option) (*Game, error) {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.finalize(); err != nil {
		return nil, err
	}
	g := &Game{
		opts:        options,
		world:       core.NewWorld(),
		schedules:   map[core.Phase]*runtime.Schedule{},
		groupStates: make(map[string]bool),
	}
	g.ctx.World = g.world

	// Provide a default static window until a runner replaces it.
	g.world.SetResource[render.Window](render.NewStaticWindow(render.Viewport{
		LogicalWidth: 1280, LogicalHeight: 720,
		WindowWidth: 1280, WindowHeight: 720,
	}))

	sim := gametime.NewSimulation(options.fixedDT)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Simulation, error) {
		return sim, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide simulation time: %w", err)
	}
	g.sim = sim

	// The runner overwrites the snapshot resource before each Advance. The
	// zero value reads as no input until the first poll.
	if err := g.world.Provide(func(*core.World) (*input.Snapshot, error) {
		return &input.Snapshot{}, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide input snapshot: %w", err)
	}

	real := gametime.NewReal(options.MaxFrameTime)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Real, error) {
		return real, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide real time: %w", err)
	}
	g.real = real

	codecs := options.AssetCodecs
	if codecs == nil {
		codecs = NewAssetCodecs()
	}
	loader, err := asset.NewLoader(builtin.WithFilesystem(options.Filesystem), asset.LoaderOptions{
		Codecs: codecs,
	})
	if err != nil {
		return nil, fmt.Errorf("castrum: create asset loader: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*asset.Loader, error) {
		return loader, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide asset loader: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*asset.Codecs, error) {
		return codecs, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide asset codecs: %w", err)
	}

	reg := serialization.NewRegistry(g.world)
	if err := gametime.RegisterBlueprints(reg); err != nil {
		return nil, fmt.Errorf("castrum: engine blueprints: %w", err)
	}
	if err := registerEngine(reg); err != nil {
		return nil, fmt.Errorf("castrum: engine components: %w", err)
	}
	if err := registerAssets(reg); err != nil {
		return nil, fmt.Errorf("castrum: engine assets: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*serialization.Registry, error) {
		return reg, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide blueprint registry: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFixed, core.PrevTransformSystemName, core.NewPrevTransformCapture()); err != nil {
		return nil, err
	}

	scheduler := gametime.NewScheduler(g.world)
	if err := g.world.Provide(func(*core.World) (*gametime.Scheduler, error) {
		return scheduler, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide scheduler: %w", err)
	}
	if err := g.addEngineFirst(core.PhasePreFixed, gametime.SchedulerSystemName, scheduler.System()); err != nil {
		return nil, err
	}

	if err := g.world.Provide(func(*core.World) (*gametime.Groups, error) {
		return gametime.NewGroups(), nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide timer groups: %w", err)
	}
	if err := g.addEngineFirst(core.PhaseFixed, gametime.TimerSystemName, gametime.NewTimerSystem()); err != nil {
		return nil, err
	}
	if err := g.addEngineFirst(core.PhaseFixed, animation.SpriteAnimationSystem, animation.NewSpriteAnimationSystem()); err != nil {
		return nil, err
	}
	if err := g.addEngineFirst(core.PhaseFixed, animation.SystemName, animation.Animate); err != nil {
		return nil, err
	}
	if err := g.world.ProvideEager(func(*core.World) (*collision.Queries, error) {
		return collision.NewQueries()
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide collision queries: %w", err)
	}
	if err := g.addEngineFirst(core.PhaseFixed, collision.SystemName, collision.NewSystem()); err != nil {
		return nil, err
	}

	hub := signal.NewHub()
	if err := g.world.Provide(func(*core.World) (*signal.Hub, error) {
		return hub, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide signal hub: %w", err)
	}
	g.hub = hub
	if err := g.schedule(core.PhasePreFrame).SetOpening(runtime.Entry{
		Name: "engine.signal-begin-frame",
		System: func(*core.Context) error {
			hub.BeginFrame()
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("castrum: register signal begin-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFrame).SetClosing(runtime.Entry{
		Name: "engine.signal-end-frame",
		System: func(*core.Context) error {
			hub.EndFrame()
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("castrum: register signal end-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFixed).SetClosing(runtime.Entry{
		Name: "engine.signal-end-tick",
		System: func(*core.Context) error {
			hub.EndTick()
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("castrum: register signal end-tick system: %w", err)
	}

	panel := debug.NewPanel()
	if err := g.world.Provide(func(*core.World) (*debug.Panel, error) {
		return panel, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide debug panel: %w", err)
	}

	mixer := audio.NewMixer()
	if err := g.world.Provide(func(*core.World) (*audio.Mixer, error) {
		return mixer, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide audio mixer: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*audio.Listener, error) {
		return &audio.Listener{}, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide audio listener: %w", err)
	}
	backend := &audio.Backend{}
	if err := g.world.Provide(func(*core.World) (*audio.Backend, error) {
		return backend, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide audio backend: %w", err)
	}
	var reconciler core.SystemFunc
	// Capture before other frame systems so mid-phase attachments wait for the next pass.
	if err := g.addEngineFirst(core.PhaseFrame, "engine.audio-prepare", func(*core.Context) error {
		if controller := backend.Controller(); reconciler == nil && controller != nil {
			reconciler = audio.NewAudioSystem(controller, mixer)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := g.schedule(core.PhaseFrame).AddLast(runtime.Entry{
		Name: audio.SystemName,
		System: func(ctx *core.Context) error {
			if reconciler == nil {
				return nil
			}
			return reconciler(ctx)
		},
	}); err != nil {
		return nil, fmt.Errorf("castrum: register audio system: %w", err)
	}

	if options.InputBindings != nil {
		if err := g.wireInput(options.InputBindings, options.InputDeadzone); err != nil {
			return nil, err
		}
	}

	// Features are applied in registration order: the order options were
	// passed to New decides the order their systems are registered in. Input
	// wiring runs first so feature systems in the same phases see fresh
	// actions.
	for _, f := range options.features {
		if err := f.enable(g, &options); err != nil {
			return nil, fmt.Errorf("castrum: enable %s: %w", f.name, err)
		}
	}

	// The engine's default camera yields to user-spawned primary cameras.
	camera, err := render.SpawnEngineCamera(g.world)
	if err != nil {
		return nil, fmt.Errorf("castrum: spawn main camera: %w", err)
	}
	g.mainCamera = camera
	return g, nil
}

// wireInput provides the action map and registers the engine's input
// systems first in their schedules, so feature and user systems in those
// phases see fresh actions.
func (g *Game) wireInput(bindings input.Bindings, deadzone float64) error {
	am, err := input.New(bindings, deadzone)
	if err != nil {
		return fmt.Errorf("castrum: input bindings: %w", err)
	}

	if err := g.world.Provide(func(*core.World) (input.Actions, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide action map: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (input.Edges, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide input edges: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFrame, input.FrameSystemName, input.NewFrameSystem(am)); err != nil {
		return err
	}
	return g.addEngineFirst(core.PhasePreFixed, input.TickSystemName, input.NewTickSystem(am))
}

// Options returns a copy of the game's configuration.
func (g *Game) Options() Options {
	return g.opts
}

// World returns the game's [core.World].
func (g *Game) World() *core.World {
	return g.world
}

// MainCamera returns the default camera created by [New], positioned at the
// world origin with zoom 1. Set its [core.Transform] to move it. A user-spawned
// primary camera takes precedence.
func (g *Game) MainCamera() core.Entity {
	return g.mainCamera
}

type systemOptionKind uint8

const (
	systemOptionBefore systemOptionKind = iota + 1
	systemOptionAfter
	systemOptionGroup
)

// SystemOption configures a system registration passed to [Game.AddSystem].
// Its zero value is invalid.
type SystemOption struct {
	kind  systemOptionKind
	value string
}

// Before places a system immediately before a registered user system in the same phase.
func Before(target string) SystemOption {
	return SystemOption{kind: systemOptionBefore, value: target}
}

// After places a system immediately after a registered user system in the same phase.
func After(target string) SystemOption {
	return SystemOption{kind: systemOptionAfter, value: target}
}

// InGroup assigns a system to a named group controlled by [Game.SetGroupEnabled].
// A new group starts enabled.
func InGroup(group string) SystemOption {
	return SystemOption{kind: systemOptionGroup, value: group}
}

// AddSystem registers a named system in phase. By default it runs after
// previously registered user systems. [Before] and [After] change its
// position, and [InGroup] assigns it to a group.
//
// AddSystem panics for an empty name or multiple position options. It
// returns an error for an invalid phase, a duplicate name in the phase,
// an invalid target, an invalid option, an empty group, or multiple group options.
// It also returns an error for startup phases once [Game.Startup] begins.
//
// Systems added during a phase run are eligible on the next run of that phase.
func (g *Game) AddSystem(phase core.Phase, name string, system core.SystemFunc, options ...SystemOption) error {
	if phase < core.PhasePreStartup || phase > core.PhasePostFixed {
		return fmt.Errorf("castrum: AddSystem: invalid phase %s", phase)
	}
	if phase <= core.PhasePostStartup && g.state.has(lifecycleStartupBegun) {
		return fmt.Errorf("castrum: AddSystem: startup has already begun (phase %s)", phase)
	}
	if name == "" {
		panic(fmt.Errorf("castrum: AddSystem: system name must not be empty (phase %s)", phase))
	}
	var position systemOptionKind
	var target, group string
	groupSet := false
	for _, option := range options {
		switch option.kind {
		case systemOptionBefore, systemOptionAfter:
			if position != 0 {
				panic(fmt.Errorf("castrum: AddSystem: multiple position options"))
			}
			if option.value == "" {
				return fmt.Errorf("castrum: AddSystem: target must not be empty")
			}
			position, target = option.kind, option.value
		case systemOptionGroup:
			if groupSet {
				return fmt.Errorf("castrum: AddSystem: multiple group options")
			}
			if option.value == "" {
				return fmt.Errorf("castrum: AddSystem: group must not be empty")
			}
			group, groupSet = option.value, true
		default:
			return fmt.Errorf("castrum: AddSystem: invalid system option")
		}
	}
	entry := runtime.Entry{Name: name, System: system, Group: group}
	sched, exists := g.schedules[phase]
	if !exists {
		if position != 0 {
			return fmt.Errorf("castrum: AddSystem: target system %q not found in phase %s", target, phase)
		}
		sched = g.schedule(phase)
	}
	var err error
	switch position {
	case systemOptionBefore:
		err = sched.AddBefore(target, entry)
	case systemOptionAfter:
		err = sched.AddAfter(target, entry)
	default:
		err = sched.Add(entry)
	}
	if err != nil {
		return fmt.Errorf("castrum: AddSystem: %w (phase %s)", err, phase)
	}
	return nil
}

func (g *Game) schedule(phase core.Phase) *runtime.Schedule {
	sched, ok := g.schedules[phase]
	if !ok {
		sched = runtime.NewSchedule(phase, g.groupStates)
		g.schedules[phase] = sched
	}

	return sched
}

func (g *Game) addEngineFirst(phase core.Phase, name string, system core.SystemFunc) error {
	if err := g.schedule(phase).AddFirst(runtime.Entry{Name: name, System: system}); err != nil {
		return fmt.Errorf("castrum: register engine system %q in phase %s: %w", name, phase, err)
	}
	return nil
}

// SetGroupEnabled enables or disables a registered group across all phases.
// The state also applies to systems registered in the group later. It returns
// an error if group is empty or has no registered systems. A change affects
// subsequent systems, including later systems in the current phase.
func (g *Game) SetGroupEnabled(group string, enabled bool) error {
	if group == "" {
		return fmt.Errorf("castrum: SetGroupEnabled: group must not be empty")
	}
	if _, registered := g.groupStates[group]; !registered {
		return fmt.Errorf("castrum: SetGroupEnabled: group %q is not registered", group)
	}
	g.groupStates[group] = enabled
	return nil
}

// Startup runs [core.PhasePreStartup], [core.PhaseStartup], and
// [core.PhasePostStartup] once, after resolving eager resources. Runners call
// it before entering the platform loop; a second call returns an error.
func (g *Game) Startup() error {
	if g.state.has(lifecycleStartupBegun) {
		return errors.New("castrum: Startup already ran")
	}
	g.state |= lifecycleStartupBegun
	if err := g.world.ResolveEager(); err != nil {
		return err
	}
	if err := g.run(core.PhasePreStartup); err != nil {
		return err
	}
	if err := g.run(core.PhaseStartup); err != nil {
		return err
	}
	return g.run(core.PhasePostStartup)
}

// Advance runs one frame followed by any fixed ticks due. Runners pass the
// time since the previous frame.
//
// A frame runs [core.PhasePreFrame], [core.PhaseFrame], and [core.PhasePostFrame]
// with DeltaTime capped by [WithMaxFrameTime]. The [gametime.Real] resource
// records the original elapsed time. Each fixed tick runs [core.PhasePreFixed],
// [core.PhaseFixed], and [core.PhasePostFixed]. Advance discards the remaining
// accumulated time when it reaches the limit set by [WithMaxTicksPerFrame].
//
// Pause and time scale changes made during any frame phase affect the fixed
// ticks in the same call. Changes made during fixed ticks take effect on the
// next call.
//
// If the scaled elapsed time or the resulting accumulated time cannot be
// represented as a time.Duration, Advance returns an error after the frame
// phases without changing the accumulator.
func (g *Game) Advance(elapsed time.Duration) error {
	g.real.Delta = elapsed
	g.real.Elapsed += elapsed
	if elapsed > g.real.MaxFrameTime {
		elapsed = g.real.MaxFrameTime
	}
	g.real.Frame++
	g.ctx.DeltaTime = elapsed
	if err := g.run(core.PhasePreFrame); err != nil {
		return err
	}
	if err := g.run(core.PhaseFrame); err != nil {
		return err
	}
	if err := g.run(core.PhasePostFrame); err != nil {
		return err
	}
	if g.sim.Paused {
		return nil
	}
	contribution := elapsed
	if g.sim.Scale != 1 {
		scaled := float64(elapsed) * g.sim.Scale
		if math.IsNaN(scaled) || math.IsInf(scaled, 0) ||
			scaled >= -float64(math.MinInt64) || scaled < float64(math.MinInt64) {
			return fmt.Errorf("castrum: scaled elapsed time %v is not representable", scaled)
		}
		contribution = time.Duration(scaled)
	}
	if contribution > 0 && g.acc > math.MaxInt64-contribution ||
		contribution < 0 && g.acc < math.MinInt64-contribution {
		return fmt.Errorf("castrum: simulation accumulator overflow adding %v", contribution)
	}
	g.acc += contribution
	ticked := 0
	for g.acc >= g.sim.Delta {
		g.sim.Tick++
		g.ctx.Tick = g.sim.Tick
		// Keep the world's change tick in step with the simulation tick so
		// change-detection stamps are comparable with Context.Tick.
		g.world.AdvanceTick()
		g.ctx.DeltaTime = g.sim.Delta
		if err := g.run(core.PhasePreFixed); err != nil {
			return err
		}
		if err := g.run(core.PhaseFixed); err != nil {
			return err
		}
		if err := g.run(core.PhasePostFixed); err != nil {
			return err
		}
		g.acc -= g.sim.Delta
		ticked++
		if ticked >= g.opts.MaxTicksPerFrame {
			g.acc = 0 // drop the backlog rather than spiral
			break
		}
	}
	return nil
}

// Alpha returns the fixed-loop remainder as a fraction of the tick interval
// for render interpolation.
func (g *Game) Alpha() float64 {
	return float64(g.acc) / float64(g.sim.Delta)
}

// Frame counts display frames since startup.
func (g *Game) Frame() uint64 {
	return g.real.Frame
}

// Pause suspends fixed updates and preserves the accumulated time.
//
// No time is added toward fixed ticks while paused. Frame updates and
// audio continue, and [Game.TimeScale] is unchanged.
// See [Game.Advance] for when the change takes effect.
func (g *Game) Pause() {
	g.sim.Pause()
}

// Resume re-enables fixed updates after [Game.Pause]; see [Game.Advance] for timing.
func (g *Game) Resume() {
	g.sim.Resume()
}

// Paused reports whether fixed updates are stopped by [Game.Pause].
func (g *Game) Paused() bool {
	return g.sim.Paused
}

// TimeScale returns the multiplier applied to the time accumulated toward
// fixed ticks: 0.5 is half speed, 2 is fast forward. The default is 1.
func (g *Game) TimeScale() float64 {
	return g.sim.Scale
}

// SetTimeScale sets the multiplier for elapsed time added toward fixed ticks.
//
// It panics if scale is not positive and finite. Frame timing is unchanged.
// See [Game.Advance] for when the change takes effect.
func (g *Game) SetTimeScale(scale float64) {
	if err := g.sim.SetScale(scale); err != nil {
		panic(fmt.Errorf("castrum: %w", err))
	}
}

// Context returns the live context. Runners use it when running their draw
// systems; game code receives it as a system argument.
func (g *Game) Context() *core.Context {
	return &g.ctx
}

// Quit requests termination. The active runner observes it and stops.
func (g *Game) Quit() {
	g.quit = true
}

// Quitting reports whether [Game.Quit] was called.
func (g *Game) Quitting() bool {
	return g.quit
}

// BeginRun marks the game as started by a runner and returns an error on
// subsequent calls.
func (g *Game) BeginRun() error {
	if g.state.has(lifecycleRunBegun) {
		return errors.New("castrum: BeginRun called twice")
	}
	g.state |= lifecycleRunBegun
	return nil
}

func (g *Game) run(s core.Phase) error {
	if schedule, ok := g.schedules[s]; ok {
		return schedule.Run(&g.ctx)
	}
	return nil
}
