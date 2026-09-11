package systems

import (
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/input"
)

// CameraSystem is responsible for managing the camera within the game world.
type CameraSystem struct {
	Input       input.Reader
	cameraQuery *ecs.Query
	playerQuery *ecs.Query
}

// Update finds the player and updates the primary camera to follow it.
// Camera is now an ECS component, so we query it from the world.
func (cs *CameraSystem) Update(world *ecs.World, delta float64) error {
	// Query for the primary camera
	cameras := cs.cameraQuery.EntityIDs()
	var cameraEntity ecs.EntityID
	var found bool

	for _, eid := range cameras {
		cam, err := world.GetComponent[components.Camera](eid)
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
	players := cs.playerQuery.EntityIDs()
	if len(players) > 0 {
		playerEntity := players[0]
		// Get player position
		if tx, err := world.GetComponent[components.Transform](playerEntity); err == nil {
			// Get current camera
			cam, err := world.GetComponent[components.Camera](cameraEntity)
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

	for _, action := range []struct {
		name     input.Action
		multiply float64
	}{
		{actionCameraZoomIn, zoomFactor},
		{actionCameraZoomOut, 1 / zoomFactor},
	} {
		if cs.Input.ActionHeld(action.name) {
			// Get camera for zoom update
			cam, err := world.GetComponent[components.Camera](cameraEntity)
			if err != nil {
				continue
			}

			cam.Zoom *= action.multiply
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

func (cs *CameraSystem) Init(world *ecs.World) error {
	cs.cameraQuery = ecs.NewQuery(world).WithRequiredComponents(components.Camera{})
	cs.playerQuery = ecs.NewQuery(world).WithRequiredComponents(gamecomponents.Player{})
	return nil
}

func (cs *CameraSystem) Shutdown(world *ecs.World) error {
	return nil
}
