// Command timer demonstrates the timer component as observable
// state. Three timers run side by side: a one-shot that fires once
// and stays readable, a repeating timer that restarts itself, and a
// timer created paused and started by a key press.
//
// Timers advance on their own each fixed tick; a game never calls an
// update function on them. Completions are read straight from the
// component - [timer.Timer.JustCompleted] against the current
// tick is the completion edge, and [timer.Timer.CompletedOn] keeps
// the tick of the last fire for as long as it is useful. There are
// no events and no callbacks.
//
// Each timer is its own entity, so any number of them can run for
// the same game: this program keeps three handles and reads them.
//
// Run from the repository root:
//
//	go run ./examples/timer
package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/timer"
)

const (
	screenW, screenH = 1280, 720

	barWidth  = 220
	barHeight = 48

	// rowY places the three timer rows; labelX is where each row's
	// overlay text starts.
	rowY0, rowDY = 220, 160
	labelX       = 60

	oneShotDuration = 3 * time.Second
	pulseDuration   = 1 * time.Second
	heldDuration    = 2 * time.Second
)

var (
	// pendingColor is a timer that is not accumulating time.
	pendingColor = color.RGBA{R: 120, G: 120, B: 120, A: 255}
	// runningColor is a timer mid-countdown.
	runningColor = color.RGBA{R: 110, G: 170, B: 230, A: 255}
	// firedColor is a one-shot that has completed.
	firedColor = color.RGBA{R: 110, G: 220, B: 120, A: 255}
	// flashColor marks the single tick on which a repeating timer
	// completed.
	flashColor = color.RGBA{R: 240, G: 220, B: 90, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"restart_one_shot": {input.KeyInput{Key: input.KeyR}},
		"start_held":       {input.KeyInput{Key: input.KeySpace}},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum - timers"),
		castrum.WithBindings(bindings),
		castrum.WithTimer(),
	)
	if err != nil {
		return err
	}

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The one-shot: fires once, then stops on its entity with the
	// completion tick recorded. It stays readable forever, or until
	// Restart clears the stamp for a fresh run.
	oneShot, err := spawnBar(g.World(), rowY0, oneShotDuration, false)
	if err != nil {
		return err
	}

	// The repeating timer: completes, carries any overshoot into the
	// next interval, and starts counting again - no drift even when
	// the duration does not divide the tick evenly.
	pulse, err := spawnBar(g.World(), rowY0+rowDY, pulseDuration, true)
	if err != nil {
		return err
	}

	// The paused timer: NewTimer starts timers running, so this one
	// is paused immediately by writing the Running field - the same
	// write that pauses any timer mid-countdown. SPACE sets the
	// field back and the countdown resumes from zero elapsed time.
	held, err := spawnBar(g.World(), rowY0+2*rowDY, heldDuration, false)
	if err != nil {
		return err
	}
	if err := held.Update(g.World(), func(t *timer.Timer) { t.Running = false }); err != nil {
		return err
	}

	// One fixed system reads all three timers and reacts to their
	// state; it is the only gameplay logic in the program.
	if err := g.AddSystem(core.PhaseFixed, "timer.reactions", core.SystemFunc(func(ctx *core.Context) error {
		// R restarts the one-shot from zero: elapsed time resets and
		// the completion stamp clears, so the "fired" color reverts
		// until it fires again.
		if ctx.Actions.Pressed("restart_one_shot") {
			if err := oneShot.Update(ctx.World, func(t *timer.Timer) { t.Restart() }); err != nil {
				return err
			}
		}

		// SPACE starts the paused timer by writing Running. The
		// countdown then advances on its own.
		if ctx.Actions.Pressed("start_held") {
			if err := held.Update(ctx.World, func(t *timer.Timer) { t.Running = true }); err != nil {
				return err
			}
		}

		// The one-shot's color is state, not an event: while
		// CompletedOn is zero it has not fired; once set it stays
		// set, so the color sticks without anything remembering the
		// completion.
		oneShotColor := runningColor
		if oneShotTimer, _ := oneShot.Component[timer.Timer](ctx.World); oneShotTimer.CompletedOn != 0 {
			oneShotColor = firedColor
		}
		if err := oneShot.SetComponent(ctx.World, barSprite(oneShotColor)); err != nil {
			return err
		}

		// The repeating timer flashes for exactly the tick of its
		// completion: JustCompleted reads true this tick and false
		// the next, with nothing to reset.
		pulseColor := runningColor
		if pulseTimer, _ := pulse.Component[timer.Timer](ctx.World); pulseTimer.JustCompleted(ctx.Tick) {
			pulseColor = flashColor
		}
		if err := pulse.SetComponent(ctx.World, barSprite(pulseColor)); err != nil {
			return err
		}

		// The paused timer shows its progress in color: gray while
		// paused, blue while counting, green once fired. Fired is
		// checked first: a completed one-shot has Running forced
		// back false, so nesting the check under Running would
		// leave it gray forever.
		heldColor := pendingColor
		if heldTimer, _ := held.Component[timer.Timer](ctx.World); heldTimer.HasCompleted() {
			heldColor = firedColor
		} else if heldTimer.Running {
			heldColor = runningColor
		}
		return held.SetComponent(ctx.World, barSprite(heldColor))
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	// The overlay is read-only: everything it prints is component
	// state the systems already wrote.
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		oneShotTimer, _ := oneShot.Component[timer.Timer](ctx.World)
		pulseTimer, _ := pulse.Component[timer.Timer](ctx.World)
		heldTimer, _ := held.Component[timer.Timer](ctx.World)

		ebitenutil.DebugPrintAt(screen, describe("one-shot", oneShotTimer), labelX, rowY0-8)
		ebitenutil.DebugPrintAt(screen, describe("repeating", pulseTimer), labelX, rowY0+rowDY-8)
		ebitenutil.DebugPrintAt(screen, describe("paused", heldTimer), labelX, rowY0+2*rowDY-8)
		ebitenutil.DebugPrintAt(screen,
			"R restarts the one-shot; SPACE starts the paused timer",
			labelX, screenH-40)
		return nil
	})

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

// spawnBar places one visible bar with its own timer entity: the
// bar is the timer's display, and the timer advances itself. The
// entity carries both the sprite and the timer - the bar's position
// and color are the whole presentation.
func spawnBar(world *core.World, y float64, duration time.Duration, repeating bool) (*core.Entity, error) {
	return world.NewEntity(
		core.Transform{Position: geom.Vector2{X: barWidth / 2, Y: y}},
		barSprite(runningColor),
		timer.NewTimer(duration, repeating),
	)
}

// barSprite builds the bar's drawable rect in the given state color.
func barSprite(fill color.Color) render.Sprite {
	return render.Sprite{
		Drawable: render.RectShape{Size: geom.Vector2{X: barWidth, Y: barHeight}},
		Color:    fill,
	}
}

// describe renders one timer's state as overlay text.
func describe(label string, t timer.Timer) string {
	state := "paused"
	if t.HasCompleted() {
		state = "fired"
	} else if t.Running {
		state = "running"
	}
	return fmt.Sprintf("%s: %.1fs of %.0fs, %s, last fired tick %d",
		label, t.Elapsed.Seconds(), t.Duration.Seconds(), state, t.CompletedOn)
}
