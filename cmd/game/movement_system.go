package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/types"
)

// MovementSystem applies Velocity to Transform.Position for every entity that has both.
type MovementSystem struct{}

func (s *MovementSystem) Init(world *castrum.World) error {
	return nil
}

func (s *MovementSystem) Update(world *castrum.World, delta float64) error {
	// Query for entities with both Velocity and Transform
	entities := world.Query(castrum.Types(components.Transform{}, Velocity{})...)

	for _, id := range entities {
		vel, err := castrum.GetComponent[Velocity](world, id)
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

		if err := castrum.SetComponent(world, id, transform); err != nil {
			return err
		}
	}
	return nil
}

func (s *MovementSystem) Shutdown(world *castrum.World) error {
	return nil
}

// PlayerController reads input and updates velocity on entities with a Player marker.
type PlayerController struct {
	input *castrum.Input
}

func NewPlayerController(game *castrum.Game) *PlayerController {
	return &PlayerController{input: game.Input}
}

func (pc *PlayerController) Init(world *castrum.World) error {
	return nil
}

func (pc *PlayerController) Update(world *castrum.World, delta float64) error {
	// Query for player entities (those with Player + Velocity + Transform)
	entities := world.Query(castrum.Types(Player{}, Velocity{}, components.Transform{})...)

	for _, id := range entities {
		vel, err := castrum.GetComponent[Velocity](world, id)
		if err != nil {
			continue
		}

		// Read input and update velocity
		speed := 300.0 // pixels per second
		vel.Linear = types.Vector2{X: 0, Y: 0}

		if pc.input.KeyHeld(ebiten.KeyArrowUp, false, false, false) {
			vel.Linear.Y -= speed
		}
		if pc.input.KeyHeld(ebiten.KeyArrowDown, false, false, false) {
			vel.Linear.Y += speed
		}
		if pc.input.KeyHeld(ebiten.KeyArrowLeft, false, false, false) {
			vel.Linear.X -= speed
		}
		if pc.input.KeyHeld(ebiten.KeyArrowRight, false, false, false) {
			vel.Linear.X += speed
		}

		if err := castrum.SetComponent(world, id, vel); err != nil {
			return err
		}
	}
	return nil
}

func (pc *PlayerController) Shutdown(world *castrum.World) error {
	return nil
}
