package systems

import (
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
)

// RotatorSystem advances Transform.Rotation for every entity with a Spin
// component - a minimal example of driving gameplay behavior through systems.
type RotatorSystem struct{}

func (s *RotatorSystem) Init(world *castrum.World) error {
	return nil
}

func (s *RotatorSystem) Update(world *castrum.World, delta float64) error {
	for _, id := range castrum.QueryFor[components.Spin](world) {
		spin, err := world.GetComponent[components.Spin](id)
		if err != nil {
			continue
		}
		transform, err := world.GetComponent[components.Transform](id)
		if err != nil {
			continue
		}

		transform.Rotation += spin.AngularVelocity * delta
		if err := world.SetComponent(id, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *RotatorSystem) Shutdown(world *castrum.World) error {
	return nil
}
