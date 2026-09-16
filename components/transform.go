package components

import "github.com/leonard-atorough/castrum/geom"

// Transform represents the spatial state of an entity: position, rotation,
// and scale. Scale is always a multiplier, not a pixel size — {1,1} means
// no scaling. For primitives, the base size comes from Sprite.Size; for
// textured sprites, it comes from the image or atlas region dimensions.
type Transform struct {
	Position geom.Vector2
	Rotation float64
	Scale    geom.Vector2
	Origin   geom.Vector2
}

// NewTransform creates a Transform component with the specified position,
// rotation, scale, and origin.
func NewTransform(position geom.Vector2, rotation float64, scale geom.Vector2, origin geom.Vector2) Transform {
	return Transform{
		Position: position,
		Rotation: rotation,
		Scale:    scale,
		Origin:   origin,
	}
}
