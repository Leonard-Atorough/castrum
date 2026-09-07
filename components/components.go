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

// NewTransform creates a Transform with position, rotation, scale, and optional color.
func NewTransform(position geom.Vector2, rotation float64, scale geom.Vector2) Transform {
	return Transform{
		Position: position,
		Rotation: rotation,
		Scale:    scale,
		Color:    nil,
	}
}

// NewTransformWithColor creates a Transform with all fields specified.
func NewTransformWithColor(position geom.Vector2, rotation float64, scale geom.Vector2, c color.Color) Transform {
	return Transform{
		Position: position,
		Rotation: rotation,
		Scale:    scale,
		Color:    c,
	}
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

// NewRenderable creates a Renderable with texture path and layer.
func NewRenderable(texturePath string, layer RenderLayer) Renderable {
	return Renderable{
		TexturePath: texturePath,
		Layer:       layer,
		Visible:     true,
	}
}

// NewRenderablePrimitive creates a Renderable for a procedural shape (Rectangle, Circle, etc).
func NewRenderablePrimitive(primitive PrimitiveKind, layer RenderLayer) Renderable {
	return Renderable{
		Primitive: primitive,
		Layer:     layer,
		Visible:   true,
	}
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

// NewAnimation creates an Animation for a given clip path.
func NewAnimation(clipPath string) Animation {
	return Animation{
		ClipPath:      clipPath,
		PlaybackSpeed: 1.0,
		Playing:       false,
	}
}

// Spin rotates an entity's Transform by AngularVelocity radians per second.
type Spin struct {
	AngularVelocity float64
}

// NewSpin creates a Spin component with the given angular velocity (radians per second).
func NewSpin(angularVelocity float64) Spin {
	return Spin{AngularVelocity: angularVelocity}
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

// TimerID uniquely identifies a timer within a Manager.
type TimerID string

// Timer is a component that tracks elapsed time and fires a callback when
// its duration is reached. Timers can be one-shot (fires once then is removed)
// or repeating (resets and continues firing).
//
// Timer is a value type and should be attached to entities via AddComponent.
// The TimerSystem handles update logic and callback firing.
type Timer struct {
	// Unique identifier for this timer (useful for multiple timers per entity)
	ID TimerID
	// Duration for which the timer runs
	Duration float64
	// Elapsed time since the timer started (accumulated by TimerSystem)
	ElapsedTime float64
	// Whether the timer is currently running
	Running bool
	// Whether this is a one-shot timer (fires once then stops)
	Once bool
}

// Start begins the timer, resetting elapsed time to zero.
func (t *Timer) Start() {
	t.Running = true
	t.ElapsedTime = 0
}

// Stop pauses the timer without resetting elapsed time.
// Resume can be called to continue from the same elapsed time.
func (t *Timer) Stop() {
	t.Running = false
}

// Resume continues a stopped timer from its current elapsed time.
func (t *Timer) Resume() {
	t.Running = true
}
