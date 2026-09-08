package camera

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

// System updates all Camera components in the world.
// It handles viewport clamping and other camera-specific logic.
type System struct {
	cameraQuery *core.Query
}

func (cs *System) Init(world *core.World) error {
	cs.cameraQuery = world.NewQuery().WithRequiredComponents(components.Camera{})
	return nil
}

// Update applies camera logic: clamps position within bounds.
func (cs *System) Update(world *core.World, deltaTime float64) error {
	for result := range cs.cameraQuery.Execute() {
		entityID := result.EntityID
		cam, err := world.GetComponent[components.Camera](entityID)
		if err != nil {
			continue
		}

		// Clamp position within bounds
		cam = cam.ClampPosition()
		// Update component back in world
		world.SetComponent(entityID, cam)
	}

	return nil
}

func (cs *System) Shutdown(world *core.World) error {
	return nil
}
