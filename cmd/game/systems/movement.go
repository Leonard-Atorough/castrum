package systems

import (
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
)

// MovementSystem applies Velocity to Transform.Position for every entity that has both.
type MovementSystem struct {
}

func (s *MovementSystem) Init(world *ecs.World) error {
	return nil
}

func (s *MovementSystem) Update(world *ecs.World, delta float64) error {
    for entry := range world.NewQuery().
        WithRequiredComponents(components.Transform{}, gamecomponents.Velocity{}).
        Execute() {

        id := entry.EntityID
        vel, _ := entry.Get[gamecomponents.Velocity]()
        transform, _ := entry.Get[components.Transform]()

        // Apply velocity
        transform.Position.X += vel.Linear.X * delta
        transform.Position.Y += vel.Linear.Y * delta

        // DELETE: the clamping block (lines 31-42)

        if err := world.SetComponent(id, transform); err != nil {
            return err
        }
    }
    return nil
}

func (s *MovementSystem) Shutdown(world *ecs.World) error {
	return nil
}

func NewMovementSystem() *MovementSystem {
	return &MovementSystem{}
}
