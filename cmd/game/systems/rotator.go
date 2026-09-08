package systems

import (
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
)

// RotatorSystem advances Transform.Rotation for every entity with a Spin
// component - a minimal example of driving gameplay behavior through systems.
type RotatorSystem struct {
	spinQuery *castrum.Query
}

func (s *RotatorSystem) Init(world *castrum.World) error {
	s.spinQuery = world.NewQuery().WithRequiredComponents(components.Spin{})
	return nil
}

func (s *RotatorSystem) Update(world *castrum.World, delta float64) error {
	for result := range s.spinQuery.Execute() {
		spin, err := world.GetComponent[components.Spin](result.EntityID)
		if err != nil {
			continue
		}
		transform, err := world.GetComponent[components.Transform](result.EntityID)
		if err != nil {
			continue
		}

		transform.Rotation += spin.AngularVelocity * delta
		if err := world.SetComponent(result.EntityID, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *RotatorSystem) Shutdown(world *castrum.World) error {
	return nil
}
