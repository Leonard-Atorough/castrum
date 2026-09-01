package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/types"
)

// PrimitiveRenderer draws untextured shapes (rectangles, circles, lines)
// directly, so a Renderable without a TexturePath still shows up on screen.
type PrimitiveRenderer struct{}

func NewPrimitiveRenderer() *PrimitiveRenderer {
	return &PrimitiveRenderer{}
}

func (pr *PrimitiveRenderer) Draw(screen *ebiten.Image, camera *Camera, transform components.Transform, renderable components.Renderable) {
	pos := camera.WorldToScreen(transform.Position)
	x, y := float32(pos.X), float32(pos.Y)
	zoom := float32(camera.Zoom)
	clr := colorOrDefault(transform.Color)

	switch renderable.Primitive {
	case components.PrimitiveKindCircle:
		radius := float32(transform.Scale.X) * zoom / 2
		vector.FillCircle(screen, x, y, radius, clr, true)
	case components.PrimitiveKindLine:
		half := float32(transform.Scale.X) * zoom / 2
		dx, dy := float32(math.Cos(transform.Rotation)), float32(math.Sin(transform.Rotation))
		strokeWidth := float32(transform.Scale.Y) * zoom
		vector.StrokeLine(screen, x-dx*half, y-dy*half, x+dx*half, y+dy*half, strokeWidth, clr, true)
	case components.PrimitiveKindPolygon:
		// using vector.Path to draw the polygon
		ctrl := drawPolygonPath(renderable, camera, clr, screen)
		switch ctrl {
		case 1:
			break
		}
	default: // PrimitiveKindRectangle
		drawRotatedRect(screen, x, y, float32(transform.Scale.X)*zoom, float32(transform.Scale.Y)*zoom, transform.Rotation, clr)
	}
}

func drawPolygonPath(renderable components.Renderable, camera *Camera, clr color.Color, screen *ebiten.Image) int {
	if polygon, ok := renderable.Data.(*types.Polygon); ok {
		if len(polygon.Points) < 3 {
			return 1
		}
		var path vector.Path
		first := camera.WorldToScreen(polygon.Points[0])
		path.MoveTo(float32(first.X), float32(first.Y))
		for _, point := range polygon.Points[1:] {
			p := camera.WorldToScreen(point)
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
	return 0
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
