package ebitrun

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/hajimehoshi/ebiten/v2"
)

// newEngineDrawFunc builds the engine's world renderer: it collects
// the frame's DrawList and blits each item through the provider,
// projecting world space onto the logical screen. The runner draws it
// before any user DrawFunc, so overlays land on top of the world. Its
// errors are already attributed (collect and provider errors name
// their handles) and propagate as the frame's draw error.
func newEngineDrawFunc(collector *core.Collector, provider *TextureProvider) DrawFunc {
	return DrawFunc(func(ctx *core.Context, screen *ebiten.Image) error {
		list, err := collector.Collect(ctx)
		if err != nil {
			return err
		}

		camera := list.Camera
		for _, item := range list.Items {
			var img *ebiten.Image
			if item.Rect.Empty() {
				img, err = provider.Texture(item.Texture)
			} else {
				img, err = provider.SubImageRect(item.Texture, item.Rect)
			}
			if err != nil {
				return err
			}

			screenPos := worldToScreen(item.Position, camera, screen.Bounds().Dx(), screen.Bounds().Dy())
			scaleX := item.Scale.X * camera.Zoom
			scaleY := item.Scale.Y * camera.Zoom
			if item.FlipH {
				scaleX = -scaleX
			}
			if item.FlipV {
				scaleY = -scaleY
			}

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-float64(img.Bounds().Dx())/2, -float64(img.Bounds().Dy())/2)
			op.GeoM.Scale(scaleX, scaleY)
			op.GeoM.Rotate(item.Rotation)
			op.GeoM.Translate(screenPos.X, screenPos.Y)

			// Tint × opacity, both multiplicative. RGBA is
			// alpha-premultiplied, so divide the channels out: the
			// tint multiplies by hue alone, not by the tint color's
			// own alpha. A nil or fully transparent tint carries no
			// hue and leaves the sprite untinted.
			if item.Tint != nil {
				r, g, b, a := item.Tint.RGBA()
				if a > 0 {
					op.ColorScale.Scale(float32(r)/float32(a), float32(g)/float32(a), float32(b)/float32(a), 1)
				}
			}
			op.ColorScale.ScaleAlpha(item.Opacity)

			screen.DrawImage(img, op)
		}
		return nil
	})
}

// worldToScreen projects a world-space point onto the render target:
// (world − camera) × zoom, centered at the target's midpoint. It is
// the projection seam - pure, with no ebiten types, so the camera math
// is testable headless.
func worldToScreen(world geom.Vector2, camera core.CameraView, screenWidth, screenHeight int) geom.Vector2 {
	screenX := (world.X-camera.Position.X)*camera.Zoom + float64(screenWidth)/2
	screenY := (world.Y-camera.Position.Y)*camera.Zoom + float64(screenHeight)/2
	return geom.Vector2{X: screenX, Y: screenY}
}
