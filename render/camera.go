package render

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Camera defines the framing for a camera entity. Pair it with a
// [core.Transform] to set its position. The renderer uses a primary camera
// and interpolates its position like any other entity. A user-spawned primary
// takes precedence over the engine's default camera.
//
// The runner owns the logical resolution; keeping a camera within level
// bounds is the responsibility of game systems.
type Camera struct {
	// Zoom is the world-to-screen scale and must be positive.
	Zoom float64
	// Primary marks this camera as eligible for rendering.
	Primary bool
}

// Validate reports an error when the camera's zoom is zero or negative.
func (c Camera) Validate() error {
	if c.Zoom <= 0 {
		return fmt.Errorf("zoom must be positive")
	}
	return nil
}

// engineCamera distinguishes the engine's default camera from user-spawned
// primaries so the collector can use it as a fallback.
type engineCamera struct{}

// SpawnEngineCamera adds the engine's default primary camera at the world
// origin with zoom 1. The [Collector] uses it when no user primary camera is
// available.
func SpawnEngineCamera(w *core.World) (*core.Entity, error) {
	return w.NewEntity(
		Camera{Zoom: 1, Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: geom.Vector2{X: 1, Y: 1}},
		engineCamera{},
	)
}

// CameraView is a camera's resolved position and zoom for a rendered frame.
type CameraView struct {
	// Position is the interpolated camera position in world space.
	Position geom.Vector2
	// Zoom is the scale used to project between world and screen coordinates.
	Zoom float64
}

// WorldToScreen projects a world-space point to pixel coordinates relative to
// the center of a render target of the given size. It rounds each coordinate
// to the nearest whole pixel.
func (v CameraView) WorldToScreen(world geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	screenX := math.Round((world.X-v.Position.X)*v.Zoom + float64(screenWidth)/2)
	screenY := math.Round((world.Y-v.Position.Y)*v.Zoom + float64(screenHeight)/2)
	return geom.Vector2{X: screenX, Y: screenY}
}

// ScreenToWorld converts a screen-space point to world coordinates without
// rounding, preserving fractional positions.
func (v CameraView) ScreenToWorld(screen geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	worldX := (screen.X-float64(screenWidth)/2)/v.Zoom + v.Position.X
	worldY := (screen.Y-float64(screenHeight)/2)/v.Zoom + v.Position.Y
	return geom.Vector2{X: worldX, Y: worldY}
}
