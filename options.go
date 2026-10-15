package castrum

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
)

// Optional engine systems, tracked in Options.systems so the With*
// options enable each at most once.
const (
	sysAnimation = 1 << iota
	sysCollision
	sysTimer
)

// Options is the pure-data configuration [New] converges option values into.
// It carries only data, never callbacks or handles, so a future file-backed
// Config type can map onto it 1:1. Populate it through [New] and the option
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

	// FixedTPS is the fixed simulation rate in ticks per second.
	FixedTPS int
	// MaxFrameTime limits the elapsed time accumulated in one frame.
	MaxFrameTime time.Duration
	// MaxTicksPerFrame limits catch-up ticks before the backlog is dropped.
	MaxTicksPerFrame int

	// InputBindings configures the engine-owned [input.ActionMap].
	// If nil, no action map is created; the runner's raw input snapshot
	// remains available.
	InputBindings input.Bindings

	// fixedDT is derived from FixedTPS in New; no option sets it.
	fixedDT time.Duration

	// systems records which optional engine systems the With* options
	// enabled, so New registers each at most once.
	systems uint8
}

// FixedDT returns the derived interval between fixed ticks.
func (o Options) FixedDT() time.Duration { return o.fixedDT }

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
// used to create an engine-owned [input.ActionMap], which is provided as a
// resource in the world and drives input handling automatically.
func WithBindings(bindings input.Bindings) option {
	return optionFunc(func(o *Options) { o.InputBindings = bindings })
}

// WithCollision registers the engine's collision system in the fixed
// schedule, under [github.com/Leonard-Atorough/castrum/collision.SystemName].
// Without it, colliders never produce
// [github.com/Leonard-Atorough/castrum/collision.Contacts].
// Idempotent: passing it twice, or together with [WithDefaultSystems],
// registers the system once.
func WithCollision() option {
	return optionFunc(func(o *Options) { o.systems |= sysCollision })
}

// WithTimer registers the engine's timer system in the fixed schedule,
// under [github.com/Leonard-Atorough/castrum/timer.SystemName]. Without it,
// [github.com/Leonard-Atorough/castrum/timer.Timer] components never advance.
// Idempotent.
func WithTimer() option {
	return optionFunc(func(o *Options) { o.systems |= sysTimer })
}

// WithAnimation registers the engine's animation system in the fixed
// schedule, under [github.com/Leonard-Atorough/castrum/animation.SystemName].
// Without it, animations never advance and sprites keep their current
// drawable. Idempotent.
func WithAnimation() option {
	return optionFunc(func(o *Options) { o.systems |= sysAnimation })
}

// WithDefaultSystems registers all of the engine's optional systems:
// animation, collision, and timer. It is equivalent to passing
// [WithAnimation], [WithCollision], and [WithTimer] together.
func WithDefaultSystems() option {
	return optionFunc(func(o *Options) { o.systems |= sysAnimation | sysCollision | sysTimer })
}
