package castrum

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/internal/runtime"
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

	// Simulation contract: developer decisions, not player preferences.
	FixedTPS         int
	MaxFrameTime     time.Duration
	MaxTicksPerFrame int

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
	return g, nil
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
