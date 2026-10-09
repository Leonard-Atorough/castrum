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
// in the entity's [Contacts].
//
// The zero value has no shape and is not valid for spawning. Use
// [NewCollider] or provide a valid shape, then configure its filter
// and activation as needed.
type Collider struct {
	// Shape is the collider's geometry in local space, before the
	// entity's position and rotation are applied. Supported shapes are
	// [Box] and [Circle].
	Shape ColliderShape
	// Offset shifts the shape from the entity's local origin, for
	// hitboxes that are not centered on it.
	//
	// For example, place a foot collider below a sprite or a sword
	// collider in front of its owner.
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
	// Active controls whether the collider participates in detection.
	Active bool
}

// Validate returns an error if the shape is unsupported or has invalid
// geometry, or if the offset is non-finite. It rejects rather than normalizes
// invalid geometry. Any layer and mask bitmasks are valid, including zero.
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

// NewCollider returns an active collider on layer 0 that listens to every
// layer. It validates shape before returning; invalid shapes produce an
// error. Set the fields to customize its filter, offset, trigger behavior,
// or activation. [Layers] and [Mask] convert layer indexes to bitmasks.
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

// Layers returns the bitmask for the given layer indexes for [Collider.Layers].
// For example, Layers(0, 2) includes layers 0 and 2. With no arguments it
// returns zero, so the collider belongs to no layer.
//
// It panics on an index outside 0-31.
func Layers(layers ...int) uint32 {
	return layersToMask("Layers", layers)
}

// Mask returns the bitmask for the given layer indexes for [Collider.Mask].
// For example, Mask(0) listens only to layer 0. With no arguments it returns
// zero, so the collider listens to no layers.
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

// ColliderShape is local-space geometry accepted by a [Collider]. The package
// provides [Box] and [Circle]; the interface is sealed to these built-in
// shapes.
type ColliderShape interface {
	isColliderShape()
}

// Box is an axis-aligned rectangle in collider-local space. Both corners must
// be finite, and Min must be strictly below Max on both axes.
// [Collider.Validate] rejects invalid bounds rather than normalizing them.
type Box struct {
	// Min is the rectangle's minimum local-space corner.
	Min geom.Vector2
	// Max is the rectangle's maximum local-space corner.
	Max geom.Vector2
}

// Circle is a circle in collider-local space. Its center must be finite and
// its radius finite and positive; [Collider.Validate] rejects invalid values.
type Circle struct {
	// Center is the circle's center in collider-local space.
	Center geom.Vector2
	// Radius is the circle's radius.
	Radius float64
}

func (Box) isColliderShape()    {}
func (Circle) isColliderShape() {}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
