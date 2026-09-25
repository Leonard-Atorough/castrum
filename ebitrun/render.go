package ebitrun

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
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
			if item.Shape != nil {
				drawShape(screen, item, camera)
				continue
			}
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

			// Tint and transparency are both multiplicative. RGBA is
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
			op.ColorScale.ScaleAlpha(1 - item.Transparency)

			screen.DrawImage(img, op)
		}
		return nil
	})
}

// worldToScreen projects a world-space point onto the render target
// and snaps it to whole pixels: (world − camera) × zoom, centered at
// the target's midpoint, then rounded. The snap is the pixel-grid
// policy: interpolated positions are fractional, and fractional screen
// positions make nearest-filtered texel edges wobble frame to frame
// (the pixel-art shimmer); whole-pixel motion quantizes to 1px steps -
// invisible at display rate - and keeps the texel grid stable whenever
// zoom × scale is an integer. It is the projection seam - pure, with
// no ebiten types, so the camera math is testable headless.
func worldToScreen(world geom.Vector2, camera core.CameraView, screenWidth, screenHeight int) geom.Vector2 {
	screenX := math.Round((world.X-camera.Position.X)*camera.Zoom + float64(screenWidth)/2)
	screenY := math.Round((world.Y-camera.Position.Y)*camera.Zoom + float64(screenHeight)/2)
	return geom.Vector2{X: screenX, Y: screenY}
}

// drawShape blits one shape item: geometry through the pure projection
// helpers, filled or outlined by the item's style, into the vector
// package. Pixel reads are impossible headless, so the helpers carry
// the math and this stays a smoke-tested thin layer. Rects go through
// the path API so any rotation renders with one code path; circles and
// lines use the dedicated vector calls.
func drawShape(screen *ebiten.Image, item core.DrawItem, camera core.CameraView) {
	clr := shapeColor(item.Tint, item.Transparency)
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()

	switch shape := item.Shape.(type) {
	case core.RectShape:
		center := worldToScreen(item.Position, camera, width, height)
		corners := rectCorners(center, shape.Size, item.Scale, camera.Zoom, item.Rotation)
		var path vector.Path
		path.MoveTo(float32(corners[0].X), float32(corners[0].Y))
		for _, corner := range corners[1:] {
			path.LineTo(float32(corner.X), float32(corner.Y))
		}
		path.Close()
		opts := &vector.DrawPathOptions{AntiAlias: true}
		opts.ColorScale.Scale(
			float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255)
		if item.Outline {
			vector.StrokePath(screen, &path,
				&vector.StrokeOptions{Width: float32(item.StrokeWidth * camera.Zoom)}, opts)
			return
		}
		vector.FillPath(screen, &path, &vector.FillOptions{}, opts)
	case core.CircleShape:
		center := worldToScreen(item.Position, camera, width, height)
		radius := circleRadius(shape.Radius, item.Scale.X, camera.Zoom)
		if item.Outline {
			vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius),
				float32(item.StrokeWidth*camera.Zoom), clr, true)
			return
		}
		vector.FillCircle(screen, float32(center.X), float32(center.Y), float32(radius), clr, true)
	case core.LineShape:
		start := worldToScreen(item.Position, camera, width, height)
		end := lineEnd(start, shape.To, item.Scale, camera.Zoom, item.Rotation)
		vector.StrokeLine(screen, float32(start.X), float32(start.Y), float32(end.X), float32(end.Y),
			float32(item.StrokeWidth*camera.Zoom), clr, true)
	}
}

// rectCorners returns a rect's four screen-space corners around its
// projected center, clockwise from top-left: half the size, scaled by
// the item's scale and the camera zoom, rotated by the item's
// rotation. The center is snapped once and the corners stay rigid, so
// the rect never wobbles from per-corner rounding. Pure - no ebiten
// types, testable headless.
func rectCorners(center, size, scale geom.Vector2, zoom, rotation float64) [4]geom.Vector2 {
	half := geom.Vector2{
		X: size.X * math.Abs(scale.X) * zoom / 2,
		Y: size.Y * math.Abs(scale.Y) * zoom / 2,
	}
	corners := [4]geom.Vector2{
		{X: -half.X, Y: -half.Y},
		{X: half.X, Y: -half.Y},
		{X: half.X, Y: half.Y},
		{X: -half.X, Y: half.Y},
	}
	for i, corner := range corners {
		corners[i] = center.Add(corner.Rotate(rotation))
	}
	return corners
}

// lineEnd computes a line's second endpoint in screen space, rigidly
// from the snapped start: the relative endpoint scaled, rotated, and
// zoomed. Pure.
func lineEnd(start, to, scale geom.Vector2, zoom, rotation float64) geom.Vector2 {
	offset := geom.Vector2{
		X: to.X * scale.X * zoom,
		Y: to.Y * scale.Y * zoom,
	}.Rotate(rotation)
	return start.Add(offset)
}

// circleRadius scales a circle to screen space. Non-uniform Y scale is
// ignored for circles: the X scale sizes the radius. Pure.
func circleRadius(radius, scaleX, zoom float64) float64 {
	return radius * math.Abs(scaleX) * zoom
}

// shapeColor composes a shape's draw color: the tint un-premultiplied,
// its alpha reduced by transparency. Shape tints are never nil - the
// collector defaults them to black. Pure.
func shapeColor(tint color.Color, transparency float32) color.RGBA {
	r, g, b, a := tint.RGBA()
	if a == 0 {
		return color.RGBA{}
	}
	return color.RGBA{
		R: uint8(uint32(r) * 255 / a),
		G: uint8(uint32(g) * 255 / a),
		B: uint8(uint32(b) * 255 / a),
		A: uint8(float64(a) * (1 - float64(transparency)) / 257),
	}
}
