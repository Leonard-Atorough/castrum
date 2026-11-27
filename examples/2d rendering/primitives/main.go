// Command primitive demonstrates shape drawables without loading assets.
//
// Rows show rectangles, circles or ellipses, and lines. Columns show filled,
// outlined, spinning, stretched, and half-faded variants.
//
// Run from this directory:
//
//	go run .
package main

import (
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

const (
	// rotationSpeed is radians per fixed tick.
	rotationSpeed = 0.05

	screenW, screenH = 1280, 720

	cellX0, cellDX = 200, 220
	rowY0, rowDY   = 140, 220
)

type cell struct {
	s     sprite.Primitive
	scale geom.Vector2
	spin  bool
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}

	ebitrun.New(ebitrun.WithDebugOverlay()).Run(g)
	return nil
}

func newGame() (*castrum.Game, error) {
	g := castrum.New(castrum.WithTitle("castrum - primitives"))

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return nil, err
	}

	stretch := geom.Vector2{X: 0.5, Y: 1.5}

	spectrum := []struct {
		color *color.RGBA
		cells []cell
	}{
		{
			color: &color.RGBA{G: 200, A: 128},
			cells: []cell{
				{s: sprite.Primitive{Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 100, Y: 50}}},
				{s: sprite.Primitive{Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 100, Y: 50}, Style: sprite.Style{Outline: true, StrokeWidth: 3}}},
				{s: sprite.Primitive{Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 100, Y: 50}}, spin: true},
				{s: sprite.Primitive{Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 100, Y: 50}}, scale: stretch},
				{s: sprite.Primitive{Shape: sprite.ShapeRect, Rect: geom.Vector2{X: 100, Y: 50}, Style: sprite.Style{Color: &color.RGBA{G: 200, A: 64}}}},
			},
		},
		{
			color: &color.RGBA{R: 220, A: 128},
			cells: []cell{
				{s: sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 50, Y: 50}}},
				{s: sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 50, Y: 50}, Style: sprite.Style{Outline: true, StrokeWidth: 3}}},
				// Unequal radii make an ellipse.
				{s: sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 80, Y: 50}}, spin: true},
				// Stretched by transform.
				{s: sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 50, Y: 50}}, scale: stretch},
				{s: sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: 50, Y: 50}, Style: sprite.Style{Color: &color.RGBA{R: 220, A: 64}}}},
			},
		},
		{
			color: &color.RGBA{B: 220, A: 128},
			cells: []cell{
				// A zero From anchors the segment at the position.
				{s: sprite.Primitive{Shape: sprite.ShapeLine, To: geom.Vector2{X: 80, Y: 0}, Style: sprite.Style{Outline: true, StrokeWidth: 4}}},
				// Symmetric endpoints pivot at the segment's middle.
				{s: sprite.Primitive{Shape: sprite.ShapeLine, From: geom.Vector2{X: -50, Y: 0}, To: geom.Vector2{X: 50, Y: 0}, Style: sprite.Style{Outline: true, StrokeWidth: 4}}},
				// The anchored line spins around its endpoint.
				{s: sprite.Primitive{Shape: sprite.ShapeLine, To: geom.Vector2{X: 80, Y: 0}, Style: sprite.Style{Outline: true, StrokeWidth: 4}}, spin: true},
				// Scale tilts a diagonal segment into a new slope.
				{s: sprite.Primitive{Shape: sprite.ShapeLine, From: geom.Vector2{X: -40, Y: -40}, To: geom.Vector2{X: 40, Y: 40}, Style: sprite.Style{Outline: true, StrokeWidth: 4}}, scale: stretch},
				{s: sprite.Primitive{Shape: sprite.ShapeLine, From: geom.Vector2{X: -50, Y: 0}, To: geom.Vector2{X: 50, Y: 0}, Style: sprite.Style{Outline: true, StrokeWidth: 4, Color: &color.RGBA{B: 110, A: 64}}}},
			},
		},
	}

	var spinners []core.Entity
	for r, row := range spectrum {
		for c, variant := range row.cells {
			// Set row color if none.
			if variant.s.Style.Color == nil {
				variant.s.Style.Color = row.color
			}
			position := geom.Vector2{X: float64(cellX0 + c*cellDX), Y: float64(rowY0 + r*rowDY)}
			sprite, err := g.World().NewEntity(
				variant.s,
				core.Transform{Position: position, Scale: variant.scale},
			)
			if err != nil {
				return nil, err
			}
			if variant.spin {
				spinners = append(spinners, sprite)
			}
		}
	}

	if err := g.AddSystem(core.PhaseFixed, "spin", func(ctx *core.Context) error {
		for _, sprite := range spinners {
			if err := sprite.Update(ctx.World, func(t *core.Transform) { t.Rotation += rotationSpeed }); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	panel := g.World().MustResource[*debug.Panel]()
	panel.Message("rows: rect, circle, line")
	panel.Message("columns: filled, outlined, spinning, stretched by scale, half-faded")

	return g, nil
}
