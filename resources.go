package castrum

import (
	"github.com/Leonard-Atorough/castrum/core"
)

// WithResource registers value as a world resource when [New] builds the game.
// A nil value, or one whose Validate() error fails, makes [New] panic. A type
// already registered by the engine or another option also makes [New] panic;
// use [WithResourceOverride] to replace it. Resources apply after the engine's
// own resources and before features, in option order.
func WithResource[T any](value T) option {
	return optionFunc(func(o *Options) {
		o.resources = append(o.resources, func(w *core.World) error {
			return w.InsertResource(value)
		})
	})
}

// WithResourceOverride replaces the resource or provider of type T when [New]
// builds the game. It makes [New] panic if T is not already registered at that
// point in option order, or if value is nil or fails Validate().
func WithResourceOverride[T any](value T) option {
	return optionFunc(func(o *Options) {
		o.resources = append(o.resources, func(w *core.World) error {
			return w.ReplaceResource(value)
		})
	})
}

// WithResourceProvider registers ctor to build the world resource of type T when
// [New] builds the game. ctor runs on first use and may read other resources from
// the world. A type already registered makes [New] panic; use
// [WithResourceOverride] to replace it.
func WithResourceProvider[T any](ctor func(*core.World) (T, error)) option {
	return optionFunc(func(o *Options) {
		o.resources = append(o.resources, func(w *core.World) error {
			return w.Provide(ctor)
		})
	})
}

// InsertResource sets value as a world resource on an existing game. It returns an
// error without changing the world if T is already registered, or if value is
// nil or fails Validate(). Use [Game.ReplaceResource] to replace a resource.
func (g *Game) InsertResource[T any](value T) error {
	return g.world.InsertResource(value)
}

// ReplaceResource replaces the resource of type T on an existing game. It
// returns an error without changing the world if T is not registered, or if
// value is nil or fails Validate().
func (g *Game) ReplaceResource[T any](value T) error {
	return g.world.ReplaceResource(value)
}
