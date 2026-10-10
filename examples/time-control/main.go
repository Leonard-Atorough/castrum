// Command time-control demonstrates game-level time control: two bodies
// orbit a marker on simulation time, and the keyboard retimes that
// simulation. Space pauses and resumes, the arrows speed time up and slow
// it down, and the frame phase keeps running throughout - the controls
// respond even while the simulation stands still.
//
// The on-screen HUD is engine text (render.TextSource), not a runner
// overlay: it lives on entities, renders with the world, and - because
// its update runs in the frame phase - keeps showing live state while
// the simulation is paused. The debug overlay in the corner shows the
// same effect from the runner's side: fps holds steady while tps falls
// to zero on pause and tracks the scale when running.
//
// Run from the repository root:
//
//	go run ./examples/time-control
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
	screenW, screenH = 1280, 720

	goFont = "fonts/GoRegular.ttf"

	// The HUD sits above page center, clear of both the orbit - the
	// outer body's top edge reaches y≈88 at its apex - and the runner's
	// debug overlay in the top-left corner.
	controlsY = 36
	stateY    = 72

	// orbitSpeed is radians per second of simulation time, so pausing
	// stops the orbit and scaling retimes it.
	orbitSpeed = 1.5

	// The controls run on frame time, which is never scaled.
	scaleRate = 1.5
	scaleMin  = 0.25
	scaleMax  = 4
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"pause": []input.Input{
			input.KeyInput{Key: input.KeySpace},
			input.KeyInput{Key: input.KeyP},
		},
		"speed_up": []input.Input{
			input.KeyInput{Key: input.KeyArrowUp},
			input.KeyInput{Key: input.KeyW},
		},
		"slow_down": []input.Input{
			input.KeyInput{Key: input.KeyArrowDown},
			input.KeyInput{Key: input.KeyS},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum — time control"),
		castrum.WithFilesystem(files),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	// The camera frames the screen's center, so world points are screen
	// points.
	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The controls line is static text; the state line below it is
	// rewritten every frame.
	if _, err := g.World().NewEntity(
		render.Sprite{Drawable: render.TextSource{
			Font: goFont, Text: "space: pause/resume  up/down: time scale", Size: 22,
		}},
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: controlsY}, Scale: geom.Vector2{X: 1, Y: 1}},
	); err != nil {
		return err
	}
	state, err := g.World().NewEntity(
		render.Sprite{Drawable: render.TextSource{
			Font: goFont, Text: "1.00x, running", Size: 28,
		}},
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: stateY}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		return err
	}

	if _, err := g.World().NewEntity(
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 12, Y: 12}},
			Color:    color.NRGBA{R: 255, G: 200, B: 80, A: 255},
		},
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	); err != nil {
		return err
	}
	inner, err := g.World().NewEntity(
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 20, Y: 20}},
			Color:    color.NRGBA{R: 120, G: 220, B: 120, A: 255},
		},
		core.Transform{},
	)
	if err != nil {
		return err
	}
	outer, err := g.World().NewEntity(
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 32, Y: 32}},
			Color:    color.NRGBA{R: 120, G: 160, B: 220, A: 255},
		},
		core.Transform{},
	)
	if err != nil {
		return err
	}

	// The orbit advances on the fixed tick, which is simulation time: it
	// stops when the game pauses and retimes when the scale changes, with
	// no changes to this system.
	angle := 0.0
	if err := g.AddSystem(core.PhaseFixed, "orbit", core.SystemFunc(func(ctx *core.Context) error {
		angle += orbitSpeed * ctx.DeltaTime.Seconds()
		center := geom.Vector2{X: screenW / 2, Y: screenH / 2}
		innerPos := center.Add(geom.Vector2{X: 140 * math.Cos(angle), Y: 140 * math.Sin(angle)})
		outerPos := center.Add(geom.Vector2{X: 240 * math.Cos(angle/2), Y: 240 * math.Sin(angle/2)})
		if err := inner.Update(ctx.World, func(t *core.Transform) { t.Position = innerPos }); err != nil {
			return err
		}
		return outer.Update(ctx.World, func(t *core.Transform) { t.Position = outerPos })
	})); err != nil {
		return err
	}

	// The controls run in the frame phase: it never pauses, and its
	// DeltaTime is unscaled wall-clock time, so the controls feel the
	// same no matter what the simulation is doing. A fixed-phase system
	// could never resume a paused game.
	if err := g.AddSystem(core.PhaseFrame, "time control", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Actions.JustPressed("pause") {
			if g.Paused() {
				g.Resume()
			} else {
				g.Pause()
			}
		}
		if ctx.Actions.Held("speed_up") || ctx.Actions.Held("slow_down") {
			change := scaleRate * ctx.DeltaTime.Seconds()
			if !ctx.Actions.Held("speed_up") {
				change = -change
			}
			next := math.Max(scaleMin, math.Min(scaleMax, g.TimeScale()+change))
			if err := g.SetTimeScale(next); err != nil {
				return err
			}
		}

		// The state line is frame-phase text, so it reports the paused
		// simulation while the simulation itself stands still.
		word := "running"
		if g.Paused() {
			word = "paused"
		}
		return state.Update(ctx.World, func(s *render.Sprite) {
			s.Drawable = render.TextSource{
				Font: goFont,
				Text: fmt.Sprintf("%.2fx, %s", g.TimeScale(), word),
				Size: 28,
			}
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
