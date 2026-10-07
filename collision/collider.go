package collision

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Collider is an entity's collision shape and interaction filter. The
// collision system reads it each fixed tick, transforms the shape by
// the entity's Transform - rotation and position, never scale - and
// records the results as contact state on the entity.
//
// The zero value is not spawnable: it has no shape, and Validate
// rejects it. Build one with [NewCollider] or a struct literal with
// a valid shape; every write into storage re-validates.
type Collider struct {
	// Shape is the collider's geometry, defined in local space before
	// the Transform applies.
	Shape ColliderShape
	// Offset shifts the shape from the entity's local origin, for
	// hitboxes that are not centered on it - a foot collider below
	// the sprite, a sword reach in front of it.
	Offset geom.Vector2
	// Layer is the collider's own collision layer, 0-31.
	Layer uint8
	// Mask is the bitmask of layers this collider interacts with:
	// bit i is layer i. The zero value is an empty mask, which
	// collides with nothing - a deliberate choice, not a default;
	// [NewCollider] opens it to every layer.
	Mask uint32
	// Trigger marks a collider that detects contacts without
	// implying a physical response. Detection does not differ;
	// response, when the engine grows one, will.
	Trigger bool
	// Active controls whether the collider participates in collision
	// detection at all.
	Active bool
}

// Validate checks that the collider is spawnable: a supported shape
// with valid geometry, a finite offset, and a layer in range. It
// rejects rather than repairs - an inverted rect or a negative radius
// is an authoring error, not input to normalize.
func (c Collider) Validate() error {
	switch shape := c.Shape.(type) {
	case nil:
		return fmt.Errorf("collider shape is nil")
	case RectShape:
		rect := geom.Rect{Min: shape.Min, Max: shape.Max}
		if !rect.IsValid() {
			return fmt.Errorf("rect shape must have finite, canonical bounds")
		}
		if rect.Width() <= 0 || rect.Height() <= 0 {
			return fmt.Errorf("rect shape must have positive width and height")
		}
	case CircleShape:
		if !isFinite(shape.Center.X) || !isFinite(shape.Center.Y) {
			return fmt.Errorf("circle shape must have a finite center")
		}
		if !isFinite(shape.Radius) || shape.Radius <= 0 {
			return fmt.Errorf("circle shape must have a positive radius")
		}
	default:
		return fmt.Errorf("unknown collider shape %T", c.Shape)
	}
	if !isFinite(c.Offset.X) || !isFinite(c.Offset.Y) {
		return fmt.Errorf("collider offset must be finite")
	}
	if c.Layer > 31 {
		return fmt.Errorf("layer must be between 0 and 31")
	}
	return nil
}

// NewCollider returns a Collider for shape with usable defaults: it
// is active and its mask is open to every layer. The remaining fields
// are plain and settable - assign Layer, Mask, Trigger, or Offset
// afterwards as needed; writing the result into storage re-validates
// it.
//
// It returns an error when shape is not a valid collider shape, so an
// authoring mistake surfaces at construction instead of at spawn.
func NewCollider(shape ColliderShape) (Collider, error) {
	collider := Collider{
		Shape:  shape,
		Mask:   math.MaxUint32,
		Active: true,
	}
	if err := collider.Validate(); err != nil {
		return Collider{}, err
	}
	return collider, nil
}

// CanCollideWith reports whether the layer/mask configuration allows
// the two colliders to interact: each must carry the other's layer in
// its mask. A one-sided mask does not collide.
func (c Collider) CanCollideWith(other Collider) bool {
	return c.Mask&(1<<other.Layer) != 0 && other.Mask&(1<<c.Layer) != 0
}

// ColliderShape is a collider's geometry: one of the concrete shapes
// in this package. It is a sealed sum - the only implementations are
// in this file - so exactly-one-shape is a construction guarantee,
// and the narrow phase type-switches one field instead of querying
// per variant.
type ColliderShape interface {
	isColliderShape()
}

// RectShape is an axis-aligned rectangle in local space. Min must be
// strictly below Max on both axes; Validate rejects inverted or
// degenerate rects instead of normalizing them.
type RectShape struct {
	Min geom.Vector2
	Max geom.Vector2
}

// CircleShape is a circle in local space. Radius must be positive; a
// zero or negative radius is an authoring error, not a magnitude.
type CircleShape struct {
	Center geom.Vector2
	Radius float64
}

func (RectShape) isColliderShape()   {}
func (CircleShape) isColliderShape() {}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
