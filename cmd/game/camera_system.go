package main

import (
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
)

// CameraSystem is responsible for managing the camera within the game world.
type CameraSystem struct {
	Camera *castrum.Camera
}

// We can demo here how we can use the query system to find the player entity and update the camera accordingly.
func (cs *CameraSystem) Update(world *castrum.World, delta float64) error {
	player := castrum.QueryFor[Player](world)

	if len(player) > 0 {
		playerEntity := player[0]
		// Assuming the player has a Position component
		if tx, err := castrum.GetComponent[components.Transform](world, playerEntity); err == nil {
			pos := tx.Position
			cs.Camera.Position = pos
		}
	}
	return nil
}

func (cs *CameraSystem) Init(world *castrum.World) error {
	return nil
}

func (cs *CameraSystem) Shutdown(world *castrum.World) error {
	return nil
}
