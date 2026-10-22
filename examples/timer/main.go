// Command timer demonstrates the timer component as observable
// state. Five timers run side by side: a one-shot that fires once
// and stays readable, a repeating timer that restarts itself, a
// timer created paused and started by a key press, and two grouped
// timers whose sets are paused and resumed together by the P key -
// the combat bar freezes while the ui bar keeps counting.
//
// Timers advance on their own each fixed tick; a game never calls an
// update function on them. Completions are read straight from the
// component - [timer.Timer.JustCompleted] against the current
// tick is the completion edge, and [timer.Timer.CompletedOn] keeps
// the tick of the last fire for as long as it is useful.
//
// Group state lives in the timer.Groups resource, provided by
// castrum.WithTimer alongside the timer system. A group pause never
// writes the timers it holds: their Running fields stay as they
// were, and only their accumulation stops.
//
// Each timer is its own entity, so any number of them can run for
// the same game: this program keeps five handles and reads them.
//
// Run from the repository root:
//
//	go run ./examples/timer
package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
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

	// rowY0 and rowDY place the five timer rows.
	rowY0, rowDY = 160, 120

	oneShotDuration = 3 * time.Second
	pulseDuration   = 1 * time.Second
	heldDuration    = 2 * time.Second

	// The grouped timers: both repeat, so a frozen group is visible
	// as one bar stuck mid-count while the other keeps filling.
	combatDuration = 2 * time.Second
	uiDuration     = 1500 * time.Millisecond

	combatGroup = "combat"
	uiGroup     = "ui"
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
		"restart_one_shot":    {input.KeyInput{Key: input.KeyR}},
		"start_held":          {input.KeyInput{Key: input.KeySpace}},
		"toggle_combat_group": {input.KeyInput{Key: input.KeyP}},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum - timers"),
		castrum.WithBindings(bindings),
		castrum.WithTimer(),
	)
	if err != nil {
		return err
	}

	// The groups resource is engine wiring: WithTimer provides it
	// before the loop starts, so MustResource is the right read.
	groups := g.World().MustResource[*timer.Groups]()

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The one-shot: fires once, then stops on its entity with the
	// completion tick recorded. It stays readable forever, or until
	// Restart clears the stamp for a fresh run.
	oneShot, err := spawnBar(g.World(), rowY0, oneShotDuration, false, "")
	if err != nil {
		return err
	}

	// The repeating timer: completes, carries any overshoot into the
	// next interval, and starts counting again - no drift even when
	// the duration does not divide the tick evenly.
	pulse, err := spawnBar(g.World(), rowY0+rowDY, pulseDuration, true, "")
	if err != nil {
		return err
	}

	// The paused timer: NewTimer starts timers running, so this one
	// is paused immediately by writing the Running field - the same
	// write that pauses any timer mid-countdown. SPACE sets the
	// field back and the countdown resumes from zero elapsed time.
	held, err := spawnBar(g.World(), rowY0+2*rowDY, heldDuration, false, "")
	if err != nil {
		return err
	}
	if err := held.Update(g.World(), func(t *timer.Timer) { t.Running = false }); err != nil {
		return err
	}

	// The grouped timers: membership is the Timer.Group field, set
	// at spawn. P pauses and resumes the combat group as a set -
	// the ui bar keeps counting through the freeze, and neither
	// grouped timer's own Running field ever changes.
	combat, err := spawnBar(g.World(), rowY0+3*rowDY, combatDuration, true, combatGroup)
	if err != nil {
		return err
	}
	ui, err := spawnBar(g.World(), rowY0+4*rowDY, uiDuration, true, uiGroup)
	if err != nil {
		return err
	}

	panel := g.World().MustResource[*debug.Panel]()

	panel.Status("controls", "R restarts the one-shot; SPACE starts the paused timer; P pauses the combat group")

	// One fixed system reads all five timers and reacts to their
	// state; it is the only gameplay logic in the program.
	if err := g.AddSystem(core.PhaseFixed, "timer.reactions", core.SystemFunc(func(ctx *core.Context) error {
		// R restarts the one-shot from zero: elapsed time resets and
		// the completion stamp clears, so the "fired" color reverts
		// until it fires again.
		if ctx.Actions.Pressed("restart_one_shot") {
			if err := oneShot.Update(ctx.World, func(t *timer.Timer) { t.Restart() }); err != nil {
				return err
			}
			panel.Message("one-shot restarted")
		}

		// SPACE starts the paused timer by writing Running. The
		// countdown then advances on its own.
		if ctx.Actions.Pressed("start_held") {
			if err := held.Update(ctx.World, func(t *timer.Timer) { t.Running = true }); err != nil {
				return err
			}
			panel.Message("paused timer started")
		}

		// P toggles the combat group. The write goes to the groups
		// resource only; the combat timer's own state is untouched.
		if ctx.Actions.Pressed("toggle_combat_group") {
			if groups.Paused(combatGroup) {
				groups.Resume(combatGroup)
				panel.Message("combat group resumed")
			} else {
				groups.Pause(combatGroup)
				panel.Message("combat group paused - the ui group keeps counting")
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
		if oneShotTimer, _ := oneShot.Component[timer.Timer](ctx.World); oneShotTimer.JustCompleted(ctx.Tick) {
			panel.Message("one-shot fired")
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
			panel.Message(fmt.Sprintf("repeating pulse completed on tick %d", ctx.Tick))
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
		if err := held.SetComponent(ctx.World, barSprite(heldColor)); err != nil {
			return err
		}

		// The combat bar grays out with its group, even though its
		// own Running field stays true - the group is the reason it
		// stopped. The ui bar never sees the combat toggle.
		combatColor := runningColor
		if combatTimer, _ := combat.Component[timer.Timer](ctx.World); combatTimer.JustCompleted(ctx.Tick) {
			combatColor = flashColor
			panel.Message(fmt.Sprintf("combat timer completed on tick %d", ctx.Tick))
		} else if groups.Paused(combatGroup) {
			combatColor = pendingColor
		}
		if err := combat.SetComponent(ctx.World, barSprite(combatColor)); err != nil {
			return err
		}
		uiColor := runningColor
		if uiTimer, _ := ui.Component[timer.Timer](ctx.World); uiTimer.JustCompleted(ctx.Tick) {
			uiColor = flashColor
			panel.Message(fmt.Sprintf("ui timer completed on tick %d", ctx.Tick))
		}
		return ui.SetComponent(ctx.World, barSprite(uiColor))
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

// spawnBar places one visible bar with its own timer entity: the
// bar is the timer's display, and the timer advances itself. The
// entity carries both the sprite and the timer - the bar's position
// and color are the whole presentation. A non-empty group puts the
// timer under that group's pause and scale state.
func spawnBar(world *core.World, y float64, duration time.Duration, repeating bool, group string) (*core.Entity, error) {
	timer := timer.NewTimer(duration, repeating)
	timer.Group = group
	return world.NewEntity(
		core.Transform{Position: geom.Vector2{X: barWidth / 2, Y: y}},
		barSprite(runningColor),
		timer,
	)
}

// barSprite builds the bar's drawable rect in the given state color.
func barSprite(fill color.Color) render.Sprite {
	return render.Sprite{
		Drawable: render.RectShape{Size: geom.Vector2{X: barWidth, Y: barHeight}},
		Color:    fill,
	}
}
