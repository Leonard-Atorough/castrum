// Command primitive demonstrates the shape drawables as a spectrum:
// one row per shape kind - rect, circle/ellipse, line - and one
// column per variant - filled, outlined, spinning, stretched by the
// transform's scale, half-faded. The same five knobs run across every
// row, so each column reads as one knob and each row as one geometry.
// Shapes are pure geometry: no assets are loaded.
//
// Run from the repository root:
//
//	go run ./examples/primitive
package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
)

const (
	// rotationSpeed is radians per fixed tick.
	rotationSpeed = 0.05

	screenW, screenH = 1280, 720

	// The spectrum grid: cellX0 + c*cellDX places variant column c,
	// rowY0 + r*rowDY places shape row r.
	cellX0, cellDX = 200, 220
	rowY0, rowDY   = 140, 220

	// labelX is the left margin where each row's name prints.
	labelX = 20
)

// cell is one variant in the spectrum: the sprite to draw, the
// transform's scale it renders with (zero means uniform), and whether
// the spin system advances its rotation.
type cell struct {
	sprite core.Sprite
	scale  geom.Vector2
	spin   bool
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("castrum - primitives"))
	if err != nil {
		return err
	}

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	stretch := geom.Vector2{X: 0.5, Y: 1.5}

	spectrum := []struct {
		label string
		color color.Color
		cells []cell
	}{
		{
			label: "rect",
			color: color.RGBA{G: 200, A: 255},
			cells: []cell{
				{sprite: core.Sprite{Drawable: core.RectShape{Size: geom.Vector2{X: 100, Y: 50}}}},
				{sprite: core.Sprite{Drawable: core.RectShape{Size: geom.Vector2{X: 100, Y: 50}}, Outline: true, StrokeWidth: 3}},
				{sprite: core.Sprite{Drawable: core.RectShape{Size: geom.Vector2{X: 100, Y: 50}}}, spin: true},
				{sprite: core.Sprite{Drawable: core.RectShape{Size: geom.Vector2{X: 100, Y: 50}}}, scale: stretch},
				{sprite: core.Sprite{Drawable: core.RectShape{Size: geom.Vector2{X: 100, Y: 50}}, Color: color.NRGBA{G: 200, A: 128}}},
			},
		},
		{
			label: "circle",
			color: color.RGBA{R: 220, A: 255},
			cells: []cell{
				{sprite: core.Sprite{Drawable: core.CircleShape{Radii: geom.Vector2{X: 50, Y: 50}}}},
				{sprite: core.Sprite{Drawable: core.CircleShape{Radii: geom.Vector2{X: 50, Y: 50}}, Outline: true, StrokeWidth: 3}},
				// Unequal radii declare the ellipse in the shape
				// itself; the spin shows rotation is meaningful.
				{sprite: core.Sprite{Drawable: core.CircleShape{Radii: geom.Vector2{X: 80, Y: 50}}}, spin: true},
				// Equal radii stretched by the transform: the other
				// road to an ellipse, composed per axis.
				{sprite: core.Sprite{Drawable: core.CircleShape{Radii: geom.Vector2{X: 50, Y: 50}}}, scale: stretch},
				{sprite: core.Sprite{Drawable: core.CircleShape{Radii: geom.Vector2{X: 50, Y: 50}}, Color: color.NRGBA{R: 220, A: 128}}},
			},
		},
		{
			label: "line",
			color: color.RGBA{B: 220, A: 255},
			cells: []cell{
				// A zero From anchors the segment at the position.
				{sprite: core.Sprite{Drawable: core.LineShape{To: geom.Vector2{X: 80, Y: 0}}, Outline: true, StrokeWidth: 4}},
				// Symmetric endpoints pivot at the segment's middle.
				{sprite: core.Sprite{Drawable: core.LineShape{From: geom.Vector2{X: -50, Y: 0}, To: geom.Vector2{X: 50, Y: 0}}, Outline: true, StrokeWidth: 4}},
				// The anchored line spins around its endpoint.
				{sprite: core.Sprite{Drawable: core.LineShape{To: geom.Vector2{X: 80, Y: 0}}, Outline: true, StrokeWidth: 4}, spin: true},
				// Scale tilts a diagonal segment into a new slope.
				{sprite: core.Sprite{Drawable: core.LineShape{From: geom.Vector2{X: -40, Y: -40}, To: geom.Vector2{X: 40, Y: 40}}, Outline: true, StrokeWidth: 4}, scale: stretch},
				{sprite: core.Sprite{Drawable: core.LineShape{From: geom.Vector2{X: -50, Y: 0}, To: geom.Vector2{X: 50, Y: 0}}, Outline: true, StrokeWidth: 4, Color: color.NRGBA{B: 220, A: 128}}},
			},
		},
	}

	var spinners []*core.Entity
	for r, row := range spectrum {
		for c, variant := range row.cells {
			// The color's alpha channel is the drawable's opacity:
			// the half-faded column declares its own color at half
			// alpha, everyone else takes the row's hue.
			if variant.sprite.Color == nil {
				variant.sprite.Color = row.color
			}
			// A zero scale reads as unscaled; only the stretched
			// variants declare one.
			position := geom.Vector2{X: float64(cellX0 + c*cellDX), Y: float64(rowY0 + r*rowDY)}
			sprite, err := g.World().NewEntity(
				variant.sprite,
				core.Transform{Position: position, Scale: variant.scale},
			)
			if err != nil {
				return err
			}
			if variant.spin {
				spinners = append(spinners, sprite)
			}
		}
	}

	// Spin: each fixed tick advances the spinning variants' rotation.
	// The engine's prev-transform capture and the renderer's
	// interpolation make the spin smooth at any display rate, with no
	// per-frame work here.
	if err := g.AddSystem(core.PhaseFixed, "spin", core.SystemFunc(func(ctx *core.Context) error {
		for _, sprite := range spinners {
			if err := sprite.Update(ctx.World, func(t *core.Transform) { t.Rotation += rotationSpeed }); err != nil {
				return err
			}
		}
		return nil
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	// User draws run after the engine-rendered world, so the row
	// labels and the column legend land on top of the shapes.
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		for r, row := range spectrum {
			ebitenutil.DebugPrintAt(screen, row.label, labelX, rowY0+r*rowDY-8)
		}
		ebitenutil.DebugPrintAt(screen,
			"columns: filled, outlined, spinning, stretched by scale, half-faded",
			0, screenH-20)
		return nil
	})

	return runner.Run()
}
