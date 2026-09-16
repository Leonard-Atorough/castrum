package components

import (
	"fmt"
	"math"

	"github.com/leonard-atorough/castrum/geom"
)

// Camera defines a viewport for rendering. An entity with a Camera component
// and Primary set to true is used by the renderer as the active camera.
//
// Position is the camera's center in world space. Zoom is a multiplier — 1.0
// means no zoom, 2.0 doubles the apparent size. ScreenSize is the render
// target dimensions in pixels, set by the engine when the window is created
// or resized.
//
// Bounds constrains the camera's movement via ClampPosition. The default is
// unbounded (infinite in all directions). Set finite bounds to keep the
// camera within a level or region.
type Camera struct {
	Position   geom.Vector2 // camera center in world space
	Zoom        float64      // zoom multiplier (1.0 = no zoom)
	ScreenSize  geom.Vector2I // render target dimensions in pixels
	Bounds      geom.Rect    // movement bounds; unbounded by default
	Primary     bool         // marks this as the active camera for rendering
}

// NewCamera returns a camera centered on the world origin with no zoom and
// unbounded movement. ScreenSize defaults to 800x600 if either dimension is
// zero or negative.
func NewCamera(screenWidth, screenHeight uint32, primary bool) Camera {
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth = 800
		screenHeight = 600
	}
	return Camera{
		Position:   geom.Vector2{X: 0, Y: 0},
		Zoom:       1,
		ScreenSize:  geom.Vector2I{X: int(screenWidth), Y: int(screenHeight)},
		Bounds:     UnboundedRect(),
		Primary:    primary,
	}
}

// Validate checks that the camera has a positive zoom and non-zero screen
// dimensions.
func (c Camera) Validate() error {
	if c.Zoom <= 0 {
		return fmt.Errorf("camera zoom must be positive, got %v", c.Zoom)
	}
	if c.ScreenSize.X <= 0 || c.ScreenSize.Y <= 0 {
		return fmt.Errorf("camera screen size must be positive, got %v", c.ScreenSize)
	}
	return nil
}

// Serialize converts the Camera to a map suitable for blueprint serialization
// or save-game storage. Bounds are serialized as null when unbounded (infinite)
// since JSON cannot represent infinity.
func (c Camera) Serialize() (map[string]any, error) {
	data := map[string]any{
		"position": map[string]any{"x": c.Position.X, "y": c.Position.Y},
		"zoom":     c.Zoom,
		"screenSize": map[string]any{
			"x": float64(c.ScreenSize.X),
			"y": float64(c.ScreenSize.Y),
		},
		"primary": c.Primary,
	}

	if isUnbounded(c.Bounds) {
		data["bounds"] = nil
	} else {
		data["bounds"] = map[string]any{
			"min": map[string]any{"x": c.Bounds.Min.X, "y": c.Bounds.Min.Y},
			"max": map[string]any{"x": c.Bounds.Max.X, "y": c.Bounds.Max.Y},
		}
	}

	return data, nil
}

// Deserialize populates a Camera from a serialized map, returning the
// reconstructed component. If bounds are null or absent, the camera gets
// unbounded bounds. The result is validated before returning.
func (c Camera) Deserialize(data map[string]any) (Camera, error) {
	if pos, ok := data["position"].(map[string]any); ok {
		if x, ok := pos["x"].(float64); ok {
			c.Position.X = x
		}
		if y, ok := pos["y"].(float64); ok {
			c.Position.Y = y
		}
	}
	if v, ok := data["zoom"].(float64); ok {
		c.Zoom = v
	}
	if size, ok := data["screenSize"].(map[string]any); ok {
		if x, ok := size["x"].(float64); ok {
			c.ScreenSize.X = int(x)
		}
		if y, ok := size["y"].(float64); ok {
			c.ScreenSize.Y = int(y)
		}
	}
	if v, ok := data["primary"].(bool); ok {
		c.Primary = v
	}
	if bounds, ok := data["bounds"]; ok && bounds != nil {
		if boundsMap, ok := bounds.(map[string]any); ok {
			if minMap, ok := boundsMap["min"].(map[string]any); ok {
				if x, ok := minMap["x"].(float64); ok {
					c.Bounds.Min.X = x
				}
				if y, ok := minMap["y"].(float64); ok {
					c.Bounds.Min.Y = y
				}
			}
			if maxMap, ok := boundsMap["max"].(map[string]any); ok {
				if x, ok := maxMap["x"].(float64); ok {
					c.Bounds.Max.X = x
				}
				if y, ok := maxMap["y"].(float64); ok {
					c.Bounds.Max.Y = y
				}
			}
		}
	} else {
		c.Bounds = UnboundedRect()
	}
	return c, c.Validate()
}

// UnboundedRect returns a Rect with infinite bounds in all directions,
// allowing the camera to move freely.
func UnboundedRect() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: math.Inf(-1), Y: math.Inf(-1)},
		Max: geom.Vector2{X: math.Inf(1), Y: math.Inf(1)},
	}
}

// SetScreenSize updates the render target size the camera converts against.
// Uses a pointer receiver because it mutates the camera.
func (c *Camera) SetScreenSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	c.ScreenSize = geom.Vector2I{X: width, Y: height}
}

// WorldToScreen converts a position in world space to screen space.
func (c Camera) WorldToScreen(worldPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (worldPos.X-c.Position.X)*c.Zoom + float64(c.ScreenSize.X)/2,
		Y: (worldPos.Y-c.Position.Y)*c.Zoom + float64(c.ScreenSize.Y)/2,
	}
}

// ScreenToWorld converts a position in screen space to world space.
func (c Camera) ScreenToWorld(screenPos geom.Vector2) geom.Vector2 {
	return geom.Vector2{
		X: (screenPos.X-float64(c.ScreenSize.X)/2)/c.Zoom + c.Position.X,
		Y: (screenPos.Y-float64(c.ScreenSize.Y)/2)/c.Zoom + c.Position.Y,
	}
}

// ViewportBounds returns the visible rectangle in world coordinates.
func (c Camera) ViewportBounds() geom.Rect {
	halfWidth := float64(c.ScreenSize.X) / (2 * c.Zoom)
	halfHeight := float64(c.ScreenSize.Y) / (2 * c.Zoom)
	return geom.Rect{
		Min: geom.Vector2{X: c.Position.X - halfWidth, Y: c.Position.Y - halfHeight},
		Max: geom.Vector2{X: c.Position.X + halfWidth, Y: c.Position.Y + halfHeight},
	}
}

// ClampPosition returns a copy of the camera with its position adjusted so
// the viewport stays within Bounds. The receiver is not modified.
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

// IsWorldRectVisible reports whether a world-space rectangle intersects the
// camera's viewport.
func (c Camera) IsWorldRectVisible(worldRect geom.Rect) bool {
	viewport := c.ViewportBounds()
	return viewport.Intersects(worldRect)
}

// AspectRatio returns the width-to-height ratio of the camera's screen.
func (c Camera) AspectRatio() float64 {
	return float64(c.ScreenSize.X) / float64(c.ScreenSize.Y)
}

func isUnbounded(r geom.Rect) bool {
	return math.IsInf(r.Min.X, -1) || math.IsInf(r.Min.Y, -1) ||
		math.IsInf(r.Max.X, 1) || math.IsInf(r.Max.Y, 1)
}
