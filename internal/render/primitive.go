package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/leonard-atorough/castrum/components"
)

// PrimitiveRenderer draws untextured shapes (rectangles, circles, lines)
// directly, so a Renderable without a TexturePath still shows up on screen.
type PrimitiveRenderer struct{}

func NewPrimitiveRenderer() *PrimitiveRenderer {
	return &PrimitiveRenderer{}
}

func (pr *PrimitiveRenderer) Draw(screen *ebiten.Image, cam components.Camera, transform components.Transform, sprite components.Sprite) error {
	pos := cam.WorldToScreen(transform.Position)
	zoom := float32(cam.Zoom)
	px, py := float32(pos.X), float32(pos.Y)
	// Origin shifts the sprite away from the entity position, matching the
	// GeoM pipeline in drawImage: the pivot (entity screen position) is the
	// rotation center, and shape geometry is offset by Origin in screen
	// space before being rotated around the pivot.
	originX := float32(transform.Origin.X) * float32(transform.Scale.X) * zoom
	originY := float32(transform.Origin.Y) * float32(transform.Scale.Y) * zoom
	clr := colorOrDefault(sprite.Color)

	w := float32(sprite.Size.X) * float32(transform.Scale.X) * zoom
	h := float32(sprite.Size.Y) * float32(transform.Scale.Y) * zoom

	sin, cos := float32(math.Sin(transform.Rotation)), float32(math.Cos(transform.Rotation))

	// rotatePoint maps a local-space offset to screen space by rotating
	// around the pivot (px, py).
	rotatePoint := func(lx, ly float32) (float32, float32) {
		return px + lx*cos - ly*sin, py + lx*sin + ly*cos
	}

	switch sprite.Primitive {
	case components.PrimitiveKindCircle:
		cx, cy := rotatePoint(-originX, -originY)
		radius := w / 2
		vector.FillCircle(screen, cx, cy, radius, clr, true)
	case components.PrimitiveKindLine:
		half := w / 2
		x0, y0 := rotatePoint(-half-originX, -originY)
		x1, y1 := rotatePoint(half-originX, -originY)
		vector.StrokeLine(screen, x0, y0, x1, y1, h, clr, true)
	case components.PrimitiveKindPolygon:
		if err := drawPolygonPath(sprite, cam, clr, screen); err != nil {
			return err
		}
	default: // PrimitiveKindRectangle
		halfW, halfH := w/2, h/2
		x0, y0 := rotatePoint(-halfW-originX, -halfH-originY)
		x1, y1 := rotatePoint(halfW-originX, -halfH-originY)
		x2, y2 := rotatePoint(halfW-originX, halfH-originY)
		x3, y3 := rotatePoint(-halfW-originX, halfH-originY)
		fillQuad(screen, x0, y0, x1, y1, x2, y2, x3, y3, clr)
	}
	return nil
}

func drawPolygonPath(sprite components.Sprite, cam components.Camera, clr color.Color, screen *ebiten.Image) error {
	if sprite.Polygon.Len() == 0 {
		return nil
	}
	if err := sprite.Polygon.Validate(); err != nil {
		return fmt.Errorf("invalid polygon")
	}
	points := sprite.Polygon.Points()
	var path vector.Path
	first := cam.WorldToScreen(points[0])
	path.MoveTo(float32(first.X), float32(first.Y))
	for _, point := range points[1:] {
		p := cam.WorldToScreen(point)
		path.LineTo(float32(p.X), float32(p.Y))
	}
	path.Close()

	var colorScale ebiten.ColorScale
	cr, cg, cb, ca := clr.RGBA()
	colorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{
		AntiAlias:  true,
		ColorScale: colorScale,
	})
	return nil
}

// fillQuad fills a quadrilateral defined by four screen-space corner points.
// Corners are pre-rotated by the caller; this function only builds and fills
// the path.
func fillQuad(screen *ebiten.Image, x0, y0, x1, y1, x2, y2, x3, y3 float32, clr color.Color) {
	var path vector.Path
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	path.LineTo(x2, y2)
	path.LineTo(x3, y3)
	path.Close()

	var colorScale ebiten.ColorScale
	cr, cg, cb, ca := clr.RGBA()
	colorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{
		AntiAlias:  true,
		ColorScale: colorScale,
	})
}

func colorOrDefault(c color.Color) color.Color {
	if c == nil {
		return color.White
	}
	return c
}
