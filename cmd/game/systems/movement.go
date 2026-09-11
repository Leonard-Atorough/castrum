package systems

import (
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
)

// MovementSystem applies Velocity to Transform.Position for every entity that has both.
type MovementSystem struct {
	camera *components.Camera
}

func (s *MovementSystem) Init(world *ecs.World) error {
	return nil
}

func (s *MovementSystem) Update(world *ecs.World, delta float64) error {
	// Query for entities with both Velocity and Transform

	for entry := range world.NewQuery().WithRequiredComponents(components.Transform{}, gamecomponents.Velocity{}).Execute() {
		id := entry.EntityID
		vel, _ := entry.Get[gamecomponents.Velocity]()
		transform, _ := entry.Get[components.Transform]()

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

		if err := world.SetComponent(id, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *MovementSystem) Shutdown(world *ecs.World) error {
	return nil
}

func NewMovementSystem(camera *components.Camera) *MovementSystem {
	return &MovementSystem{camera: camera}
}
