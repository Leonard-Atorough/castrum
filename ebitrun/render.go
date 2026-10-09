package ebitrun

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// newEngineDrawFunc returns the world renderer. [Runner] draws it before user
// [DrawFunc] callbacks so user content can appear over the world.
func newEngineDrawFunc(collector *core.Collector, provider *TextureProvider, fonts *FontProvider) DrawFunc {
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
			if item.Font != "" {
				if err := drawText(screen, fonts, item, camera); err != nil {
					return err
				}
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

			screenPos := camera.WorldToScreen(item.Position, screen.Bounds().Dx(), screen.Bounds().Dy())
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

			// ScaleWithColor applies tint and opacity together.
			if item.Color != nil {
				op.ColorScale.ScaleWithColor(item.Color)
			}

			screen.DrawImage(img, op)
		}
		return nil
	})
}

func drawShape(screen *ebiten.Image, item core.DrawItem, camera core.CameraView) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()

	switch shape := item.Shape.(type) {
	case core.RectShape:
		center := camera.WorldToScreen(item.Position, width, height)
		corners := rectCorners(center, shape.Size, item.Scale, camera.Zoom, item.Rotation)
		var path vector.Path
		path.MoveTo(float32(corners[0].X), float32(corners[0].Y))
		for _, corner := range corners[1:] {
			path.LineTo(float32(corner.X), float32(corner.Y))
		}
		path.Close()
		fillOrStrokePath(screen, &path, item.Color, item, camera.Zoom)
	case core.CircleShape:
		center := camera.WorldToScreen(item.Position, width, height)
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
		start := camera.WorldToScreen(item.Position, width, height)
		from := offsetPoint(start, shape.From, item.Scale, camera.Zoom, item.Rotation)
		to := offsetPoint(start, shape.To, item.Scale, camera.Zoom, item.Rotation)
		vector.StrokeLine(screen, float32(from.X), float32(from.Y), float32(to.X), float32(to.Y),
			float32(item.StrokeWidth*camera.Zoom), item.Color, true)
	}
}

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

// fillOrStrokePath applies the item's style to a path. ColorScale uses the
// premultiplied channels returned by color.Color.RGBA.
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

func offsetPoint(start, offset, scale geom.Vector2, zoom, rotation float64) geom.Vector2 {
	scaled := geom.Vector2{
		X: offset.X * scale.X * zoom,
		Y: offset.Y * scale.Y * zoom,
	}
	return start.Add(scaled.Rotate(rotation))
}

// drawText uses the bounds computed during collection so rendered text stays
// aligned with its measured and culled bounds.
func drawText(screen *ebiten.Image, fonts *FontProvider, item core.DrawItem, camera core.CameraView) error {
	face, err := fonts.Face(item.Font, item.TextSize)
	if err != nil {
		return err
	}

	screenPos := camera.WorldToScreen(item.Position, screen.Bounds().Dx(), screen.Bounds().Dy())
	scaleX := item.Scale.X * camera.Zoom
	scaleY := item.Scale.Y * camera.Zoom

	op := &text.DrawOptions{}
	op.GeoM.Translate(-item.TextWidth/2, -item.TextHeight/2)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(item.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)
	op.ColorScale.ScaleWithColor(item.Color)
	text.Draw(screen, item.Text, face, op)
	return nil
}
