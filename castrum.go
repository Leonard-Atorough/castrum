package castrum

import (
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"
	"time"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
	"github.com/Leonard-Atorough/castrum/timer"
)

// Options is the pure-data configuration [New] converges option values into.
// It carries only data, never callbacks or handles, so a future file-backed
// Config type can map onto it 1:1. Populate it through [New] and the [With*]
// constructors; direct construction skips defaults and validation.
//
// Window and graphics settings do not live here: they belong to the runner.
type Options struct {
	// Title is the game's identity; the runner displays it.
	Title string

	// Filesystem is the fs.FS asset paths resolve against: embed.FS
	// for single-binary distribution, os.DirFS for development
	// layouts. nil - the zero value - is the game's working
	// directory.
	Filesystem fs.FS

	// Simulation contract: developer decisions, not player preferences.
	FixedTPS         int
	MaxFrameTime     time.Duration
	MaxTicksPerFrame int

	// InputBindings is the mapping from user actions to physical inputs.
	// It provides an engine-native way to map raw inputs to high-level actions.
	//
	// When provided, the bindings can be accessed on the Context through [core.Context.Actions].
	//
	// If nil, no engine-owned ActionMap is created, and input must be polled manually through [core.Context.Input].
	InputBindings input.Bindings

	// fixedDT is derived from FixedTPS in New; no option sets it.
	fixedDT time.Duration
}

// FixedDT returns the derived interval between fixed ticks.
func (o Options) FixedDT() time.Duration { return o.fixedDT }

// Game is the core engine: configuration, schedules, and the fixed loop.
// It is runner-agnostic: a [Runner] drives it and owns the window, the draw
// surface, and input.
type Game struct {
	opts      Options
	world     *core.World
	ctx       core.Context
	schedules map[core.Phase]*runtime.Schedule[core.System]

	mainCamera  *core.Entity
	assetServer *asset.Server
	clips       *animation.ClipStore
	mixer       *audio.Mixer

	acc     time.Duration
	started atomic.Bool
	startup atomic.Bool
	quit    atomic.Bool
}

type option interface {
	apply(*Options)
}

type optionFunc func(*Options)

func (f optionFunc) apply(opts *Options) {
	f(opts)
}

func defaultOptions() Options {
	return Options{
		Title:            "castrum",
		FixedTPS:         60,
		MaxFrameTime:     250 * time.Millisecond,
		MaxTicksPerFrame: 5,
	}
}

