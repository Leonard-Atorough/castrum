package components

import (
	"image/color"

	"github.com/leonard-atorough/castrum/geom"
)

type Transform struct {
	Position geom.Vector2
	Rotation float64
	Scale    geom.Vector2
	Color    color.Color
}

type RenderLayer int

const (
	Layer0 RenderLayer = iota
	Layer1
	Layer2
	Layer3
	Layer4
	Layer5
	Layer6
	Layer7
	Layer8
	Layer9
	Layer10
	// Debug layer for rendering debug information
	LayerDebug
)

type Renderable struct {
	TexturePath string
	Primitive   PrimitiveKind
	Layer       RenderLayer
	Visible     bool
	Data        any // holds additional data for the primitive, e.g., *Polygon for PrimitiveKindPolygon
}

type PrimitiveKind int

const (
	PrimitiveKindRectangle PrimitiveKind = iota
	PrimitiveKindCircle
	PrimitiveKindLine
	PrimitiveKindPolygon
)

type Animation struct {
	Frames      []string
	FrameEvents map[int]func() // optional callbacks for specific frames
	FrameIndex  int            // current frame index
	Callback    func()         // optional callback function to be called when the animation finishes
	FrameTime   float64        // time elapsed since the last frame change
	FrameSpeed  float64        // seconds per frame
	Loop        bool           // indicates whether the animation should loop
	AutoPlay    bool           // indicates whether the animation should start playing automatically
	Playing     bool           // indicates whether the animation is currently playing
}

type Animatable struct {
	Animations       map[string]Animation
	CurrentAnimation string // the name of the current animation

}

func NewAnimatable(frames map[string]Animation, currentAnimation string) Animatable {
	defaultAnimation := "default"
	if _, exists := frames[currentAnimation]; exists {
		defaultAnimation = currentAnimation
	}
	return Animatable{
		Animations:       frames,
		CurrentAnimation: defaultAnimation,
	}
}

// Spin rotates an entity's Transform by AngularVelocity radians per second.
type Spin struct {
	AngularVelocity float64
}

// Collider represents a collision shape for an entity.
type Collider struct {
	Shape   any    // geom.Circle or geom.Rect, defined in local space
	Layer   uint32 // The layer this collider belongs to
	Mask    uint32 // The collision masks determine which layers this collider can interact with.
	Trigger bool   // Indicates if this collider is a trigger (does not generate physical collisions)
	Active  bool   // Indicates if this collider is currently active
}

func NewCollider(shape any, active, trigger bool, layer uint32, collidesWith ...uint) Collider {
	mask := layersToMask(collidesWith...)

	return Collider{
		Shape:   shape,
		Layer:   layer,
		Mask:    mask,
		Active:  active,
		Trigger: trigger,
	}
}

func (c Collider) ColliderShape() any {
	return c.Shape
}

func (c Collider) BoundingBox() geom.Rect {
	switch s := c.Shape.(type) {
	case geom.Circle:
		return s.BoundingBox()
	case geom.Rect:
		return s
	default:
		return geom.Rect{}
	}
}

func (c Collider) CanCollideWith(other *Collider) bool {
	return (c.Mask&(1<<other.Layer)) != 0 && (other.Mask&(1<<c.Layer)) != 0
}

func layersToMask(layers ...uint) uint32 {
	var mask uint32 = 0
	for _, layer := range layers {
		mask |= 1 << layer
	}
	return mask
}
