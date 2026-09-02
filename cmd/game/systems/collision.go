package systems

import (
	"fmt"

	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/cmd/game/components"
)

type CollisionSystem struct {
}

func (c *CollisionSystem) Update(world *castrum.World, deltaTime float64) {
	// first we query archetype for entites with Collider components
	colliders := castrum.QueryFor[components.Collider](world)

	// we want to check collision between the player (which has a marker component of player) and the other colliders so
	playerCollider := castrum.QueryFor[components.Player](world)

	// iterate through all colliders against the player collider (expect for the collider that is the player collider and use a collider.)
	// Each collider implements an interface which exposes a shape field via the Shape function and each collider implement the Shape function
	// to return a shape type. All shape geom implement an intersect check.
	fmt.Printf("Colliders: %v, PlayerColliders: %v\n", colliders, playerCollider)
}
