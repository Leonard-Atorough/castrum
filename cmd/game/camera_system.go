package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/components"
)

// CameraSystem is responsible for managing the camera within the game world.
type CameraSystem struct {
	Camera *castrum.Camera
	Input  *castrum.Input
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

	zoomFactor := 1.015 // 5% per frame

	for _, key := range []struct {
		ebitenKey ebiten.Key
		multiply  float64
	}{
		{ebiten.KeyZ, zoomFactor},     // Zoom in
		{ebiten.KeyX, 1 / zoomFactor}, // Zoom out (multiply by 0.95)
	} {
		if cs.Input.KeyHeld(key.ebitenKey, false, true, false) {
			cs.Camera.Zoom *= key.multiply
			// Clamp to sensible bounds
			if cs.Camera.Zoom < 0.1 {
				cs.Camera.Zoom = 0.1
			}
			if cs.Camera.Zoom > 10.0 {
				cs.Camera.Zoom = 10.0
			}
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
