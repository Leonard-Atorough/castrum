package castrum

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/timer"
)

// Game owns a game's world, configuration, and update schedules. A [Runner]
// supplies the platform loop, window, draw surface, and input.
type Game struct {
	opts        Options
	world       *core.World
	ctx         core.Context
	schedules   map[core.Phase]*runtime.Schedule[core.System]
	groupStates map[string]bool

	mainCamera *core.Entity
	hub        *signal.Hub

	acc     time.Duration
	started atomic.Bool
	startup atomic.Bool
	quit    atomic.Bool

	paused    bool
	timeScale float64
}

// New creates a [Game] with default options overridden by opts. It returns
// an error if the resulting configuration is invalid.
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
		schedules:   map[core.Phase]*runtime.Schedule[core.System]{},
		groupStates: make(map[string]bool),
		timeScale:   1,
	}
	g.ctx.World = g.world

	loader := asset.NewLoader(options.Filesystem)
	if err := g.world.Provide(func(*core.World) (*asset.Loader, error) {
		return loader, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide asset loader: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, core.PrevTransformSystemName, core.NewPrevTransformCapture()); err != nil {
		return nil, err
	}

	clips := animation.NewClipStore()
	if err := g.world.Provide(func(*core.World) (*animation.ClipStore, error) {
		return clips, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide clip store: %w", err)
	}

	hub := signal.NewHub()
	if err := g.world.Provide(func(*core.World) (*signal.Hub, error) {
		return hub, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide signal hub: %w", err)
	}
	g.hub = hub

	panel := debug.NewPanel()
	if err := g.world.Provide(func(*core.World) (*debug.Panel, error) {
		return panel, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide debug panel: %w", err)
	}

	if options.systems&sysAnimation != 0 {
		if err := g.AddSystem(core.PhaseFixed, animation.SystemName, animation.NewAnimationSystem(clips)); err != nil {
			return nil, err
		}
	}

	if options.systems&sysCollision != 0 {
		if err := g.AddSystem(core.PhaseFixed, collision.SystemName, collision.NewSystem()); err != nil {
			return nil, err
		}
	}

	if options.systems&sysTimer != 0 {
		if err := g.AddSystem(core.PhaseFixed, timer.SystemName, timer.NewSystem()); err != nil {
			return nil, err
		}
		if err := g.world.Provide(func(*core.World) (*timer.Groups, error) {
			return timer.NewGroups(), nil
		}); err != nil {
			return nil, fmt.Errorf("castrum: provide timer groups: %w", err)
		}
	}

	if options.systems&sysScenes != 0 {
		director := scene.NewDirector()
		if err := g.world.Provide(func(*core.World) (*scene.Director, error) {
			return director, nil
		}); err != nil {
			return nil, fmt.Errorf("castrum: provide scene director: %w", err)
		}
		if err := g.AddSystem(core.PhaseStartup, scene.StartupSystemName, scene.NewStartupSystem(director)); err != nil {
			return nil, err
		}
		if err := g.AddSystem(core.PhaseFrame, scene.FrameSystemName, scene.NewFrameSystem(director)); err != nil {
			return nil, err
		}
	}

	mixer := audio.NewMixer()
	if err := g.world.Provide(func(*core.World) (*audio.Mixer, error) {
		return mixer, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide audio mixer: %w", err)
	}

	// The engine's default camera yields to user-spawned primary cameras.
	camera, err := render.SpawnEngineCamera(g.world)
	if err != nil {
		return nil, fmt.Errorf("castrum: spawn main camera: %w", err)
	}
	g.mainCamera = camera
	if options.InputBindings != nil {
		if err := g.wireInput(options.InputBindings); err != nil {
			return nil, err
		}
	}
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

// AddSystem registers an ungrouped system under name in phase's schedule.
// Systems run in registration order. It returns an error if name is empty.
func (g *Game) AddSystem(phase core.Phase, name string, system core.System) error {
	return g.AddSystemToGroup(phase, "", name, system)
}

// AddSystemToGroup registers a system under name in phase's schedule,
// optionally assigning it to group. Systems run in registration order.
//
// An empty group leaves the system ungrouped. The name appears in error
// messages; an empty name returns an error. Named groups start enabled and
// retain their state when systems are registered later.
func (g *Game) AddSystemToGroup(phase core.Phase, group, name string, system core.System) error {
	if name == "" {
		return fmt.Errorf("castrum: AddSystemToGroup: system name must not be empty (phase %s)", phase)
	}
	sched, ok := g.schedules[phase]
	if !ok {
		sched = runtime.NewSchedule(phase, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		g.schedules[phase] = sched
	}
	sched.Add(runtime.Entry[core.System]{Name: name, System: system, Group: group})
	if group != "" {
		if enabled, ok := g.groupStates[group]; ok {
			sched.SetGroupEnabled(group, enabled)
		}
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
	registered := false
	for _, sched := range g.schedules {
		if sched.Registered(group) {
			registered = true
			break
		}
	}
	if !registered {
		return fmt.Errorf("castrum: SetGroupEnabled: group %q is not registered", group)
	}
	g.groupStates[group] = enabled
	for _, sched := range g.schedules {
		sched.SetGroupEnabled(group, enabled)
	}
	return nil
}

// Startup runs the [core.PhaseStartup] systems once. Runners call it before
// entering the platform loop; a second call returns an error.
func (g *Game) Startup() error {
	if !g.startup.CompareAndSwap(false, true) {
		return errors.New("castrum: Startup already ran")
	}
	if err := g.world.ResolveEager(); err != nil {
		return err
	}
	return g.run(core.PhaseStartup)
}

// Advance runs one display frame: [core.PhaseFrame], then every fixed tick due
// for elapsed, clamped to MaxFrameTime. Runners call it once per platform
// frame with the time since the previous call.
//
// While the game is paused (see [Game.Pause]), the frame phase still runs
// and DeltaTime still reports unscaled wall-clock elapsed time, but no
// time is accumulated and no fixed ticks run. Time scaling (see
// [Game.SetTimeScale]) multiplies only the elapsed time contributed to the
// fixed-loop accumulator; DeltaTime and the frame phase are unaffected.
// Both controls take effect on the next call.
func (g *Game) Advance(elapsed time.Duration) error {
	if elapsed > g.opts.MaxFrameTime {
		elapsed = g.opts.MaxFrameTime
	}
	g.ctx.Frame++
	g.ctx.DeltaTime = elapsed
	g.hub.BeginFrame()
	if err := g.run(core.PhaseFrame); err != nil {
		return err
	}
	g.hub.EndFrame()
	if g.paused {
		return nil
	}
	g.acc += time.Duration(float64(elapsed) * g.timeScale)
	ticked := 0
	for g.acc >= g.opts.fixedDT {
		g.ctx.Tick++
		g.ctx.DeltaTime = g.opts.fixedDT
		if err := g.run(core.PhaseFixed); err != nil {
			return err
		}
		g.hub.EndTick()
		g.acc -= g.opts.fixedDT
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
	return float64(g.acc) / float64(g.opts.fixedDT)
}

// Pause stops fixed updates and stops adding time to the fixed-loop
// accumulator. Frame updates still run, so input and rendering can continue.
// The change takes effect on the next [Game.Advance] call.
//
// Pausing does not change [Game.TimeScale] or discard the accumulator.
// [Game.Resume] continues from where the game stopped. Audio is not paused
// automatically.
func (g *Game) Pause() {
	g.paused = true
}

// Resume clears the paused state. The next [Game.Advance] continues fixed
// updates from the accumulator position preserved by [Game.Pause].
func (g *Game) Resume() {
	g.paused = false
}

// Paused reports whether fixed updates are stopped by [Game.Pause].
func (g *Game) Paused() bool {
	return g.paused
}

// TimeScale returns the multiplier applied to the time accumulated toward
// fixed ticks: 0.5 is half speed, 2 is fast forward. The default is 1.
func (g *Game) TimeScale() float64 {
	return g.timeScale
}

// SetTimeScale sets the multiplier applied to the time accumulated toward
// fixed ticks. It returns an error for a scale of zero or less; use
// [Game.Pause] to stop fixed updates while preserving the accumulator.
//
// The scale multiplies only the accumulator's time contribution; the frame
// phase and [core.Context.DeltaTime] during frame updates are unaffected.
// The new scale takes effect on the next [Game.Advance] call.
func (g *Game) SetTimeScale(scale float64) error {
	if scale <= 0 {
		return fmt.Errorf("castrum: time scale %v, want > 0; use Pause to stop the simulation", scale)
	}
	g.timeScale = scale
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
