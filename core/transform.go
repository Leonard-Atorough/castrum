package core

import (
	"github.com/Leonard-Atorough/castrum/geom"
)

// Transform represents the 2d spatial state of an entity.
// Transform is an entity's placement in the world: position,
// rotation, and scale, relative to a position offset.
//
// A Transform always comes paired with a [PrevTransform]: spawn and
// AddComponent attach one automatically, snapshotting the position
// so the renderer can interpolate motion between fixed ticks.
type Transform struct {
	// Position represents the position of the transform in 2D space.
	Position geom.Vector2
	// Rotation represents the rotation of the transform in radians.
	Rotation float64
	// Scale represents the scale of the transform in 2D space. A zero
	// axis - the zero value - reads as unscaled: collection treats it
	// as 1.
	Scale geom.Vector2
	// Offset represents the offset of the transform relative to its position.
	Offset geom.Vector2
}
