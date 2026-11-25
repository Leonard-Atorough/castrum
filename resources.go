package castrum

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// WithResource registers value as a world resource when [New] builds the game.
// If the resource type implements Validate() error, the value is validated and
// [New] panics on failure. Resources apply after the engine's own resources
// and before features, in option order; a later WithResource for the same type
// replaces the earlier one.
func WithResource[T any](value T) option {
	return optionFunc(func(o *Options) {
		o.resources = append(o.resources, func(w *core.World) error {
			return registerResource(w, value)
		})
	})
}

// AddResource registers value as a world resource on an existing game,
// replacing any resource of the same type. It returns an error without
// changing the world if the type implements Validate() error and the value
// is invalid.
func (g *Game) AddResource[T any](value T) error {
	return registerResource(g.world, value)
}

func registerResource[T any](w *core.World, value T) error {
	if v, ok := any(value).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("castrum: resource %T: %w", value, err)
		}
	}
	w.SetResource(value)
	return nil
}
