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

			// The color carries the fade: ScaleWithColor feeds its
			// alpha-premultiplied channels straight into the color
			// scale, so the color's alpha scales hue and coverage
			// together. A nil color leaves the sprite untouched and
			// fully opaque.
			if item.Color != nil {
				op.ColorScale.ScaleWithColor(item.Color)
			}

			screen.DrawImage(img, op)
		}
		return nil
	})
}

// worldToScreen projects a world-space point onto the render target
// and snaps it to whole pixels: (world - camera) × zoom, centered at
// the target's midpoint, then rounded. The snap is the pixel-grid
// policy: interpolated positions are fractional, and fractional screen
// positions make nearest-filtered texel edges wobble frame to frame
// (the pixel-art shimmer); whole-pixel motion quantizes to 1px steps -
// invisible at display rate - and keeps the texel grid stable whenever
// zoom × scale is an integer. It is the projection seam - pure, with
func worldToScreen(world geom.Vector2, camera core.CameraView, screenWidth, screenHeight int) geom.Vector2 {
	screenX := math.Round((world.X-camera.Position.X)*camera.Zoom + float64(screenWidth)/2)
	screenY := math.Round((world.Y-camera.Position.Y)*camera.Zoom + float64(screenHeight)/2)
	return geom.Vector2{X: screenX, Y: screenY}
}

// drawShape blits one shape item: geometry through the pure projection
// helpers, filled or outlined by the item's style, into the vector
// package. Pixel reads are impossible headless, so the helpers carry
// the math and this stays a smoke-tested thin layer. Rects and circles
// go through the path API so any rotation - and, for circles, any
// non-uniform scale - renders with one code path; lines use the
// dedicated vector call.
func drawShape(screen *ebiten.Image, item core.DrawItem, camera core.CameraView) {
	// Shape items always carry a concrete color - the collector
	// defaults nil to black - and its alpha channel is the shape's
	// opacity, the same contract the sprite path has.
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
		fillOrStrokePath(screen, &path, item.Color, item, camera.Zoom)
	case core.CircleShape:
		center := worldToScreen(item.Position, camera, width, height)
		points := ellipsePoints(center, shape.Radii, item.Scale, camera.Zoom, item.Rotation)
		var path vector.Path
		path.MoveTo(float32(points[0].X), float32(points[0].Y))
		for q := 1; q < 12; q += 3 {
			anchor := (q + 2) % 12
			path.CubicTo(
				float32(points[q].X), float32(points[q].Y),
				float32(points[q+1].X), float32(points[q+1].Y),
				float32(points[anchor].X), float32(points[anchor].Y))
		}
		path.Close()
		fillOrStrokePath(screen, &path, item.Color, item, camera.Zoom)
	case core.LineShape:
		start := worldToScreen(item.Position, camera, width, height)
		from := offsetPoint(start, shape.From, item.Scale, camera.Zoom, item.Rotation)
		to := offsetPoint(start, shape.To, item.Scale, camera.Zoom, item.Rotation)
		vector.StrokeLine(screen, float32(from.X), float32(from.Y), float32(to.X), float32(to.Y),
			float32(item.StrokeWidth*camera.Zoom), item.Color, true)
	}
}

// rectCorners returns a rect's four screen-space corners around its
// projected center, clockwise from top-left: half the size, scaled by
// the item's scale and the camera zoom, rotated by the item's
// rotation. The center is snapped once and the corners stay rigid, so
// the rect never wobbles from per-corner rounding.
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

// ellipsePoints returns a circle's twelve screen-space Bézier points
// around its projected center: four anchor/control-point triples
// tracing the quarter arcs (anchor, two controls, repeating), with the
// radii scaled by the item's X/Y scale and the camera zoom, rotated by
// the item's rotation. Equal radii draw a circle; unequal radii draw an
// ellipse through the same Béziers - the geometry the shape declares
// and the transform's scale compose. The center is snapped once and
// the points stay rigid, mirroring the rect's anti-wobble policy.
func ellipsePoints(center, radii, scale geom.Vector2, zoom, rotation float64) [12]geom.Vector2 {
	k := 4 * (math.Sqrt(2) - 1) / 3

	rx := radii.X * math.Abs(scale.X) * zoom
	ry := radii.Y * math.Abs(scale.Y) * zoom

	points := [12]geom.Vector2{
		{X: rx, Y: 0},
		{X: rx, Y: k * ry},
		{X: k * rx, Y: ry},
		{X: 0, Y: ry},
		{X: -k * rx, Y: ry},
		{X: -rx, Y: k * ry},
		{X: -rx, Y: 0},
		{X: -rx, Y: -k * ry},
		{X: -k * rx, Y: -ry},
		{X: 0, Y: -ry},
		{X: k * rx, Y: -ry},
		{X: rx, Y: -k * ry},
	}

	for i, p := range points {
		points[i] = center.Add(p.Rotate(rotation))
	}
	return points
}

// fillOrStrokePath finishes a path-backed shape: the color fed
// straight to ScaleWithColor, then stroked or filled by the item's
// style, with the stroke width zoomed to screen space. Rects and
// circles share it as their path-API exit. The color must be a valid
// alpha-premultiplied color.Color - every color.Color is, so long as
// color.RGBA values carry premultiplied channels - because
// ScaleWithColor uses the channels as premultiplied scale factors.
func fillOrStrokePath(screen *ebiten.Image, path *vector.Path, clr color.Color, item core.DrawItem, zoom float64) {
	opts := &vector.DrawPathOptions{AntiAlias: true}
	opts.ColorScale.ScaleWithColor(clr)
	if item.Outline {
		vector.StrokePath(screen, path,
			&vector.StrokeOptions{Width: float32(item.StrokeWidth * zoom)}, opts)
		return
	}
	vector.FillPath(screen, path, &vector.FillOptions{}, opts)
}

// offsetPoint maps a drawable-relative offset to screen space,
// rigidly from the snapped position: the offset scaled, zoomed, and
// rotated. A line's two endpoints both map through it. Pure.
func offsetPoint(start, offset, scale geom.Vector2, zoom, rotation float64) geom.Vector2 {
	scaled := geom.Vector2{
		X: offset.X * scale.X * zoom,
		Y: offset.Y * scale.Y * zoom,
	}
	return start.Add(scaled.Rotate(rotation))
}
