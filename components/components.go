package components

import (
	"image/color"
	"math"

	"github.com/leonard-atorough/castrum/geom"
)

// Transform represents the position, rotation, scale, and color of an entity.
// It is used to control the visual representation and transformation of an entity in the scene.
type Transform struct {
	Position geom.Vector2
	Rotation float64
	Scale    geom.Vector2
	Color    color.Color
}

// NewTransform creates a Transform component with the specified position, rotation, and scale.
// The color is set to transparent if not provided.
func NewTransform(position geom.Vector2, rotation float64, scale geom.Vector2, c color.Color) Transform {
	if c == nil {
		c = color.Transparent
	}
	return Transform{
		Position: position,
		Rotation: rotation,
		Scale:    scale,
		Color:    c,
	}
}

func NewTransformWithDefault() Transform {
	return NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, color.Transparent)
}

// SceneTag marks which scene an entity belongs to, for query-time scene filtering.
type SceneTag struct {
	SceneID string
}

type Sprite struct {
	TexturePath string
	Primitive   PrimitiveType
	Layer       uint8 // which of 32 layers to render on (0-31)
	SortOrder   int8  // [-128..127], higher values render on top within the layer
	Visible     bool
	Data        any // holds additional data for the primitive, e.g., *Polygon for PrimitiveKindPolygon
}

// NewSprite creates a new Renderable component with the specified properties.
func NewSprite(texturePath string, Primitive PrimitiveType, Layer uint8, SortOrder int8, Visible bool, Data any) Sprite {
	if Data == nil {
		Data = struct{}{}
	}
	if Layer > 31 {
		Layer = 31
	}
	if Primitive < PrimitiveKindRectangle || Primitive > PrimitiveKindPolygon {
		Primitive = PrimitiveKindRectangle
	}
	return Sprite{
		TexturePath: texturePath,
		Primitive:   Primitive,
		Layer:       Layer,
		SortOrder:   SortOrder,
		Visible:     Visible,
		Data:        Data,
	}
}

// PrimitiveType represents the type of a procedural shape for rendering.
type PrimitiveType int

const (
	PrimitiveKindRectangle PrimitiveType = iota
	PrimitiveKindCircle
	PrimitiveKindLine
	PrimitiveKindPolygon
)

// Animation holds playback state for an animating entity.
// The animation definition (frames, frame speed, loop) is managed by the AnimationManager.
type Animation struct {
	ClipPath      string  // ID to look up the AnimationClip in the manager
	FrameIndex    int     // current frame index
	FrameTime     float64 // accumulated time for current frame (seconds)
	Playing       bool    // is the animation running
	PlaybackSpeed float64 // playback multiplier (1.0 = normal speed)
}

// NewAnimation creates an Animation for a given clip ID.
func NewAnimation(clipID string, autoplay bool) Animation {
	return Animation{
		ClipPath:      clipID,
		PlaybackSpeed: 1.0,
		Playing:       autoplay,
	}
}

// ColliderShapeContext defines the interface that collision shapes must implement to provide a bounding box.
type ColliderShapeContext interface {
	BoundingBox() geom.Rect
}

// Collider represents a collision shape for an entity.
type Collider struct {
	Shape   ColliderShapeContext // geom.Circle or geom.Rect, defined in local space
	Layer   uint8                // The layer this collider belongs to
	Mask    uint32               // The collision masks determine which layers this collider can interact with.
	Trigger bool                 // Indicates if this collider is a trigger (does not generate physical collisions)
	Active  bool                 // Indicates if this collider is currently active
}

// NewCollider creates a new Collider component with the specified properties.
func NewCollider(shape ColliderShapeContext, active, trigger bool, layer uint8, collidesWith ...uint) Collider {
	mask := layersToMask(collidesWith...)
	if layer > 31 {
		layer = 31
	}
	return Collider{
		Shape:   shape,
		Layer:   layer,
		Mask:    mask,
		Active:  active,
		Trigger: trigger,
	}
}

