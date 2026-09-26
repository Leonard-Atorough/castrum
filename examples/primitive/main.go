// Command primitive demonstrates the shape drawables: a filled circle,
// an outlined rect, and a stroked line — three tints, spinning at
// fixed ticks. Shapes are pure geometry: no assets are loaded.
//
// Run from the repository root:
//
//	go run ./examples/primitive
package main

import (
	"fmt"
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
)

func main() {
	if err := run(); err != nil {
		fmt.Println("primitives: ", err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("castrum - primitives"))
	if err != nil {
		return err
	}

	// The camera frames the screen: at the center of the logical
	// resolution with zoom 1, world coordinates are screen
	// coordinates.
	if _, err := g.World().NewEntity(
		core.Camera{Zoom: 1, Primary: true},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		core.PrevTransform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	); err != nil {
		return err
	}

	// Three shapes, three tints, three styles: the circle fills, the
	// rect outlines, the line strokes — a line is drawn by its stroke
	// width, so it carries one.
	shapes := []core.Sprite{
		{
			Drawable: core.CircleShape{Radius: 50},
			Tint:     color.RGBA{R: 220, A: 255},
		},
		{
			Drawable: core.CircleShape{Radius: 50},
			Tint:     color.RGBA{R: 220, A: 255},
			Outline:     true,
			StrokeWidth: 3,
		},
		{
			Drawable:    core.RectShape{Size: geom.Vector2{X: 100, Y: 50}},
			Tint:        color.RGBA{G: 200, A: 255},
			StrokeWidth: 3,
		},
		{
			Drawable:    core.RectShape{Size: geom.Vector2{X: 100, Y: 50}},
			Tint:        color.RGBA{G: 200, A: 255},
			Outline:     true,
			StrokeWidth: 3,
		},
		{
			Drawable:    core.LineShape{To: geom.Vector2{X: 100, Y: 100}},
			Tint:        color.RGBA{B: 220, A: 255},
			Outline:     true,
			StrokeWidth: 4,
		},
	}

	positions := []geom.Vector2{
		{X: screenW / 6, Y: screenH / 2},
		{X: screenW / 3, Y: screenH / 2},
		{X: screenW / 2, Y: screenH / 2},
		{X: screenW * 2 / 3, Y: screenH / 2},
		{X: screenW * 5 / 6, Y: screenH / 2},
	}

	var sprites []*core.Entity
	for i, shape := range shapes {
		sprite, err := g.World().NewEntity(
			shape,
			core.Transform{Position: positions[i], Scale: geom.Vector2{X: 1, Y: 1}},
			core.PrevTransform{Position: positions[i]},
		)
		if err != nil {
			return err
		}
		sprites = append(sprites, sprite)
	}

	// Spin: each fixed tick advances every shape's rotation. The
	// engine's prev-transform capture and the renderer's interpolation
	// make the spin smooth at any display rate, with no per-frame
	// work here.
	if err := g.AddSystem(core.PhaseFixed, "spin", core.SystemFunc(func(ctx *core.Context) error {
		for _, sprite := range sprites {
			transform, ok := sprite.Component[core.Transform](ctx.World)
			if !ok {
				continue
			}
			transform.Rotation += rotationSpeed
			if err := sprite.SetComponent(ctx.World, transform); err != nil {
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

	// User draws run after the engine-rendered world, so this lands
	// on top of the shapes.
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, "castrum primitives — filled circle, outlined rect, stroked line")
		return nil
	})

	return runner.Run()
}
