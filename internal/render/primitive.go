package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
)

// PrimitiveRenderer draws untextured shapes (rectangles, circles, lines)
// directly, so a Renderable without a TexturePath still shows up on screen.
type PrimitiveRenderer struct{}

func NewPrimitiveRenderer() *PrimitiveRenderer {
	return &PrimitiveRenderer{}
}

func (pr *PrimitiveRenderer) Draw(screen *ebiten.Image, cam components.Camera, transform components.Transform, sprite components.Sprite) error {
	pos := cam.WorldToScreen(transform.Position)
	x, y := float32(pos.X), float32(pos.Y)
	zoom := float32(cam.Zoom)
	clr := colorOrDefault(sprite.Color)

	w := float32(sprite.Size.X) * float32(transform.Scale.X) * zoom
	h := float32(sprite.Size.Y) * float32(transform.Scale.Y) * zoom

	switch sprite.Primitive {
	case components.PrimitiveKindCircle:
		radius := w / 2
		vector.FillCircle(screen, x, y, radius, clr, true)
	case components.PrimitiveKindLine:
		half := w / 2
		dx, dy := float32(math.Cos(transform.Rotation)), float32(math.Sin(transform.Rotation))
		vector.StrokeLine(screen, x-dx*half, y-dy*half, x+dx*half, y+dy*half, h, clr, true)
	case components.PrimitiveKindPolygon:
		if err := drawPolygonPath(sprite, cam, clr, screen); err != nil {
			return err
		}
	default: // PrimitiveKindRectangle
		drawRotatedRect(screen, x, y, w, h, transform.Rotation, clr)
	}
	return nil
}

func drawPolygonPath(sprite components.Sprite, cam components.Camera, clr color.Color, screen *ebiten.Image) error {
	if polygon, ok := sprite.Data.(geom.Polygon); ok {
		if err := polygon.Validate(); err != nil {
			return fmt.Errorf("invalid polygon")
		}
		points := polygon.Points()
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
	}
	return nil
}

// drawRotatedRect fills a width x height rectangle centered at (cx, cy) and
// rotated by angle radians. vector.DrawFilledRect has no rotation parameter,
// so the four corners are rotated by hand into a vector.Path instead.
func drawRotatedRect(screen *ebiten.Image, cx, cy, width, height float32, angle float64, clr color.Color) {
	halfW, halfH := width/2, height/2
	sin, cos := float32(math.Sin(angle)), float32(math.Cos(angle))

	corner := func(x, y float32) (float32, float32) {
		return cx + x*cos - y*sin, cy + x*sin + y*cos
	}

	x0, y0 := corner(-halfW, -halfH)
	x1, y1 := corner(halfW, -halfH)
	x2, y2 := corner(halfW, halfH)
	x3, y3 := corner(-halfW, halfH)

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
