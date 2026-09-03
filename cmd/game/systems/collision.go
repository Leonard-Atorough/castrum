package systems

import (
	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/internal/collision"
)

// CollisionSystem handles collision response logic using event-based collision events
// and contact geometry from the collision manager. It demonstrates the collision
// API by destroying obstacles on Enter, logging Stay events, and removing entities on Exit.
type CollisionSystem struct {
	collision *castrum.Collision
}

func NewCollisionSystem(collisionMgr *castrum.Collision) *CollisionSystem {
	return &CollisionSystem{collision: collisionMgr}
}

// Update processes collision events emitted by the collision manager each frame.
// Collision events include contact geometry (point, normal, penetration) for
// sophisticated game logic like knockback, sliding, or environmental reactions.
func (c *CollisionSystem) Update(world *castrum.World, deltaTime float64) error {
	if c.collision == nil {
		return nil
	}

	// Process all collision events emitted this frame (Enter, Stay, Exit)
	events := c.collision.Events()
	for _, evt := range events {
		switch evt.CollisionEventType {
		case collision.CollisionEnter:
			// Contact detected: destroy the obstacle (non-player entity)
			// In a real game, use evt.Point (contact location) and evt.Normal
			// (surface direction) for knockback, vfx, or sound.
			if c.isPlayer(world, evt.PairKey.EntityA) {
				world.DestroyEntity(evt.PairKey.EntityB, true)
			} else if c.isPlayer(world, evt.PairKey.EntityB) {
				world.DestroyEntity(evt.PairKey.EntityA, true)
			}

		case collision.CollisionStay:
			// Contact ongoing. Contact geometry is recomputed if either entity moved,
			// or cached from previous frame if both static. Use for sustained effects.
			_ = evt // Stay handling for game logic goes here (e.g., damage over time)

		case collision.CollisionExit:
			// Contact ended. Use for cleanup (remove burn effect, stop sound, etc).
			_ = evt
		}
	}

	return nil
}

// isPlayer checks if an entity has a Player component.
func (c *CollisionSystem) isPlayer(world *castrum.World, entityID castrum.EntityID) bool {
	_, err := castrum.GetComponent[gamecomponents.Player](world, entityID)
	return err == nil
}

func (c *CollisionSystem) Init(world *castrum.World) error {
	return nil
}

func (c *CollisionSystem) Shutdown(world *castrum.World) error {
	return nil
}
