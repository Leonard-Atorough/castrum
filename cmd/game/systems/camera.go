package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/camera"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/input"
)

// CameraSystem is responsible for managing the camera within the game world.
type CameraSystem struct {
	Input *castrum.Input
}

// Update finds the player and updates the primary camera to follow it.
// Camera is now an ECS component, so we query it from the world.
func (cs *CameraSystem) Update(world *castrum.World, delta float64) error {
	// Query for the primary camera
	cameras := core.QueryFor[camera.Camera](world)
	var cameraEntity castrum.EntityID
	var found bool

	for _, eid := range cameras {
		cam, err := world.GetComponent[camera.Camera](eid)
		if err != nil {
			continue
		}
		if cam.Primary {
			cameraEntity = eid
			found = true
			break
		}
	}

	if !found {
		return nil // No primary camera found
	}

	// Query for the player
	players := castrum.QueryFor[gamecomponents.Player](world)
	if len(players) > 0 {
		playerEntity := players[0]
		// Get player position
		if tx, err := world.GetComponent[components.Transform](playerEntity); err == nil {
			// Get current camera
			cam, err := world.GetComponent[camera.Camera](cameraEntity)
			if err != nil {
				return nil
			}

			// Update camera position to follow player
			cam.Position = tx.Position
			// Apply camera bounds clamping
			cam = cam.ClampPosition()

			// Write camera back to world
			if err := world.SetComponent(cameraEntity, cam); err != nil {
				return err
			}
		}
	}

	// Handle zoom input
	zoomFactor := 1.015 // 5% per frame

	for _, key := range []struct {
		ebitenKey ebiten.Key
		multiply  float64
	}{
		{ebiten.KeyZ, zoomFactor},     // Zoom in
		{ebiten.KeyX, 1 / zoomFactor}, // Zoom out (multiply by 0.95)
	} {
		if cs.Input.KeyHeld(key.ebitenKey, input.Modifiers{Shift: false, Ctrl: true, Alt: false}) {
			// Get camera for zoom update
			cam, err := world.GetComponent[camera.Camera](cameraEntity)
			if err != nil {
				continue
			}

			cam.Zoom *= key.multiply
			// Clamp to sensible bounds
			if cam.Zoom < 0.1 {
				cam.Zoom = 0.1
			}
			if cam.Zoom > 10.0 {
				cam.Zoom = 10.0
			}

			// Write camera back to world
			if err := world.SetComponent(cameraEntity, cam); err != nil {
				continue
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
