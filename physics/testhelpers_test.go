package physics

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
)

// mustCollider is a test helper that constructs a Collider with a zero offset
// and panics on validation error. It keeps test call sites concise.
func mustCollider(shape components.ColliderShapeContext, active, trigger bool, layer uint8, collidesWith ...uint) components.Collider {
	c, err := components.NewCollider(shape, active, trigger, geom.Vector2{}, layer, collidesWith...)
	if err != nil {
		panic(err)
	}
	return c
}
