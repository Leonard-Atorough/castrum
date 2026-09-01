package systems

import (
	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
)

// MovementSystem applies Velocity to Transform.Position for every entity that has both.
type MovementSystem struct {
	camera *castrum.Camera
}

func (s *MovementSystem) Init(world *castrum.World) error {
	return nil
}

func (s *MovementSystem) Update(world *castrum.World, delta float64) error {
	// Query for entities with both Velocity and Transform
	entities := world.Query(castrum.Types(components.Transform{}, gamecomponents.Velocity{})...)

	for _, id := range entities {
		vel, err := castrum.GetComponent[gamecomponents.Velocity](world, id)
		if err != nil {
			continue
		}
		transform, err := castrum.GetComponent[components.Transform](world, id)
		if err != nil {
			continue
		}

		// Apply velocity to position
		transform.Position.X += vel.Linear.X * delta
		transform.Position.Y += vel.Linear.Y * delta

		// Clamp position to world bounds
		if transform.Position.X < s.camera.Bounds.Min.X {
			transform.Position.X = s.camera.Bounds.Min.X
		}
		if transform.Position.Y < s.camera.Bounds.Min.Y {
			transform.Position.Y = s.camera.Bounds.Min.Y
		}
		if transform.Position.X > s.camera.Bounds.Max.X {
			transform.Position.X = s.camera.Bounds.Max.X
		}
		if transform.Position.Y > s.camera.Bounds.Max.Y {
			transform.Position.Y = s.camera.Bounds.Max.Y
		}

		if err := castrum.SetComponent(world, id, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *MovementSystem) Shutdown(world *castrum.World) error {
	return nil
}

func NewMovementSystem(camera *castrum.Camera) *MovementSystem {
	return &MovementSystem{camera: camera}
}
