package core

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Transform represents the 2d spatial state of an entity.
type Transform struct {
	// Position represents the position of the transform in 2D space.
	Position geom.Vector2
	// Rotation represents the rotation of the transform in radians.
	Rotation float64
	// Scale represents the scale of the transform in 2D space.
	Scale geom.Vector2
	// Offset represents the offset of the transform relative to its position.
	Offset geom.Vector2
}

func (t Transform) Validate() error {
	if t.Scale.X == 0 || t.Scale.Y == 0 {
		return fmt.Errorf("scale components must be non-zero")
	}
	return nil
}
