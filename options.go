package castrum

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/state"
)

// A feature wires a named engine capability into a game under construction.
// Features are applied in registration order when [New] builds the game.
// Registering an already-registered name is a no-op, which makes repeated
// argument-less options idempotent.
type feature struct {
	name   string
	enable func(*Game, *Options) error
}

// Options contains the configuration used to create a [Game]. Use [New] with
// option constructors to apply defaults and validate values.
//
// Window and graphics settings belong to the runner.
type Options struct {
	// Title is the title displayed by the runner.
	Title string

	// Filesystem supplies game asset files. Engine-shipped assets under
	// "builtin/" are available alongside it. If nil, game assets resolve from
	// the current working directory.
	Filesystem fs.FS

	// AssetCodecs configures the game's loader. New snapshots its registrations.
	// Nil selects [NewAssetCodecs].
	AssetCodecs *asset.Codecs

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
	// InputDeadzone is the map-wide gamepad axis threshold. Zero selects
	// [input.DefaultDeadzone].
	InputDeadzone float64

	inputPreferences bool

	fixedDT time.Duration

	features []feature
	states   []stateRegistration
}

type stateRegistration struct {
	name    string
	system  core.System
	provide func(*core.World) error
}

// enableFeature registers f under name unless a feature with that name is
// already registered, in which case it is a no-op.
func (o *Options) enableFeature(name string, enable func(*Game, *Options) error) {
	for _, existing := range o.features {
		if existing.name == name {
			return
		}
	}
	o.features = append(o.features, feature{name: name, enable: enable})
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
	if o.inputPreferences {
		if err := (input.Preferences{Bindings: o.InputBindings, Deadzone: o.InputDeadzone}).Validate(); err != nil {
			return fmt.Errorf("castrum: input preferences: %w", err)
		}
	}
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

// WithFilesystem sets the filesystem used to resolve game assets. Built-in
// assets under "builtin/" remain available.
func WithFilesystem(filesystem fs.FS) option {
	return optionFunc(func(o *Options) { o.Filesystem = filesystem })
}

// WithAssetCodecs supplies codec configuration for the game's content loader.
// New snapshots the registrations; later changes do not affect that game.
// Nil selects [NewAssetCodecs].
func WithAssetCodecs(codecs *asset.Codecs) option {
	return optionFunc(func(o *Options) { o.AssetCodecs = codecs })
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

// WithBindings configures the game's input bindings without changing its
// deadzone. Nil disables the action map.
func WithBindings(bindings input.Bindings) option {
	return optionFunc(func(o *Options) {
		o.InputBindings = bindings
		o.inputPreferences = false
	})
}

// WithInputPreferences configures the game's initial bindings and deadzone.
// [New] validates and copies the values; it does not retain a preference group.
// Later [WithBindings] options override only the bindings.
func WithInputPreferences(p input.Preferences) option {
	return optionFunc(func(o *Options) {
		o.InputBindings = p.Bindings
		o.InputDeadzone = p.Deadzone
		o.inputPreferences = true
	})
}

// WithScenes enables the scene system and provides a scene Director resource.
// Passing it more than once, or together with [WithDefaultSystems], has no
// additional effect.
func WithScenes() option {
	return optionFunc(func(o *Options) {
		o.enableFeature("scenes", func(g *Game, _ *Options) error {
			director := scene.NewDirector()
			if err := g.world.Provide(func(*core.World) (*scene.Director, error) {
				return director, nil
			}); err != nil {
				return fmt.Errorf("castrum: provide scene director: %w", err)
			}
			if err := g.addEngineFirst(core.PhaseStartup, scene.StartupSystemName, scene.NewStartupSystem(director)); err != nil {
				return err
			}
			return g.addEngineFirst(core.PhaseFrame, scene.FrameSystemName, scene.NewFrameSystem(director))
		})
	})
}

// WithState provides machine as a world resource and enables its system in
// [core.PhasePreStartup] and [core.PhasePreFrame]. Repeated options register
// machines in order. [New] returns an error for a nil machine or duplicate
// machine resource type.
func WithState[S comparable](machine *state.Machine[S]) option {
	system := state.NewSystem(machine)
	return optionFunc(func(o *Options) {
		o.states = append(o.states, stateRegistration{
			name:   fmt.Sprintf("%T", machine),
			system: system,
			provide: func(world *core.World) error {
				if machine == nil {
					return fmt.Errorf("state machine must not be nil")
				}
				return world.Provide(func(*core.World) (*state.Machine[S], error) {
					return machine, nil
				})
			},
		})
		o.enableFeature("state", func(g *Game, opts *Options) error {
			for _, registration := range opts.states {
				if err := registration.provide(g.world); err != nil {
					return fmt.Errorf("castrum: provide state machine %s: %w", registration.name, err)
				}
			}
			states := core.SystemFunc(func(ctx *core.Context) error {
				for _, registration := range opts.states {
					if err := registration.system.Update(ctx); err != nil {
						return fmt.Errorf("%s: %w", registration.name, err)
					}
				}
				return nil
			})
			if err := g.addEngineFirst(core.PhasePreStartup, state.SystemName, states); err != nil {
				return err
			}
			return g.addEngineFirst(core.PhasePreFrame, state.SystemName, states)
		})
	})
}
