package render

import (
	"math"

	"github.com/leonard-atorough/castrum/geom"
)

type Camera struct {
	// The position of the camera in world space.
	Position geom.Vector2
	// The zoom level of the camera.
	Zoom float64
	// The size of the screen in pixels.
	ScreenSize geom.Vector2I
	// The rotation of the camera in radians.
	Rotation float64
	// The rectangular bounds within which the camera can move.
	Bounds geom.Rect
}

// NewCamera returns a camera centered on the world origin with no zoom and no
// movement bounds. Call SetScreenSize once the render target size is known
// (Game.Layout keeps this in sync); set Bounds explicitly to constrain movement.
func NewCamera() *Camera {
	return &Camera{
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
func (c *Camera) WorldToScreen(worldPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (worldPos.X-c.Position.X)*c.Zoom + float64(c.ScreenSize.X)/2,
		Y: (worldPos.Y-c.Position.Y)*c.Zoom + float64(c.ScreenSize.Y)/2,
	}
}

// ScreenToWorld converts a position in screen space to world space based on the camera's position, zoom, and screen size.
func (c *Camera) ScreenToWorld(screenPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (screenPos.X-float64(c.ScreenSize.X)/2)/c.Zoom + c.Position.X,
		Y: (screenPos.Y-float64(c.ScreenSize.Y)/2)/c.Zoom + c.Position.Y,
	}
}

// Get visible rectangle in world coordinates based on the camera's position, zoom, and screen size.
func (c *Camera) ViewportBounds() geom.Rect {
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

// clamp camera position within the bounds.
func (c *Camera) ClampPosition() {
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
}

// check if a world space rectangle is within the camera's viewport.
func (c *Camera) IsWorldRectVisible(worldRect geom.Rect) bool {
	viewport := c.ViewportBounds()
	return viewport.Intersects(worldRect)
}

func (c *Camera) AspectRatio() float64 {
	return float64(c.ScreenSize.X) / float64(c.ScreenSize.Y)
}
