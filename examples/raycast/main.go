// Command raycast demonstrates querying the collision system's synchronized
// collider state. Move the cursor to aim a segment from the marker; the debug
// overlay reports the hit collider without querying its entity again.
//
// Run from this directory:
//
//	go run .
package main

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

var (
	markerColor = color.RGBA{R: 240, G: 220, B: 90, A: 255}
	circleColor = color.RGBA{R: 90, G: 180, B: 240, A: 255}
	boxColor    = color.RGBA{R: 190, G: 120, B: 220, A: 255}
	missColor   = color.RGBA{R: 220, G: 100, B: 100, A: 255}
	hitColor    = color.RGBA{R: 100, G: 240, B: 130, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - raycast"),
	)
	if err != nil {
		return err
	}

	world := g.World()
	queries := world.MustResource[*collision.Queries]()
	panel := world.MustResource[*debug.Panel]()
	origin := geom.Vector2{X: -500}

	if _, err := world.NewEntity(
		core.Transform{Position: origin},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 9, Y: 9}},
			Color:    markerColor,
		},
	); err != nil {
		return err
	}

	if err := addTarget(world, geom.Vector2{X: -180, Y: -120},
		collision.Circle{Radius: 42},
		render.CircleShape{Radii: geom.Vector2{X: 42, Y: 42}},
		0, circleColor); err != nil {
		return err
	}
	if err := addTarget(world, geom.Vector2{X: 120, Y: 0},
		collision.Box{
			Min: geom.Vector2{X: -70, Y: -35},
			Max: geom.Vector2{X: 70, Y: 35},
		},
		render.RectShape{Size: geom.Vector2{X: 140, Y: 70}},
		1, boxColor); err != nil {
		return err
	}
	if err := addTarget(world, geom.Vector2{X: 340, Y: 150},
		collision.Circle{Radius: 56},
		render.CircleShape{Radii: geom.Vector2{X: 56, Y: 56}},
		0, circleColor); err != nil {
		return err
	}
	if err := addTarget(world, geom.Vector2{X: 500, Y: -100},
		collision.Box{
			Min: geom.Vector2{X: -60, Y: -40},
			Max: geom.Vector2{X: 60, Y: 40},
		},
		render.RectShape{Size: geom.Vector2{X: 120, Y: 80}},
		1, boxColor); err != nil {
		return err
	}
	if err := addTarget(world, geom.Vector2{X: 550, Y: 250},
		collision.Circle{Radius: 50},
		render.CircleShape{Radii: geom.Vector2{X: 50, Y: 50}},
		0, circleColor); err != nil {
		return err
	}

	beam, err := world.NewEntity(
		core.Transform{Position: origin},
		render.Sprite{
			Drawable:    render.LineShape{To: geom.Vector2{X: 1}},
			StrokeWidth: 3,
			Color:       missColor,
		},
	)
	if err != nil {
		return err
	}
	panel.Status("aim", "Move the cursor to raycast from the yellow marker.")

	if err := g.AddSystem(core.PhaseFixed, "raycast example", core.SystemFunc(func(ctx *core.Context) error {
		view, err := ctx.World.Resource[*render.CameraView]()
		if errors.Is(err, core.ErrResourceNotRegistered) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("raycast example: get camera view: %w", err)
		}

		end := view.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		segment := geom.Segment{Start: origin, End: end}
		hit, found := queries.Raycast(segment, collision.RaycastFilter{
			Mask: collision.Layers(0, 1),
		})

		lineEnd := end.Sub(origin)
		if lineEnd.X == 0 && lineEnd.Y == 0 {
			lineEnd.X = 1
		}
		lineColor := missColor
		if found {
			lineColor = hitColor
			panel.Status("hit", fmt.Sprintf(
				"Entity %d: %T, layers %#x, trigger %t, distance %.0f",
				hit.Entity, hit.Collider.Shape, hit.Collider.Layers,
				hit.Collider.Trigger, hit.Distance,
			))
		} else {
			panel.Status("hit", "No hit.")
		}
		return beam.SetComponent(ctx.World, render.Sprite{
			Drawable:    render.LineShape{To: lineEnd},
			StrokeWidth: 3,
			Color:       lineColor,
		})
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	return g.Run(runner)
}

func addTarget(
	world *core.World,
	position geom.Vector2,
	shape collision.ColliderShape,
	drawable render.Drawable,
	layer int,
	tint color.Color,
) error {
	collider, err := collision.NewCollider(shape)
	if err != nil {
		return err
	}
	collider.Layers = collision.Layers(layer)
	_, err = world.NewEntity(
		core.Transform{Position: position},
		collider,
		render.Sprite{Drawable: drawable, Color: tint},
	)
	return err
}
