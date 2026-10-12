// Command camera demonstrates camera rotation, zoom, and bounds.
// Use the arrow keys to pan, A and D to rotate, and W and S to zoom.
// The camera uses the level rectangle as its bounds. The engine clamps
// the final view to that rectangle, while the controls keep the camera
// position within the area where the view can move.
//
// Run from the repository root:
//
//	go run ./examples/camera
package main

import (
	"embed"
	"fmt"
	"image/color"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

//go:embed fonts/GoRegular.ttf
var files embed.FS

const (
	goFont = "fonts/GoRegular.ttf"

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
	edgeColor = color.NRGBA{R: 255, G: 200, B: 80, A: 255}
	dotColor  = color.NRGBA{R: 120, G: 120, B: 120, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"pan_left": []input.Input{
			input.KeyInput{Key: input.KeyArrowLeft},
		},
		"pan_right": []input.Input{
			input.KeyInput{Key: input.KeyArrowRight},
		},
		"pan_up": []input.Input{
			input.KeyInput{Key: input.KeyArrowUp},
		},
		"pan_down": []input.Input{
			input.KeyInput{Key: input.KeyArrowDown},
		},
		"rotate_left": []input.Input{
			input.KeyInput{Key: input.KeyA},
		},
		"rotate_right": []input.Input{
			input.KeyInput{Key: input.KeyD},
		},
		"zoom": []input.Input{
			input.KeyPairInput{Positive: input.KeyW, Negative: input.KeyS},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum — camera"),
		castrum.WithFilesystem(files),
		castrum.WithBindings(bindings),
	)
	if err != nil {
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
				render.Sprite{
					Drawable:  render.CircleShape{Radii: geom.Vector2{X: dotRadius, Y: dotRadius}},
					Color:     dotColor,
					SortOrder: -1,
				},
			); err != nil {
				return err
			}
		}
	}
	edges := []render.LineShape{
		{From: geom.Vector2{}, To: geom.Vector2{X: levelW}}, // top
		{From: geom.Vector2{}, To: geom.Vector2{X: levelW}}, // bottom
		{From: geom.Vector2{}, To: geom.Vector2{Y: levelH}}, // left
		{From: geom.Vector2{}, To: geom.Vector2{Y: levelH}}, // right
	}
	for i, pos := range []geom.Vector2{
		{X: 0, Y: 0}, {X: 0, Y: levelH}, {X: 0, Y: 0}, {X: levelW, Y: 0},
	} {
		if _, err := g.World().NewEntity(
			core.Transform{Position: pos},
			render.Sprite{
				Drawable:    edges[i],
				Outline:     true,
				StrokeWidth: edgeWidth,
				Color:       edgeColor,
			},
		); err != nil {
			return err
		}
	}

	// The HUD contains a static controls line and a live state line.
	if _, err := g.World().NewEntity(
		render.Sprite{Drawable: render.TextSource{
			Font: goFont, Text: "arrows: pan  A/D: rotate  W/S: zoom", Size: 22,
		}},
		core.Transform{Position: geom.Vector2{X: center.X, Y: controlsY}, Scale: geom.Vector2{X: 1, Y: 1}},
	); err != nil {
		return err
	}
	state, err := g.World().NewEntity(
		render.Sprite{Drawable: render.TextSource{
			Font: goFont, Text: "0 degrees, 1.00x", Size: 28,
		}},
		core.Transform{Position: geom.Vector2{X: center.X, Y: stateY}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		return err
	}

	// This system updates the camera's position, rotation, and zoom.
	// The engine resolves the final view and clamps it to the camera
	// bounds; this system keeps the position within the corresponding
	// movement window.
	if err := g.AddSystem(core.PhaseFixed, "camera.controls", core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return camera.Update(ctx.World, func(t *core.Transform) {
			// Pan relative to the screen by rotating the input direction
			// into world space. Normalizing the vector keeps diagonal
			// movement at the same speed as horizontal or vertical
			// movement.
			pan := geom.Vector2{}
			if ctx.Actions.Held("pan_left") {
				pan.X -= 1
			}
			if ctx.Actions.Held("pan_right") {
				pan.X += 1
			}
			if ctx.Actions.Held("pan_up") {
				pan.Y -= 1
			}
			if ctx.Actions.Held("pan_down") {
				pan.Y += 1
			}
			if !pan.IsZero() {
				t.Position = t.Position.Add(pan.Normalize().Mul(panSpeed * dt).Rotate(t.Rotation))
			}
			if ctx.Actions.Held("rotate_left") {
				t.Rotation -= rotateSpeed * dt
			}
			if ctx.Actions.Held("rotate_right") {
				t.Rotation += rotateSpeed * dt
			}
			if ctx.Actions.Held("zoom") {
				change := ctx.Actions.Axis("zoom") * zoomRate * dt
				t.Scale.X = math.Max(zoomMin, math.Min(zoomMax, t.Scale.X+change))
				t.Scale.Y = t.Scale.X
			}

			// Keep the position inside the movement window used by the
			// engine when it clamps the view. The window is based on the
			// view's circumscribed half-extents at the current zoom and
			// rotation. If the position leaves this window, the view
			// stays pinned until the position catches up. When zooming
			// out makes the view larger than the bounds, the window is
			// empty, so both the view and position stay at the center.
			halfW := float64(ctx.LogicalWidth) / (2 * t.Scale.X)
			halfH := float64(ctx.LogicalHeight) / (2 * t.Scale.Y)
			s, c := math.Abs(math.Sin(t.Rotation)), math.Abs(math.Cos(t.Rotation))
			extentX := c*halfW + s*halfH
			extentY := s*halfW + c*halfH
			if extentX >= float64(levelW)/2 {
				t.Position.X = center.X
			} else {
				t.Position.X = clamp(t.Position.X, extentX, float64(levelW)-extentX)
			}
			if extentY >= float64(levelH)/2 {
				t.Position.Y = center.Y
			} else {
				t.Position.Y = clamp(t.Position.Y, extentY, float64(levelH)-extentY)
			}
		})
	})); err != nil {
		return err
	}

	// Report rotation and zoom from the same transform the renderer
	// uses to resolve the view.
	if err := g.AddSystem(core.PhaseFrame, "camera.hud", core.SystemFunc(func(ctx *core.Context) error {
		transform, _ := camera.Component[core.Transform](ctx.World)
		return state.Update(ctx.World, func(s *render.Sprite) {
			s.Drawable = render.TextSource{
				Font: goFont,
				Text: fmt.Sprintf("%.0f degrees, %.2fx",
					transform.Rotation*180/math.Pi, transform.Scale.X),
				Size: 28,
			}
		})
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	return g.Run(runner)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
