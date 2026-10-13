// Command tween demonstrates fixed-time movement and frame-time tint changes.
// Space pauses the simulation, 1/2/3 select half/normal/double speed, and R
// restarts both tweens. The tint continues while movement is paused.
//
// Run from the repository root:
//
//	go run ./examples/tween
package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/tween"
)

const (
	screenW, screenH = 1280, 720
	tweenDuration    = 4 * time.Second
)

type demo struct {
	motion tween.Tween
	tint   tween.Tween
	body   *core.Entity
	swatch *core.Entity
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, d, err := newGame()
	if err != nil {
		return err
	}
	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrintAt(screen,
			"space: pause/resume   1: half speed   2: normal speed   3: double speed   R: restart   Esc: quit",
			80, 100)
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("simulation: %.1fx   paused: %t", g.TimeScale(), g.Paused()), 80, 130)
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("fixed movement: cubic in-out   weight: %.3f   complete: %t",
				d.motion.Progress(), d.motion.IsComplete()), 240, 200)
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("frame tint: quadratic in-out   weight: %.3f   complete: %t",
				d.tint.Progress(), d.tint.IsComplete()), 240, 380)
		ebitenutil.DebugPrintAt(screen,
			"Both hold their endpoints. Restarted movement updates on the next fixed tick.", 240, 600)
		return nil
	})
	return g.Run(runner)
}

func newGame() (*castrum.Game, *demo, error) {
	g, err := castrum.New(
		castrum.WithTitle("castrum - tween"),
		castrum.WithBindings(input.Bindings{
			"pause":   []input.Input{input.KeyInput{Key: input.KeySpace}},
			"half":    []input.Input{input.KeyInput{Key: input.Key1}},
			"normal":  []input.Input{input.KeyInput{Key: input.Key2}},
			"double":  []input.Input{input.KeyInput{Key: input.Key3}},
			"restart": []input.Input{input.KeyInput{Key: input.KeyR}},
			"quit":    []input.Input{input.KeyInput{Key: input.KeyEscape}},
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	d := &demo{}
	d.motion, err = tween.NewTween(tweenDuration, geom.EaseInOutCubic)
	if err != nil {
		return nil, nil, err
	}
	d.tint, err = tween.NewTween(tweenDuration, geom.EaseInOutQuad)
	if err != nil {
		return nil, nil, err
	}
	if err := g.MainCamera().Update(g.World(), func(c *render.Camera) {
		c.Anchor = geom.Vector2{X: -0.5, Y: -0.5}
	}); err != nil {
		return nil, nil, err
	}

	start := geom.Vector2{X: 240, Y: 260}
	end := geom.Vector2{X: 1040, Y: 260}
	if _, err := g.World().NewEntity(
		core.Transform{Position: start},
		render.Sprite{
			Drawable:    render.LineShape{To: end.Sub(start)},
			Outline:     true,
			StrokeWidth: 2,
			Color:       color.NRGBA{R: 100, G: 100, B: 100, A: 255},
		},
	); err != nil {
		return nil, nil, err
	}
	d.body, err = g.World().NewEntity(
		core.Transform{Position: start},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 24, Y: 24}},
			Color:    color.NRGBA{R: 100, G: 220, B: 140, A: 255},
			Layer:    1,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	d.swatch, err = g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: 480}},
		render.Sprite{
			Drawable: render.RectShape{Size: geom.Vector2{X: 240, Y: 120}},
			Color:    color.NRGBA{R: 255, G: 180, B: 60, A: 255},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	if err := g.AddSystem(core.PhaseFixed, "tween.motion", core.SystemFunc(func(ctx *core.Context) error {
		weight, err := d.motion.Update(ctx.DeltaTime)
		if err != nil {
			return err
		}
		return d.body.Update(ctx.World, func(t *core.Transform) {
			t.Position = start.Lerp(end, weight)
		})
	})); err != nil {
		return nil, nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "tween.presentation", core.SystemFunc(func(ctx *core.Context) error {
		if ctx.Actions.JustPressed("pause") {
			if g.Paused() {
				g.Resume()
			} else {
				g.Pause()
			}
		}
		for _, speed := range []struct {
			action input.Action
			scale  float64
		}{
			{"half", 0.5},
			{"normal", 1},
			{"double", 2},
		} {
			if ctx.Actions.JustPressed(speed.action) {
				if err := g.SetTimeScale(speed.scale); err != nil {
					return err
				}
			}
		}
		if ctx.Actions.JustPressed("restart") {
			d.motion.Reset()
			d.tint.Reset()
		}
		if ctx.Actions.JustPressed("quit") {
			g.Quit()
		}

		// Only tint changes on frame time; transform snapshots stay on fixed ticks.
		weight, err := d.tint.Update(ctx.DeltaTime)
		if err != nil {
			return err
		}
		return d.swatch.Update(ctx.World, func(s *render.Sprite) {
			s.Color = color.NRGBA{
				R: uint8(math.Round(geom.Lerp(255, 80, weight))),
				G: uint8(math.Round(geom.Lerp(180, 160, weight))),
				B: uint8(math.Round(geom.Lerp(60, 255, weight))),
				A: 255,
			}
		})
	})); err != nil {
		return nil, nil, err
	}
	return g, d, nil
}
