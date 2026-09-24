package ebitrun

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/hajimehoshi/ebiten/v2"
)

// newSceneDrawFunc returns a closure: Collect(ctx) (wrap its error as scene draw: %w — that's fail-fast surfacing through r.drawErr → Update)
func newSceneDrawFunc(collector *core.Collector, provider *TextureProvider) DrawFunc {
	return DrawFunc(func(ctx *core.Context, screen *ebiten.Image) error {
		scene, err := collector.Collect(ctx)
		if err != nil {
			return fmt.Errorf("scene draw: %w", err)
		}
		// Use the scene and provider to perform drawing here.
		return nil
	})
}

func worldToScreen(world geom.Vector2, camera core.CameraView, screenWidth, screenHeight int) geom.Vector2 {
	screenX := (world.X-camera.Position.X)*camera.Zoom + float64(screenWidth)/2
	screenY := (world.Y-camera.Position.Y)*camera.Zoom + float64(screenHeight)/2
	return geom.Vector2{X: screenX, Y: screenY}
}

func screenToWorld(screen geom.Vector2, camera core.CameraView, screenWidth, screenHeight int) geom.Vector2 {
	worldX := (screen.X-float64(screenWidth)/2)/camera.Zoom + camera.Position.X
	worldY := (screen.Y-float64(screenHeight)/2)/camera.Zoom + camera.Position.Y
	return geom.Vector2{X: worldX, Y: worldY}
}
