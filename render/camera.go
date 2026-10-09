package render

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Camera is the framing half of a camera entity: pair it with a
// [core.Transform], which owns the position. An entity with a Camera and
// Primary set is the camera the renderer uses; cameras move by moving
// their Transform, and interpolated rendering covers them the same as
// any other entity.
//
// castrum.New spawns a default primary camera and hands it out
// through Game.MainCamera; a user-spawned primary takes precedence
// over it.
//
// The render target's dimensions are not part of the Camera: the runner
// owns the logical resolution. Keeping a camera inside level bounds is
// a game-system concern, not component data.
type Camera struct {
	// Zoom is a multiplier on the world-to-screen scale; 1.0 means none.
	Zoom float64
	// Primary marks this camera as the one the renderer uses.
	Primary bool
}

func (c Camera) Validate() error {
	if c.Zoom <= 0 {
		return fmt.Errorf("zoom must be positive")
	}
	return nil
}

// engineCamera marks the engine's default camera, distinguishing it
// from user-spawned primaries: the collector prefers a user primary
// and falls back to this one. Unexported so only the engine attaches
// it - SpawnEngineCamera is the constructor.
type engineCamera struct{}

// SpawnEngineCamera creates the engine's default primary camera:
// zoom 1 at the world origin, marked as the engine's. The collector
// prefers any user-spawned primary camera over it.
//
// castrum.New spawns it and hands it out through Game.MainCamera;
// games never spawn it themselves. Hosts that build a world without
// castrum.New can call it to give their collector a working camera.
func SpawnEngineCamera(w *core.World) (*core.Entity, error) {
	return w.NewEntity(
		Camera{Zoom: 1, Primary: true},
		core.Transform{Position: geom.Vector2{}, Scale: geom.Vector2{X: 1, Y: 1}},
		engineCamera{},
	)
}

// CameraView is the resolved view of a camera for rendering: the
// interpolated position (previous → current by the collect alpha)
// and the current zoom - everything the blit and input picking need
// to project between world space and the screen.
type CameraView struct {
	Position geom.Vector2
	Zoom     float64
}

// WorldToScreen projects a world-space point onto the render target and snaps it to whole pixels.
//
// Pixel snapping is applied to keep the texel grid stable during rendering.
func (v CameraView) WorldToScreen(world geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	screenX := math.Round((world.X-v.Position.X)*v.Zoom + float64(screenWidth)/2)
	screenY := math.Round((world.Y-v.Position.Y)*v.Zoom + float64(screenHeight)/2)
	return geom.Vector2{X: screenX, Y: screenY}
}

// ScreenToWorld converts a screen-space point back to world coordinates.
//
// This is the inverse of WorldToScreen, without the rounding: a screen point maps back to fractional world coordinates.
func (v CameraView) ScreenToWorld(screen geom.Vector2, screenWidth, screenHeight int) geom.Vector2 {
	worldX := (screen.X-float64(screenWidth)/2)/v.Zoom + v.Position.X
	worldY := (screen.Y-float64(screenHeight)/2)/v.Zoom + v.Position.Y
	return geom.Vector2{X: worldX, Y: worldY}
}
