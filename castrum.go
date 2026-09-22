package castrum

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
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

// Schedule identifies when a system runs.
type Schedule int8

const (
	// ScheduleStartup runs once, before the first frame.
	ScheduleStartup Schedule = iota
	// ScheduleFrame runs once per display frame, before fixed ticks.
	// Timers and input consumption belong here.
	ScheduleFrame
	// ScheduleFixed runs at the fixed simulation rate. Movement and
	// physics belong here.
	ScheduleFixed
)

//Note: could try using stringer here
func (s Schedule) String() string {
	switch s {
	case ScheduleStartup:
		return "startup"
	case ScheduleFrame:
		return "frame"
	case ScheduleFixed:
		return "fixed"
	default:
		return fmt.Sprintf("schedule(%d)", int8(s))
	}
}

// Context carries per-frame state to systems. It is reused across calls;
// systems must not retain it.
type Context struct {
	// Tick counts fixed simulation steps since startup.
	Tick uint64
	// Frame counts display frames since startup.
	Frame uint64
	// DeltaTime is the current schedule's step: elapsed frame time in
	// ScheduleFrame, the fixed tick interval in ScheduleFixed.
	DeltaTime time.Duration
	// Alpha is the fixed-loop remainder over the tick interval, for
	// interpolating between the last two ticks. Only a runner sets it,
	// when running its draw systems.
	Alpha float64
}

// System is a unit of game logic bound to a Schedule. State persists via
// closures; there is no system interface.
type System func(ctx *Context) error

// NOTE: This is bound to change. System will be an interface with three methods:
// - Startup(world *World) error
// - Frame(world *World, ctx *Context) error
// - Fixed(world *World, ctx *Context) error

// Game is the core engine: configuration, schedules, and the fixed loop.
// It is runner-agnostic: a [Runner] drives it and owns the window, the draw
// surface, and input.
type Game struct {
	opts      Options
	schedules map[Schedule][]System
	ctx       Context

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

// New creates a Game from defaults overridden by opts. Invalid values
// panic: registration is a build-time error by design, and the stack points
// at the offending constructor.
func New(opts ...option) *Game {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	options.finalize()
	return &Game{schedules: map[Schedule][]System{}}
}

// finalize derives dependent values and enforces cross-field invariants.
// Per-field validation lives in the option constructors; this is the
// single owner of derived state.
func (o *Options) finalize() {
	o.fixedDT = time.Second / time.Duration(o.FixedTPS)
	if o.fixedDT <= 0 {
		panic(fmt.Sprintf("castrum: FixedTPS %d yields a non-representable tick interval", o.FixedTPS))
	}
}

// Options returns a copy of the converged configuration. The runner reads
// game identity (title) from it.
func (g *Game) Options() Options {
	return g.opts
}

// AddSystem binds systems to a schedule. Systems run in registration order.
func (g *Game) AddSystem(s Schedule, systems ...System) {
	g.schedules[s] = append(g.schedules[s], systems...)
}

// Startup runs the ScheduleStartup systems once. Runners call it before
// entering the platform loop; a second call returns an error.
func (g *Game) Startup() error {
	if !g.startup.CompareAndSwap(false, true) {
		return errors.New("castrum: Startup already ran")
	}
	return g.run(ScheduleStartup)
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
	if err := g.run(ScheduleFrame); err != nil {
		return err
	}
	g.acc += elapsed
	ticked := 0
	for g.acc >= g.opts.fixedDT {
		g.ctx.Tick++
		g.ctx.DeltaTime = g.opts.fixedDT
		if err := g.run(ScheduleFixed); err != nil {
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
func (g *Game) Context() *Context {
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

func (g *Game) run(s Schedule) error {
	for _, sys := range g.schedules[s] {
		if err := sys(&g.ctx); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}

// WithTitle sets the game title. The active runner displays it.
func WithTitle(title string) option {
	return optionFunc(func(o *Options) { o.Title = title })
}

// WithFixedTPS sets the fixed simulation rate in ticks per second.
// Panics if not positive. Default is 60.
func WithFixedTPS(tps int) option {
	return optionFunc(func(o *Options) {
		if tps <= 0 {
			panic(fmt.Sprintf("castrum: WithFixedTPS: got %d, want > 0", tps))
		}
		o.FixedTPS = tps
	})
}

// WithMaxFrameTime sets the per-frame accumulator clamp, the
// spiral-of-death guard. Default is 250ms.
func WithMaxFrameTime(d time.Duration) option {
	return optionFunc(func(o *Options) {
		if d <= 0 {
			panic(fmt.Sprintf("castrum: WithMaxFrameTime: got %v, want > 0", d))
		}
		o.MaxFrameTime = d
	})
}

// WithMaxTicksPerFrame sets how many fixed ticks may run in one display
// frame before excess accumulated time is dropped. Default is 5.
func WithMaxTicksPerFrame(n int) option {
	return optionFunc(func(o *Options) {
		if n <= 0 {
			panic(fmt.Sprintf("castrum: WithMaxTicksPerFrame: got %d, want > 0", n))
		}
		o.MaxTicksPerFrame = n
	})
}
