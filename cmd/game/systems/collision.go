package systems

import (
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/cmd/game/components"
	"github.com/leonard-atorough/castrum/geom"
)

type CollisionSystem struct {
}

func (c *CollisionSystem) Update(world *castrum.World, deltaTime float64) {
	colliders := castrum.QueryFor[components.Collider](world)
	players := castrum.QueryFor[components.Player](world)

	for _, playerID := range players {
		playerCollider, _ := castrum.GetComponent[components.Collider](world, playerID)

		for _, colliderID := range colliders {
			if playerID != colliderID {
				otherCollider, _ := castrum.GetComponent[components.Collider](world, colliderID)

				if geom.IntersectsAny(playerCollider.Shape(), otherCollider.Shape()).Collided {
					world.DestroyEntity(playerID, true)
					break
				}
			}
		}
	}
}