func (c Collider) BoundingBox() geom.Rect {
	return c.Shape.BoundingBox()
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

// Camera is a component that defines a viewport for rendering.
// Mark Primary=true to make it the active camera used by the renderer.
type Camera struct {
	// The position of the camera in world space
	Position geom.Vector2
	// The zoom level of the camera
	Zoom float64
	// The size of the screen in pixels
	ScreenSize geom.Vector2I
	// The rotation of the camera in radians
	Rotation float64
	// The rectangular bounds within which the camera can move
	Bounds geom.Rect
	// Mark this as the primary (active) camera for rendering
	Primary bool
}

// NewCamera returns a camera centered on the world origin with no zoom and no
// movement bounds. Call SetScreenSize once the render target size is known;
// set Bounds explicitly to constrain movement.
func NewCamera() Camera {
	return Camera{
		Zoom:   1,
		Bounds: unboundedRect(),
	}
}

func unboundedRect() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: math.Inf(-1), Y: math.Inf(-1)},
		Max: geom.Vector2{X: math.Inf(1), Y: math.Inf(1)},
	}
}

// SetScreenSize updates the render target size the camera converts against.
func (c *Camera) SetScreenSize(width, height int) {
	c.ScreenSize = geom.Vector2I{X: width, Y: height}
}

// WorldToScreen converts a position in world space to screen space based on the camera's position, zoom, and screen size.
func (c Camera) WorldToScreen(worldPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (worldPos.X-c.Position.X)*c.Zoom + float64(c.ScreenSize.X)/2,
		Y: (worldPos.Y-c.Position.Y)*c.Zoom + float64(c.ScreenSize.Y)/2,
	}
}

// ScreenToWorld converts a position in screen space to world space based on the camera's position, zoom, and screen size.
func (c Camera) ScreenToWorld(screenPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (screenPos.X-float64(c.ScreenSize.X)/2)/c.Zoom + c.Position.X,
		Y: (screenPos.Y-float64(c.ScreenSize.Y)/2)/c.Zoom + c.Position.Y,
	}
}

// ViewportBounds returns the visible rectangle in world coordinates based on the camera's position, zoom, and screen size.
func (c Camera) ViewportBounds() geom.Rect {
	halfWidth := float64(c.ScreenSize.X) / (2 * c.Zoom)
	halfHeight := float64(c.ScreenSize.Y) / (2 * c.Zoom)
	min := geom.Vector2{
		X: c.Position.X - halfWidth,
		Y: c.Position.Y - halfHeight,
	}
	max := geom.Vector2{
		X: c.Position.X + halfWidth,
		Y: c.Position.Y + halfHeight,
	}
	return geom.Rect{Min: min, Max: max}
}

// ClampPosition constrains the camera position within its bounds.
// Returns a new Camera with the clamped position; the receiver is not modified.
func (c Camera) ClampPosition() Camera {
	viewport := c.ViewportBounds()
	if viewport.Min.X < c.Bounds.Min.X {
		c.Position.X += c.Bounds.Min.X - viewport.Min.X
	}
	if viewport.Max.X > c.Bounds.Max.X {
		c.Position.X -= viewport.Max.X - c.Bounds.Max.X
	}
	if viewport.Min.Y < c.Bounds.Min.Y {
		c.Position.Y += c.Bounds.Min.Y - viewport.Min.Y
	}
	if viewport.Max.Y > c.Bounds.Max.Y {
		c.Position.Y -= viewport.Max.Y - c.Bounds.Max.Y
	}
	return c
}

// IsWorldRectVisible checks if a world space rectangle is within the camera's viewport.
func (c Camera) IsWorldRectVisible(worldRect geom.Rect) bool {
	viewport := c.ViewportBounds()
	return viewport.Intersects(worldRect)
}

// AspectRatio returns the aspect ratio of the camera's screen.
func (c Camera) AspectRatio() float64 {
	return float64(c.ScreenSize.X) / float64(c.ScreenSize.Y)
}
