package systems

import (
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/input"
)

// PlayerController reads input and updates velocity on entities with a Player marker.
type PlayerController struct {
	input input.Reader
}

func NewPlayerController(input input.Reader) *PlayerController {
	return &PlayerController{input: input}
}

func (pc *PlayerController) Init(world *ecs.World) error {
	return nil
}

func (pc *PlayerController) Update(world *ecs.World, delta float64) error {
	for entry := range world.NewQuery().WithRequiredComponents(gamecomponents.Player{}, gamecomponents.Velocity{}, components.Transform{}).Execute() {
		vel, err := entry.Get[gamecomponents.Velocity]()
		if err != nil {
			continue
		}

		// Read input and update velocity
		speed := 300.0 // pixels per second
		vel.Linear = geom.Vector2{X: 0, Y: 0}

		if pc.input.ActionHeld(actionMoveUp) {
			vel.Linear.Y -= speed
		}
		if pc.input.ActionHeld(actionMoveDown) {
			vel.Linear.Y += speed
		}
		if pc.input.ActionHeld(actionMoveLeft) {
			vel.Linear.X -= speed
		}
		if pc.input.ActionHeld(actionMoveRight) {
			vel.Linear.X += speed
		}

		if err := world.SetComponent(entry.EntityID, vel); err != nil {
			return err
		}
	}
	return nil
}

func (pc *PlayerController) Shutdown(world *ecs.World) error {
	return nil
}
