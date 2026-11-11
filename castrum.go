package castrum

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/gametime"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

// Game owns a game's world, configuration, and update schedules. A [Runner]
// supplies the platform loop, window, draw surface, and input.
type Game struct {
	opts        Options
	world       *core.World
	ctx         core.Context
	schedules   map[core.Phase]*runtime.Schedule
	groupStates map[string]bool

	mainCamera *core.Entity
	hub        *signal.Hub

	acc     time.Duration
	started atomic.Bool
	startup atomic.Bool
	quit    atomic.Bool

	sim  *gametime.Simulation
	real *gametime.Real
}

// New creates a [Game] with default options overridden by opts. It returns
// an error if the resulting configuration is invalid.
//
// The asset loader resolves engine-shipped assets under "builtin/" alongside
// the configured filesystem.
func New(opts ...option) (*Game, error) {
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

	sim := gametime.NewSimulation(options.fixedDT)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Simulation, error) {
		return sim, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide simulation time: %w", err)
	}
	g.sim = sim

	real := gametime.NewReal(options.MaxFrameTime)
	if err := g.world.ProvideEager(func(*core.World) (*gametime.Real, error) {
		return real, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide real time: %w", err)
	}
	g.real = real

	loader := asset.NewLoader(builtin.WithFilesystem(options.Filesystem))
	if err := g.world.Provide(func(*core.World) (*asset.Loader, error) {
		return loader, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide asset loader: %w", err)
	}

	if err := g.addEngineFirst(core.PhasePreFixed, core.PrevTransformSystemName, core.NewPrevTransformCapture()); err != nil {
		return nil, err
	}

	// Declaration-driven systems: their work is declared by component
	// presence, so the component is the registration. With no matching
	// entities they cost an empty query per tick. Their resources are
	// provided here so the systems resolve them on first update.
	clips := animation.NewClipStore()
	if err := g.world.Provide(func(*core.World) (*animation.ClipStore, error) {
		return clips, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide clip store: %w", err)
	}
	if err := g.world.Provide(func(*core.World) (*gametime.Groups, error) {
		return gametime.NewGroups(), nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide timer groups: %w", err)
	}
	if err := g.addEngineFirst(core.PhaseFixed, gametime.TimerSystemName, gametime.NewTimerSystem()); err != nil {
		return nil, err
	}
	if err := g.addEngineFirst(core.PhaseFixed, animation.SystemName, animation.NewAnimationSystem(clips)); err != nil {
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
		System: core.SystemFunc(func(*core.Context) error {
			hub.BeginFrame()
			return nil
		}),
	}); err != nil {
		return nil, fmt.Errorf("castrum: register signal begin-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFrame).SetClosing(runtime.Entry{
		Name: "engine.signal-end-frame",
		System: core.SystemFunc(func(*core.Context) error {
			hub.EndFrame()
			return nil
		}),
	}); err != nil {
		return nil, fmt.Errorf("castrum: register signal end-frame system: %w", err)
	}
	if err := g.schedule(core.PhasePostFixed).SetClosing(runtime.Entry{
		Name: "engine.signal-end-tick",
		System: core.SystemFunc(func(*core.Context) error {
			hub.EndTick()
			return nil
		}),
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
	var reconciler core.System
	// Capture before other frame systems so mid-phase attachments wait for the next pass.
	if err := g.addEngineFirst(core.PhaseFrame, "engine.audio-prepare", core.SystemFunc(func(*core.Context) error {
		if controller := backend.Controller(); reconciler == nil && controller != nil {
			reconciler = audio.NewAudioSystem(controller, mixer)
		}
		return nil
	})); err != nil {
		return nil, err
	}
	if err := g.schedule(core.PhaseFrame).AddLast(runtime.Entry{
		Name: audio.SystemName,
		System: core.SystemFunc(func(ctx *core.Context) error {
			if reconciler == nil {
				return nil
			}
			return reconciler.Update(ctx)
		}),
	}); err != nil {
		return nil, fmt.Errorf("castrum: register audio system: %w", err)
	}

	if options.InputBindings != nil {
		if err := g.wireInput(options.InputBindings); err != nil {
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

// Options returns a copy of the game's configuration. Runners use it to read
// the game title.
func (g *Game) Options() Options {
	return g.opts
}

// World returns the game's world: entities and resources live there.
func (g *Game) World() *core.World {
	return g.world
}

// MainCamera returns the default camera created by [New], positioned at the
// world origin with zoom 1. Set its [core.Transform] to move it. A user-spawned
// primary camera takes precedence.
func (g *Game) MainCamera() *core.Entity {
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
// previously registered user systems. [Before], [After], and [InGroup] configure
// its position and group.
//
// Names must be nonempty and unique within the phase. Relative targets must
// already be registered user systems in the same phase. At most one
// position and one group may be specified. Invalid phases, options, names,
// and targets return an error. Startup phases reject registration once
// [Game.Startup] begins.
//
// Systems added during a phase run are eligible on the next run of that
// phase. Group changes affect systems that have not yet run. Registration
// and group changes must not run concurrently with each other or with
// schedule execution.
func (g *Game) AddSystem(phase core.Phase, name string, system core.System, options ...SystemOption) error {
	if phase < core.PhasePreStartup || phase > core.PhasePostFixed {
		return fmt.Errorf("castrum: AddSystem: invalid phase %s", phase)
	}
	if phase <= core.PhasePostStartup && g.startup.Load() {
		return fmt.Errorf("castrum: AddSystem: startup has already begun (phase %s)", phase)
	}
	if name == "" {
		return fmt.Errorf("castrum: AddSystem: system name must not be empty (phase %s)", phase)
	}
	var position systemOptionKind
	var target, group string
	groupSet := false
	for _, option := range options {
		switch option.kind {
		case systemOptionBefore, systemOptionAfter:
			if position != 0 {
				return fmt.Errorf("castrum: AddSystem: multiple position options")
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

func (g *Game) addEngineFirst(phase core.Phase, name string, system core.System) error {
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
	if !g.startup.CompareAndSwap(false, true) {
		return errors.New("castrum: Startup already ran")
	}
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

// Advance runs [core.PhasePreFrame], [core.PhaseFrame], [core.PhasePostFrame],
// then every fixed tick due for elapsed, clamped to MaxFrameTime. Each tick
// runs [core.PhasePreFixed], [core.PhaseFixed], and [core.PhasePostFixed].
// Runners call Advance once per platform frame with the time since the
// previous call.
//
// While the game is paused (see [Game.Pause]), the frame phase still runs
// and DeltaTime still reports unscaled wall-clock elapsed time, but no
// time is accumulated and no fixed ticks run. Time scaling (see
// [Game.SetTimeScale]) multiplies only the elapsed time contributed to the
// fixed-loop accumulator; DeltaTime and the frame phase are unaffected.
// Both controls take effect on the next call.
// An unrepresentable scaled contribution or accumulator sum returns an error
// without changing the accumulator; frame systems have already run.
func (g *Game) Advance(elapsed time.Duration) error {
	g.real.Delta = elapsed
	g.real.Elapsed += elapsed
	if elapsed > g.real.MaxFrameTime {
		elapsed = g.real.MaxFrameTime
	}
	g.real.Frame++
	g.ctx.Frame = g.real.Frame
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

// Alpha returns the fixed-loop remainder over the tick interval, in [0, 1).
//
// While the game is paused, the accumulator holds its value, so Alpha stays
// fixed; renderers keep the last interpolated position instead of jumping.
func (g *Game) Alpha() float64 {
	return float64(g.acc) / float64(g.sim.Delta)
}

// Pause stops fixed updates and stops adding time to the fixed-loop
// accumulator. Frame updates still run, so input and rendering can continue.
// The change takes effect on the next [Game.Advance] call.
//
// Pausing does not change [Game.TimeScale] or discard the accumulator.
// [Game.Resume] continues from where the game stopped. Audio is not paused
// automatically.
func (g *Game) Pause() {
	g.sim.Pause()
}

// Resume clears the paused state. The next [Game.Advance] continues fixed
// updates from the accumulator position preserved by [Game.Pause].
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

// SetTimeScale sets the multiplier applied to the time accumulated toward
// fixed ticks. It returns an error for a scale of zero or less; use
// [Game.Pause] to stop fixed updates while preserving the accumulator.
//
// The scale multiplies only the accumulator's time contribution; the frame
// phase and [core.Context.DeltaTime] during frame updates are unaffected.
// The new scale takes effect on the next [Game.Advance] call.
func (g *Game) SetTimeScale(scale float64) error {
	if err := g.sim.SetScale(scale); err != nil {
		return fmt.Errorf("castrum: %w", err)
	}
	return nil
}

// Context returns the live context. Runners use it when running their draw
// systems; game code receives it as a system argument.
func (g *Game) Context() *core.Context {
	return &g.ctx
}

// Quit requests termination. The active runner observes it and stops.
func (g *Game) Quit() {
	g.quit.Store(true)
}

// Quitting reports whether Quit was called.
func (g *Game) Quitting() bool {
	return g.quit.Load()
}

// Runner drives a Game: it owns the platform loop, the draw surface, and
// input, and calls the Game loop hooks. Run blocks until the game quits.
type Runner interface {
	Run() error
}

// Run hands control to r. It may be called once; a second call returns an
// error.
func (g *Game) Run(r Runner) error {
	if !g.started.CompareAndSwap(false, true) {
		return errors.New("castrum: Run called twice")
	}
	return r.Run()
}

func (g *Game) run(s core.Phase) error {
	if schedule, ok := g.schedules[s]; ok {
		return schedule.Run(&g.ctx)
	}
	return nil
}
