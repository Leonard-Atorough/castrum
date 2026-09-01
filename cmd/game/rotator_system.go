package main

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

// RotatorSystem advances Transform.Rotation for every entity with a Spin
// component - a minimal example of driving gameplay behavior through systems.
type RotatorSystem struct{}

func (s *RotatorSystem) Init(world *core.World) error {
	return nil
}

func (s *RotatorSystem) Update(world *core.World, delta float64) error {
	for _, id := range core.QueryFor[components.Spin](world) {
		spin, err := core.GetComponent[components.Spin](world, id)
		if err != nil {
			continue
		}
		transform, err := core.GetComponent[components.Transform](world, id)
		if err != nil {
			continue
		}

		transform.Rotation += spin.AngularVelocity * delta
		if err := core.SetComponent(world, id, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *RotatorSystem) Shutdown(world *core.World) error {
	return nil
}
