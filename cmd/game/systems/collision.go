package systems

import (
	"fmt"

	"github.com/leonard-atorough/castrum"
	gamecomponents "github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/physics"
)

// CollisionSystem handles collision response logic using event-based collision events
// and contact geometry from the collision system. It demonstrates the collision
// API by destroying obstacles on Enter, logging Stay events, and removing entities on Exit.
type CollisionSystem struct {
	bus            *events.EventBus
	bufferedEvents []physics.CollisionEvent
	subscriptionID int
}

func NewCollisionSystem() *CollisionSystem {
	return &CollisionSystem{
		bufferedEvents: make([]physics.CollisionEvent, 0, 64),
	}
}

func (c *CollisionSystem) Init(world *ecs.World) error {
	// Get the EventBus from world resources
	bus, ok := castrum.GetResource[*events.EventBus](world)
	if !ok {
		return fmt.Errorf("EventBus not registered in world resources")
	}
	c.bus = bus

	// Subscribe to collision events
	c.subscriptionID = c.bus.On(func(_ events.EventMeta, evt physics.CollisionEvent) {
		c.bufferedEvents = append(c.bufferedEvents, evt)
	}, false)

	return nil
}

// Update processes collision events emitted by the collision manager each frame.
// Collision events include contact geometry (point, normal, penetration) for
// sophisticated game logic like knockback, sliding, or environmental reactions.
func (c *CollisionSystem) Update(world *ecs.World, deltaTime float64) error {
	if c.bus == nil {
		return nil
	}

	// Process all collision events buffered this frame (Enter, Stay, Exit)
	for _, evt := range c.bufferedEvents {
		switch evt.CollisionEventType {
		case physics.CollisionEnter:
			// Contact detected: destroy the obstacle (non-player entity)
			// In a real game, use evt.Point (contact location) and evt.Normal
			// (surface direction) for knockback, vfx, or sound.
			if c.isPlayer(world, evt.PairKey.EntityA) {
				world.DestroyEntity(evt.PairKey.EntityB, true)
			} else if c.isPlayer(world, evt.PairKey.EntityB) {
				world.DestroyEntity(evt.PairKey.EntityA, true)
			}

		case physics.CollisionStay:
			// Contact ongoing. Contact geometry is recomputed if either entity moved,
			// or cached from previous frame if both static. Use for sustained effects.
			_ = evt // Stay handling for game logic goes here (e.g., damage over time)

		case physics.CollisionExit:
			// Contact ended. Use for cleanup (remove burn effect, stop sound, etc).
			_ = evt
		}
	}

	// Clear buffered events for next frame
	c.bufferedEvents = c.bufferedEvents[:0]

	return nil
}

func (c *CollisionSystem) Shutdown(world *ecs.World) error {
	if c.bus != nil {
		c.bus.Unsubscribe(c.subscriptionID)
	}
	return nil
}

// isPlayer checks if an entity has a Player component.
func (c *CollisionSystem) isPlayer(world *ecs.World, entityID ecs.EntityID) bool {
	_, err := world.GetComponent[gamecomponents.Player](entityID)
	return err == nil
}
