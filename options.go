package castrum

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
)

// Bit flags make repeated system options idempotent.
const (
	sysAnimation = 1 << iota
	sysCollision
	sysTimer
	sysScenes
)

// Options contains the configuration used to create a [Game]. Use [New] with
// option constructors to apply defaults and validate values.
//
// Window and graphics settings belong to the runner.
type Options struct {
	// Title is the title displayed by the runner.
	Title string

	// Filesystem supplies the asset files resolved by the game. If nil, the
	// game uses the current working directory.
	Filesystem fs.FS

	// FixedTPS is the number of fixed simulation ticks per second.
	FixedTPS int

	// MaxFrameTime is the maximum elapsed time accepted for one frame.
	MaxFrameTime time.Duration

	// MaxTicksPerFrame is the maximum number of fixed ticks run in one display
	// frame.
	MaxTicksPerFrame int

	// InputBindings configures the engine-owned [input.ActionMap]. If nil, no
	// action map is created, but the runner's raw input snapshot remains
	// available.
	InputBindings input.Bindings

	fixedDT time.Duration

	systems uint8
}

// FixedDT returns the interval between fixed simulation ticks. [New] derives
// it from [Options.FixedTPS].
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

// WithTitle sets the game's title.
func WithTitle(title string) option {
	return optionFunc(func(o *Options) { o.Title = title })
}

// WithFilesystem sets the filesystem used to resolve assets.
func WithFilesystem(filesystem fs.FS) option {
	return optionFunc(func(o *Options) { o.Filesystem = filesystem })
}

// WithFixedTPS sets the fixed simulation rate in ticks per second.
// The default is 60. [New] returns an error if the rate is invalid.
func WithFixedTPS(tps int) option {
	return optionFunc(func(o *Options) { o.FixedTPS = tps })
}

// WithMaxFrameTime sets the maximum elapsed time accepted for one frame.
// The default is 250 ms. [New] returns an error if d is not positive.
func WithMaxFrameTime(d time.Duration) option {
	return optionFunc(func(o *Options) { o.MaxFrameTime = d })
}

// WithMaxTicksPerFrame sets the maximum number of fixed ticks run in one
// display frame. The default is 5. [New] returns an error if n is not
// positive.
func WithMaxTicksPerFrame(n int) option {
	return optionFunc(func(o *Options) { o.MaxTicksPerFrame = n })
}

// WithBindings configures the game's input bindings.
func WithBindings(bindings input.Bindings) option {
	return optionFunc(func(o *Options) { o.InputBindings = bindings })
}

// WithCollision enables collision processing during fixed updates. Without
// it, colliders do not produce contacts. Passing it more than once, or with
// [WithDefaultSystems], has no additional effect.
func WithCollision() option {
	return optionFunc(func(o *Options) { o.systems |= sysCollision })
}

// WithTimer enables timer updates during fixed updates. Without it, timer
// components do not advance.
func WithTimer() option {
	return optionFunc(func(o *Options) { o.systems |= sysTimer })
}

// WithAnimation enables animation updates during fixed updates. Without it,
// animated sprites do not advance.
func WithAnimation() option {
	return optionFunc(func(o *Options) { o.systems |= sysAnimation })
}

// WithScenes enables the scene system and provides a scene Director resource.
func WithScenes() option {
	return optionFunc(func(o *Options) { o.systems |= sysScenes })
}

// WithDefaultSystems enables animation, collision, and timer processing. It
// is equivalent to passing [WithAnimation], [WithCollision], and [WithTimer].
func WithDefaultSystems() option {
	return optionFunc(func(o *Options) { o.systems |= sysAnimation | sysCollision | sysTimer })
}
