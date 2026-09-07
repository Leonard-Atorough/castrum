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

// SceneTag marks which scene an entity belongs to, for query-time scene filtering.
type SceneTag struct {
	SceneID string
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

type RenderDepth int

type Renderable struct {
	TexturePath string
	Primitive   PrimitiveKind
	Layer       RenderLayer
	Depth       RenderDepth // [0..n], higher values render on top
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

// Animation holds playback state for an animating entity.
// The animation definition (frames, frame speed, loop) is loaded separately as an asset (AnimationClip).
type Animation struct {
	ClipPath      string  // path to the .anim.yaml asset
	FrameIndex    int     // current frame
	FrameTime     float64 // accumulated time for current frame (seconds)
	Playing       bool    // is the animation running
	PlaybackSpeed float64 // playback multiplier (1.0 = normal speed)
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
