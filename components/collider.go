package components

import (
	"fmt"

	"github.com/leonard-atorough/castrum/geom"
)

// ColliderShapeContext defines the local-space bounds required by a collider.
// Physics currently supports [geom.Circle] and [geom.Rect] for exact collision
// testing. Other bounds-providing types are not valid physics shapes yet.
type ColliderShapeContext interface {
	BoundingBox() geom.Rect
}

// Collider represents a collision shape for an entity.
//
// Shape is a [geom.Circle] or [geom.Rect] defined in local space (relative to
// the entity, before Transform is applied). Offset shifts the shape within the
// entity's local space — useful for hitboxes that aren't centered on the
// entity origin (e.g., a foot collider below the sprite).
//
// Layer and Mask control which colliders can interact. A collider on layer L
// with mask M can collide with a collider on layer L2 if bit L2 is set in M
// and bit L is set in the other collider's mask. Layers are 0-31.
//
// Trigger colliders detect and emit collision events without implying a
// physical response. Active controls whether the collider participates in
// collision detection at all.
type Collider struct {
	Shape   ColliderShapeContext // geom.Circle or geom.Rect, defined in local space
	Layer   uint8                // collision layer (0-31)
	Mask    uint32               // bitmask of layers this collider can interact with
	Trigger bool                 // detect and emit events without collision response
	Active  bool                 // whether this collider participates in collision detection
	Offset  geom.Vector2         // local-space offset from the entity origin
}

// NewCollider creates a validated Collider component. Layer is clamped to 31.
// CollidesWith specifies which layers (by index) this collider can interact
// with; the resulting mask is the bitwise OR of (1 << layer) for each.
func NewCollider(shape ColliderShapeContext, active, trigger bool, offset geom.Vector2, layer uint8, collidesWith ...uint) (Collider, error) {
	if layer > 31 {
		layer = 31
	}
	collider := Collider{
		Shape:   shape,
		Layer:   layer,
		Mask:    layersToMask(collidesWith...),
		Active:  active,
		Trigger: trigger,
		Offset:  offset,
	}
	if err := collider.Validate(); err != nil {
		return Collider{}, fmt.Errorf("invalid collider: %w", err)
	}
	return collider, nil
}

// Validate checks that the collider has a supported shape and a valid layer.
func (c Collider) Validate() error {
	if c.Shape == nil {
		return fmt.Errorf("collider shape is nil")
	}
	if !c.IsSupportedShape() {
		return fmt.Errorf("unsupported collider shape: %T", c.Shape)
	}
	if c.Layer > 31 {
		return fmt.Errorf("collider layer %d exceeds maximum of 31", c.Layer)
	}
	return nil
}

// Serialize converts the Collider to a map suitable for blueprint
// serialization or save-game storage. The shape is serialized by type:
// geom.Circle as a center+radius map, geom.Rect as a min+max map.
func (c Collider) Serialize() (map[string]any, error) {
	data := map[string]any{
		"layer":   c.Layer,
		"mask":    c.Mask,
		"trigger": c.Trigger,
		"active":  c.Active,
		"offset": map[string]float64{
			"x": c.Offset.X,
			"y": c.Offset.Y,
		},
	}

	switch shape := c.Shape.(type) {
	case geom.Circle:
		data["shape"] = map[string]any{
			"type":   "circle",
			"center": map[string]float64{"x": shape.Center.X, "y": shape.Center.Y},
			"radius": shape.Radius,
		}
	case geom.Rect:
		data["shape"] = map[string]any{
			"type": "rect",
			"min":  map[string]float64{"x": shape.Min.X, "y": shape.Min.Y},
			"max":  map[string]float64{"x": shape.Max.X, "y": shape.Max.Y},
		}
	default:
		return nil, fmt.Errorf("cannot serialize unsupported shape %T", c.Shape)
	}

	return data, nil
}

// Deserialize populates a Collider from a serialized map, returning the
// reconstructed component. The shape is reconstructed by type discriminator.
// The result is validated before returning.
func (c Collider) Deserialize(data map[string]any) (Collider, error) {
	if shapeData, ok := data["shape"].(map[string]any); ok {
		shapeType, _ := shapeData["type"].(string)
		switch shapeType {
		case "circle":
			center := geom.Vector2{}
			if centerMap, ok := shapeData["center"].(map[string]any); ok {
				if x, ok := centerMap["x"].(float64); ok {
					center.X = x
				}
				if y, ok := centerMap["y"].(float64); ok {
					center.Y = y
				}
			}
			radius, _ := shapeData["radius"].(float64)
			c.Shape = geom.Circle{Center: center, Radius: radius}
		case "rect":
			min := geom.Vector2{}
			max := geom.Vector2{}
			if minMap, ok := shapeData["min"].(map[string]any); ok {
				if x, ok := minMap["x"].(float64); ok {
					min.X = x
				}
				if y, ok := minMap["y"].(float64); ok {
					min.Y = y
				}
			}
			if maxMap, ok := shapeData["max"].(map[string]any); ok {
				if x, ok := maxMap["x"].(float64); ok {
					max.X = x
				}
				if y, ok := maxMap["y"].(float64); ok {
					max.Y = y
				}
			}
			c.Shape = geom.Rect{Min: min, Max: max}
		}
	}
	if v, ok := data["layer"].(float64); ok {
		c.Layer = uint8(v)
	}
	if v, ok := data["mask"].(float64); ok {
		c.Mask = uint32(v)
	}
	if v, ok := data["trigger"].(bool); ok {
		c.Trigger = v
	}
	if v, ok := data["active"].(bool); ok {
		c.Active = v
	}
	if offset, ok := data["offset"].(map[string]any); ok {
		if x, ok := offset["x"].(float64); ok {
			c.Offset.X = x
		}
		if y, ok := offset["y"].(float64); ok {
			c.Offset.Y = y
		}
	}
	return c, c.Validate()
}

// BoundingBox returns the local-space bounding box of the collider shape,
// shifted by Offset. Returns a zero Rect if the shape is nil.
func (c Collider) BoundingBox() geom.Rect {
	if c.Shape == nil {
		return geom.Rect{}
	}
	box := c.Shape.BoundingBox()
	return geom.Rect{
		Min: geom.Vector2{X: box.Min.X + c.Offset.X, Y: box.Min.Y + c.Offset.Y},
		Max: geom.Vector2{X: box.Max.X + c.Offset.X, Y: box.Max.Y + c.Offset.Y},
	}
}

// IsSupportedShape reports whether physics has exact narrow-phase support for
// the collider's concrete shape type.
func (c Collider) IsSupportedShape() bool {
	switch c.Shape.(type) {
	case geom.Circle, geom.Rect:
		return true
	default:
		return false
	}
}

// CanCollideWith reports whether two colliders can interact based on their
// layer/mask configuration. Both colliders must have the other's layer bit set
// in their respective masks.
func (c Collider) CanCollideWith(other Collider) bool {
	return (c.Mask&(1<<other.Layer)) != 0 && (other.Mask&(1<<c.Layer)) != 0
}

func layersToMask(layers ...uint) uint32 {
	var mask uint32
	for _, layer := range layers {
		mask |= 1 << layer
	}
	return mask
}