// New creates a Game from defaults overridden by opts. The option
// constructors never fail; all validation happens here in a single pass,
// and New returns an error for invalid values, leaving no half-built
// game behind.
func New(opts ...option) (*Game, error) {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.finalize(); err != nil {
		return nil, err
	}
	g := &Game{
		opts:      options,
		world:     core.NewWorld(),
		schedules: map[core.Phase]*runtime.Schedule[core.System]{},
	}
	g.ctx.World = g.world

	g.assetServer = asset.New(options.Filesystem)
	if err := g.world.Provide(func(*core.World) (*asset.Server, error) {
		return g.assetServer, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide asset server: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, "engine.prev-transform", core.NewPrevTransformCapture()); err != nil {
		return nil, err
	}

	clips := animation.NewClipStore()
	g.clips = clips
	if err := g.world.Provide(func(*core.World) (*animation.ClipStore, error) {
		return clips, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide clip store: %w", err)
	}

	// The advancer costs an empty query in games without Animation
	// entities.
	if err := g.AddSystem(core.PhaseFixed, "engine.animation", animation.NewAnimationSystem(clips)); err != nil {
		return nil, err
	}

	if err := g.AddSystem(core.PhaseFixed, "engine.collision", collision.NewSystem()); err != nil {
		return nil, err
	}

	if err := g.AddSystem(core.PhaseFixed, "engine.timer", timer.NewSystem()); err != nil {
		return nil, err
	}

	mixer := audio.NewMixer()
	g.mixer = mixer
	if err := g.world.Provide(func(*core.World) (*audio.Mixer, error) {
		return mixer, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum: provide audio mixer: %w", err)
	}

	// The engine.s default camera: every game gets a working viewport without wiring.
	// It yields to any user-spawned primary - the collector prefers user cameras.
	camera, err := core.SpawnEngineCamera(g.world)
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

// wireInput resolves input bindings into an engine-owned action map,
// provides it as a resource in the world, and schedules its update and
// tick systems.
func (g *Game) wireInput(bindings input.Bindings) error {
	am, err := input.New(bindings, 0)
	if err != nil {
		return fmt.Errorf("castrum: input bindings: %w", err)
	}

	if err := g.world.Provide(func(*core.World) (*input.ActionMap, error) {
		return am, nil
	}); err != nil {
		return fmt.Errorf("castrum: provide action map: %w", err)
	}

	if err := g.AddSystem(core.PhaseFrame, "engine.input-update", newInputUpdateSystem(am)); err != nil {
		return err
	}

	g.Context().Actions = am
	return g.AddSystem(core.PhaseFixed, "engine.input-tick", newInputTickSystem(am))
}

func (o *Options) finalize() error {
	if o.FixedTPS <= 0 {
		return fmt.Errorf("castrum: FixedTPS = %d, want > 0", o.FixedTPS)
	}
	if o.MaxFrameTime <= 0 {
		return fmt.Errorf("castrum: MaxFrameTime = %v, want > 0", o.MaxFrameTime)
	}
	if o.MaxTicksPerFrame <= 0 {
		return fmt.Errorf("castrum: MaxTicksPerFrame = %d, want > 0", o.MaxTicksPerFrame)
	}
	o.fixedDT = time.Second / time.Duration(o.FixedTPS)
	if o.fixedDT <= 0 {
		return fmt.Errorf("castrum: FixedTPS %d yields a non-representable tick interval", o.FixedTPS)
	}
	return nil
}

// Options returns a copy of the converged configuration. The runner reads
// game identity (title) from it.
func (g *Game) Options() Options {
	return g.opts
}

// World returns the game's world: entities and resources live there.
func (g *Game) World() *core.World {
	return g.world
}

// MainCamera returns the engine.s default camera entity: zoom 1 at
// the world origin, spawned by New. Move it by writing its
// Transform. Spawning your own Camera with Primary set replaces the
// view entirely - the collector prefers user-spawned primaries.
func (g *Game) MainCamera() *core.Entity {
	return g.mainCamera
}

// AssetServer returns the game's asset server: the engine provides it
// at New over the configured filesystem, so atlas registration works
// before any runner exists. Systems access the same server through
// the world's resource locator.
func (g *Game) AssetServer() *asset.Server {
	return g.assetServer
}

// Clips returns the game's animation clip store: the engine provides
// it at New, so games Add clips at setup time. Systems access the
// same store through the world's resource locator.
func (g *Game) Clips() *animation.ClipStore {
	return g.clips
}

// Mixer returns the game's audio mixer: the engine provides it at New.
// Volume levels and the global pause are set through it; a runner
// applies them to its players. Systems access the same mixer through
// the world's resource locator.
func (g *Game) Mixer() *audio.Mixer {
	return g.mixer
}

// AddSystem binds systems to a schedule under a name. Systems run in
// registration order. The name identifies the registration in error
// messages and will serve as the handle for future ordering constraints;
// Returns an error if the name is empty.
func (g *Game) AddSystem(phase core.Phase, name string, system core.System) error {
	if name == "" {
		return fmt.Errorf("castrum: AddSystem: system name must not be empty (phase %s)", phase)
	}
	sched, ok := g.schedules[phase]
	if !ok {
		sched = runtime.NewSchedule(phase, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		g.schedules[phase] = sched
	}
	sched.Add(runtime.Entry[core.System]{Name: name, System: system})

	return nil
}

// Startup runs the ScheduleStartup systems once. Runners call it before
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

// Advance runs one display frame: ScheduleFrame, then every fixed tick due
// for elapsed, clamped to MaxFrameTime. Runners call it once per platform
// frame with the time since the previous call.
func (g *Game) Advance(elapsed time.Duration) error {
	if elapsed > g.opts.MaxFrameTime {
		elapsed = g.opts.MaxFrameTime
	}
	g.ctx.Frame++
	g.ctx.DeltaTime = elapsed
	if err := g.run(core.PhaseFrame); err != nil {
		return err
	}
	g.acc += elapsed
	ticked := 0
	for g.acc >= g.opts.fixedDT {
		g.ctx.Tick++
		g.ctx.DeltaTime = g.opts.fixedDT
		if err := g.run(core.PhaseFixed); err != nil {
			return err
		}
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
func (g *Game) Alpha() float64 {
	return float64(g.acc) / float64(g.opts.fixedDT)
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

// WithTitle sets the game title. The active runner displays it.
func WithTitle(title string) option {
	return optionFunc(func(o *Options) { o.Title = title })
}

// WithFilesystem sets the fs.FS asset paths resolve against. The
// default is the game's working directory.
func WithFilesystem(filesystem fs.FS) option {
	return optionFunc(func(o *Options) { o.Filesystem = filesystem })
}

// WithFixedTPS sets the fixed simulation rate in ticks per second.
// Default is 60. Invalid values are reported by [New].
func WithFixedTPS(tps int) option {
	return optionFunc(func(o *Options) { o.FixedTPS = tps })
}

// WithMaxFrameTime sets the per-frame accumulator clamp, the
// spiral-of-death guard. Default is 250ms. Invalid values are reported
// by [New].
func WithMaxFrameTime(d time.Duration) option {
	return optionFunc(func(o *Options) { o.MaxFrameTime = d })
}

// WithMaxTicksPerFrame sets how many fixed ticks may run in one display
// frame before excess accumulated time is dropped. Default is 5.
// Invalid values are reported by [New].
func WithMaxTicksPerFrame(n int) option {
	return optionFunc(func(o *Options) { o.MaxTicksPerFrame = n })
}

// WithBindings sets the input bindings for the game. These bindings are
// used to create an engine-owned ActionMap, which is provided as a resource
// in the world and drives input handling automatically.
func WithBindings(bindings input.Bindings) option {
	return optionFunc(func(o *Options) { o.InputBindings = bindings })
}
