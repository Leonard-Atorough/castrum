package camera

import (
	"math"
	"reflect"

	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

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

// System updates all Camera components in the world.
// It handles viewport clamping and other camera-specific logic.
type System struct{}

func NewSystem() *System {
	return &System{}
}

func (cs *System) Init(world *core.World) error {
	return nil
}

// Update applies camera logic: clamps position within bounds.
func (cs *System) Update(world *core.World, deltaTime float64) error {
	cameras := core.QueryFor[Camera](world)
	for _, entityID := range cameras {
		camComp, err := world.GetComponent(entityID, reflect.TypeFor[Camera]())
		if err != nil {
			continue
		}

		cam := camComp.(Camera)
		// Clamp position within bounds
		cam = cam.ClampPosition()
		// Update component back in world
		world.SetComponent(entityID, reflect.TypeFor[Camera](), cam)
	}

	return nil
}

func (cs *System) Shutdown(world *core.World) error {
	return nil
}
