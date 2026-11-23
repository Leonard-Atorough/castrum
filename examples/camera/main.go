// Command camera demonstrates camera rotation, zoom, and bounds.
// Use the arrow keys to pan, A and D to rotate, and W and S to zoom.
// The camera uses the level rectangle as its bounds. The engine clamps
// the final view to that rectangle, while Camera.MovePosition keeps the
// controls within the area where the view can move.
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/sprite"
)

const (
	goFont = builtin.Font

	// The level rectangle is also the camera's bounds. Panning stops at
	// its edges, and zooming out beyond its size centers the view.
	levelW, levelH = 1600, 900

	gridStep  = 100 // world units between reference dots
	dotRadius = 4

	edgeWidth = 4 // the bounds outline's stroke width

	panSpeed    = 600 // world units per second
	rotateSpeed = 1.5 // radians per second
	zoomRate    = 1.5 // zoom multiplier per second
	zoomMin     = 0.25
	zoomMax     = 4

	// Place the HUD above the center of the page, clear of the level
	// outline.
	controlsY = 100
	stateY    = 140
)

var (
	edgeColor = &color.RGBA{R: 255, G: 200, B: 80, A: 255}
	dotColor  = &color.RGBA{R: 120, G: 120, B: 120, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"pan_left": []input.Input{
			input.KeyInput{Key: input.KeyArrowLeft}.Binding(),
		},
		"pan_right": []input.Input{
			input.KeyInput{Key: input.KeyArrowRight}.Binding(),
		},
		"pan_up": []input.Input{
			input.KeyInput{Key: input.KeyArrowUp}.Binding(),
		},
		"pan_down": []input.Input{
			input.KeyInput{Key: input.KeyArrowDown}.Binding(),
		},
		"rotate_left": []input.Input{
			input.KeyInput{Key: input.KeyA}.Binding(),
		},
		"rotate_right": []input.Input{
			input.KeyInput{Key: input.KeyD}.Binding(),
		},
		"zoom": []input.Input{
			input.KeyPairInput{Positive: input.KeyW, Negative: input.KeyS}.Binding(),
		},
	}

	g := castrum.New(
		castrum.WithTitle("castrum — camera"),
		castrum.WithBindings(bindings),
	)
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return err
	}

	bounds := geom.RectFromMinMax(geom.Vector2{}, geom.Vector2{X: levelW, Y: levelH})
	center := bounds.Center()

	// The user camera takes precedence over the engine camera. Setting
	// bounds here lets the engine enforce the level frame automatically.
	camera, err := render.SpawnCamera(g.World(),
		render.Camera{Primary: true, Bounds: bounds}, center, 1)
	if err != nil {
		return err
	}

	// The dot grid makes camera movement visible, and the amber outline
	// shows the camera bounds.
	for x := 0; x <= levelW; x += gridStep {
		for y := 0; y <= levelH; y += gridStep {
			if _, err := g.World().NewEntity(
				core.Transform{Position: geom.Vector2{X: float64(x), Y: float64(y)}},
				sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: dotRadius, Y: dotRadius},
					Style: sprite.Style{Color: dotColor, SortOrder: -1}},
			); err != nil {
				return err
			}
		}
	}
	edges := []sprite.Primitive{
		{Shape: sprite.ShapeLine, From: geom.Vector2{}, To: geom.Vector2{X: levelW}}, // top
		{Shape: sprite.ShapeLine, From: geom.Vector2{}, To: geom.Vector2{X: levelW}}, // bottom
		{Shape: sprite.ShapeLine, From: geom.Vector2{}, To: geom.Vector2{Y: levelH}}, // left
		{Shape: sprite.ShapeLine, From: geom.Vector2{}, To: geom.Vector2{Y: levelH}}, // right
	}
	for i, pos := range []geom.Vector2{
		{X: 0, Y: 0}, {X: 0, Y: levelH}, {X: 0, Y: 0}, {X: levelW, Y: 0},
	} {
		if _, err := g.World().NewEntity(
			core.Transform{Position: pos},
			sprite.Primitive{
				Shape: edges[i].Shape, From: edges[i].From, To: edges[i].To,
				Style: sprite.Style{Outline: true, StrokeWidth: edgeWidth, Color: edgeColor},
			},
		); err != nil {
			return err
		}
	}

	// The HUD contains a static controls line and a live state line.
	if _, err := g.World().NewEntity(
		sprite.Text{Asset: goFont, Text: "arrows: pan  A/D: rotate  W/S: zoom", TextSize: 22},
		core.Transform{Position: geom.Vector2{X: center.X, Y: controlsY}, Scale: geom.Vector2{X: 1, Y: 1}},
	); err != nil {
		return err
	}
	state, err := g.World().NewEntity(
		sprite.Text{Asset: goFont, Text: "0 degrees, 1.00x", TextSize: 28},
		core.Transform{Position: geom.Vector2{X: center.X, Y: stateY}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		return err
	}

	if err := g.AddSystem(core.PhaseFixed, "camera.controls", func(ctx *core.Context) error {
		actions, _ := ctx.World.Resource[input.Actions]()
		dt := ctx.DeltaTime.Seconds()
		cam, ok := camera.Component[render.Camera](ctx.World)
		if !ok {
			return fmt.Errorf("camera controls: missing Camera component")
		}
		t, ok := camera.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("camera controls: missing Transform component")
		}

		pan := geom.Vector2{}
		if actions.Held("pan_left") {
			pan.X -= 1
		}
		if actions.Held("pan_right") {
			pan.X += 1
		}
		if actions.Held("pan_up") {
			pan.Y -= 1
		}
		if actions.Held("pan_down") {
			pan.Y += 1
		}
		displacement := geom.Vector2{}
		if !pan.IsZero() {
			displacement = pan.Normalize().Mul(panSpeed * dt).Rotate(t.Rotation)
		}
		if actions.Held("rotate_left") {
			t.Rotation -= rotateSpeed * dt
		}
		if actions.Held("rotate_right") {
			t.Rotation += rotateSpeed * dt
		}
		if actions.Held("zoom") {
			change := actions.Axis("zoom") * zoomRate * dt
			t.Scale.X = math.Max(zoomMin, math.Min(zoomMax, t.Scale.X+change))
			t.Scale.Y = t.Scale.X
		}
		position, err := cam.MovePosition(t, displacement, ctx.LogicalWidth, ctx.LogicalHeight)
		if err != nil {
			return err
		}
		t.Position = position
		return camera.SetComponent(ctx.World, t)
	}); err != nil {
		return err
	}

	// Report rotation and zoom from the same transform the renderer
	// uses to resolve the view.
	if err := g.AddSystem(core.PhaseFrame, "camera.hud", func(ctx *core.Context) error {
		transform, _ := camera.Component[core.Transform](ctx.World)
		return state.Update(ctx.World, func(x *sprite.Text) {
			x.Text = fmt.Sprintf("%.0f degrees, %.2fx",
				transform.Rotation*180/math.Pi, transform.Scale.X)
		})
	}); err != nil {
		return err
	}

	runner := ebitrun.New()

	runner.Run(g)
	return nil
}
