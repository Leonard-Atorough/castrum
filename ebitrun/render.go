package ebitrun

import (
	"errors"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/sprite"
)

func newEngineDrawFunc(collector *render.Collector, provider *TextureProvider, fonts *FontProvider) DrawFunc {
	return DrawFunc(func(ctx *core.Context, screen *ebiten.Image, alpha float64) error {
		list, err := collector.Collect(ctx, alpha)
		if err != nil {
			return err
		}
		return drawList(screen, list, provider, fonts)
	})
}

func drawList(screen *ebiten.Image, list render.DrawList, provider *TextureProvider, fonts *FontProvider) error {
	var errs []error
	for _, item := range list.Items {
		if err := drawItem(screen, list.Camera, item, provider, fonts); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func drawItem(screen *ebiten.Image, camera render.CameraView, item render.DrawItem, provider *TextureProvider, fonts *FontProvider) error {
	switch p := item.Payload().(type) {
	case render.TextureDraw:
		return drawTexture(screen, camera, item, p, provider)
	case render.TextDraw:
		return drawText(screen, camera, item, p, fonts)
	case render.ShapeDraw:
		return drawShape(screen, camera, item, p)
	default:
		return fmt.Errorf("ebitrun: unknown draw item payload type: %T", p)
	}
}

func drawTexture(screen *ebiten.Image, camera render.CameraView, item render.DrawItem, p render.TextureDraw, provider *TextureProvider) error {
	var img *ebiten.Image
	var err error
	if p.Rect.Empty() {
		img, err = provider.Texture(p.Texture)
	} else {
		img, err = provider.SubImageRect(p.Texture, p.Rect)
	}
	if err != nil {
		return err
	}

	screenPos := camera.WorldToScreen(item.Position, screen.Bounds().Dx(), screen.Bounds().Dy())
	scaleX := item.Scale.X * camera.Zoom
	scaleY := item.Scale.Y * camera.Zoom
	if p.FlipH {
		scaleX = -scaleX
	}
	if p.FlipV {
		scaleY = -scaleY
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(img.Bounds().Dx())/2, -float64(img.Bounds().Dy())/2)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(item.Rotation - camera.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	// ScaleWithColor applies tint and opacity together.
	if item.Color != nil {
		op.ColorScale.ScaleWithColor(item.Color)
	}

	screen.DrawImage(img, op)
	return nil
}

func drawShape(screen *ebiten.Image, camera render.CameraView, item render.DrawItem, p render.ShapeDraw) error {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	rotation := item.Rotation - camera.Rotation

	switch shape := p.Shape.(type) {
	case sprite.RectShape:
		center := camera.WorldToScreen(item.Position, width, height)
		corners := rectCorners(center, shape.Size, item.Scale, camera.Zoom, rotation)
		var path vector.Path
		path.MoveTo(float32(corners[0].X), float32(corners[0].Y))
		for _, corner := range corners[1:] {
			path.LineTo(float32(corner.X), float32(corner.Y))
		}
		path.Close()
		fillOrStrokePath(screen, &path, item.Color, p, camera.Zoom)
	case sprite.CircleShape:
		center := camera.WorldToScreen(item.Position, width, height)
		points := ellipsePoints(center, shape.Radii, item.Scale, camera.Zoom, rotation)
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
		fillOrStrokePath(screen, &path, item.Color, p, camera.Zoom)
	case sprite.LineShape:
		start := camera.WorldToScreen(item.Position, width, height)
		from := offsetPoint(start, shape.From, item.Scale, camera.Zoom, rotation)
		to := offsetPoint(start, shape.To, item.Scale, camera.Zoom, rotation)
		vector.StrokeLine(screen, float32(from.X), float32(from.Y), float32(to.X), float32(to.Y),
			float32(p.StrokeWidth*camera.Zoom), item.Color, true)
	default:
		return fmt.Errorf("ebitrun: unknown shape type: %T", p.Shape)
	}
	return nil
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

func fillOrStrokePath(screen *ebiten.Image, path *vector.Path, clr color.Color, p render.ShapeDraw, zoom float64) {
	opts := &vector.DrawPathOptions{AntiAlias: true}
	opts.ColorScale.ScaleWithColor(clr)
	if p.Outline {
		vector.StrokePath(screen, path,
			&vector.StrokeOptions{Width: float32(p.StrokeWidth * zoom)}, opts)
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

func drawText(screen *ebiten.Image, camera render.CameraView, item render.DrawItem, p render.TextDraw, fonts *FontProvider) error {

	face, err := fonts.Face(p.Font, p.Size)
	if err != nil {
		return err
	}

	screenPos := camera.WorldToScreen(item.Position, screen.Bounds().Dx(), screen.Bounds().Dy())
	scaleX := item.Scale.X * camera.Zoom
	scaleY := item.Scale.Y * camera.Zoom

	op := &text.DrawOptions{}
	op.GeoM.Translate(-p.Bounds.X/2, -p.Bounds.Y/2)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(item.Rotation - camera.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)
	op.ColorScale.ScaleWithColor(item.Color)
	text.Draw(screen, p.Text, face, op)
	return nil
}
