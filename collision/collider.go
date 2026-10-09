package collision

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Collider defines an entity's collision shape and interaction
// filter. An entity must also have a Transform to participate in
// detection. Each fixed tick, the collision system applies the
// transform's position and rotation, and records detected overlaps
// in the entity's Contacts.
//
// The zero value is not spawnable: it has no shape, and Validate
// rejects it. Build one with [NewCollider] or a struct literal with
// a valid shape.
type Collider struct {
	// Shape is the collider's geometry in local space, before the
	// entity's position and rotation are applied. Supported shapes are
	// Box and Circle.
	Shape ColliderShape
	// Offset shifts the shape from the entity's local origin, for
	// hitboxes that are not centered on it.
	//
	// e.g. a foot collider belowthe sprite, a sword reach in front of it.
	Offset geom.Vector2
	// Layers is the bitmask of layers this collider belongs to:
	// bit i is layer i. A collider may occupy any combination of the
	// 32 layers. The zero value belongs to no layer, so it collides
	// with nothing.
	//
	// [NewCollider] places it on layer 0, and [Layers] builds the
	// bitmask from layer numbers.
	Layers uint32
	// Mask is the bitmask of layers this collider interacts with:
	// bit i is layer i. The zero value is an empty mask, which
	// collides with nothing.
	//
	// [NewCollider] opens it to every layer, and [Mask] builds the
	// bitmask from layer numbers.
	Mask uint32
	// Trigger marks contacts involving this collider as triggers.
	Trigger bool
	// Active controls whether the collider participates in collision
	// detection at all.
	Active bool
}

// Validate checks that the collider is spawnable: a supported shape
// with valid geometry and a finite offset. It rejects rather than
// repairs - an inverted rect or a negative radius is an authoring
// error, not input to normalize. Layers and Mask accept any bitmask;
// a collider that belongs to no layer, like one that listens to
// none, simply collides with nothing.
func (c Collider) Validate() error {
	switch shape := c.Shape.(type) {
	case nil:
		return fmt.Errorf("collider shape is nil")
	case Box:
		rect := geom.Rect{Min: shape.Min, Max: shape.Max}
		if !rect.IsValid() {
			return fmt.Errorf("rect shape must have finite, canonical bounds")
		}
		if rect.Width() <= 0 || rect.Height() <= 0 {
			return fmt.Errorf("rect shape must have positive width and height")
		}
	case Circle:
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
	return nil
}

// NewCollider returns a ready-to-spawn Collider for shape. It starts
// active, belongs to layer 0, and listens to every layer. Set Layers,
// Mask, Trigger, or Offset afterwards to give it the interaction
// behavior the game needs; [Layers] and [Mask] turn layer numbers
// into bitmasks.
//
// NewCollider validates shape immediately, so an authoring mistake is
// reported while the entity is being built instead of later during a
// fixed tick. The collider is validated again when written into world
// storage.
func NewCollider(shape ColliderShape) (Collider, error) {
	collider := Collider{
		Shape:  shape,
		Layers: 1,
		Mask:   math.MaxUint32,
		Active: true,
	}
	if err := collider.Validate(); err != nil {
		return Collider{}, err
	}
	return collider, nil
}

// CanCollideWith reports whether the layer/mask configuration allows
// the two colliders to interact: each must listen to at least one
// layer the other sits on. A one-sided mask does not collide.
func (c Collider) CanCollideWith(other Collider) bool {
	return c.Mask&other.Layers != 0 && other.Mask&c.Layers != 0
}

// Layers returns the bitmask for the given layer indexes, for a
// Collider's Layers field: Layers(0, 2) is a collider that belongs
// to layers 0 and 2. The zero-argument form is the empty bitmask -
// a collider on no layer collides with nothing.
//
// It panics on an index outside 0-31.
func Layers(layers ...int) uint32 {
	return layersToMask("Layers", layers)
}

// Mask returns the bitmask for the given layer indexes, for a
// Collider's Mask field: Mask(0) listens to layer 0 alone. The
// zero-argument form is the empty mask - collide with nothing.
//
// It panics on an index outside 0-31.
func Mask(layers ...int) uint32 {
	return layersToMask("Mask", layers)
}

func layersToMask(name string, layers []int) uint32 {
	var mask uint32
	for _, layer := range layers {
		if layer < 0 || layer > 31 {
			panic(fmt.Sprintf("collision: %s: layer %d is outside 0-31", name, layer))
		}
		mask |= 1 << layer
	}
	return mask
}

// ColliderShape is a collider's geometry: one of the concrete shapes
// in this package. It is a sealed sum whose only implementations are
// in this file, and the narrow phase type-switches one field instead
// of querying per variant.
type ColliderShape interface {
	isColliderShape()
}

// Box is an axis-aligned rectangle in local space. Both corners
// must be finite, and Min must be strictly below Max on both axes.
// [Collider.Validate] rejects inverted or degenerate rectangles
// instead of normalizing them.
type Box struct {
	Min geom.Vector2
	Max geom.Vector2
}

// Circle is a circle in local space. Its center must be finite
// and its radius must be finite and positive; invalid values are
// rejected by [Collider.Validate].
type Circle struct {
	Center geom.Vector2
	Radius float64
}

func (Box) isColliderShape()    {}
func (Circle) isColliderShape() {}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
