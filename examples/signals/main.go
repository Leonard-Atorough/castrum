// Command signals demonstrates reacting to an existing signal and
// authoring new ones, as one repeating cycle of a small launch
// sequence.
//
// The engine half: an injector timer repeats once a second. The
// charge system reads the engine's [timer.Completed] signals and
// grows the core by one step per injection while charging - no system
// reads a Timer field to detect a completion, and nothing polls
// JustCompleted.
//
// The game half: two game-defined signals carry the sequence across
// its transitions.
//
//   - coreCharged is emitted the moment the core reaches full
//     charge. It is caught in the same tick by the later-registered
//     sequence system, which arms the launch countdown, and one
//     display frame later by the frame annunciator, which logs the
//     transition to the debug panel.
//   - launched is emitted when the countdown timer completes: the
//     charge system translates that completion into the game's own
//     vocabulary, and the flight system flies the core while the
//     annunciator logs the launch.
//
// Each custom signal has two independent readers, one per phase, and
// both observe it exactly once from a single emission. When the core
// clears the screen, the sequence resets and the cycle begins again.
//
// Run from the repository root:
//
//	go run ./examples/signals
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
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/timer"
)

const (
	screenW, screenH = 1280, 720

	injectorW, injectorH = 220, 48
	injectorY            = 620

	coreX, coreY = 640, 320

	maxCharge   = 5
	coreBase    = 80
	chargeStep  = 24
	flashTicks  = 5
	flightSpeed = 14

	injectInterval = time.Second
	countdownLen   = 3 * time.Second

	// ticksPerHalfSecond drives the countdown blink.
	ticksPerHalfSecond = 30
)

var (
	runningColor = color.RGBA{R: 110, G: 170, B: 230, A: 255}
	flashColor   = color.RGBA{R: 240, G: 220, B: 90, A: 255}
	chargedColor = color.RGBA{R: 240, G: 200, B: 60, A: 255}
	countdownOn  = color.RGBA{R: 230, G: 90, B: 90, A: 255}
	countdownOff = color.RGBA{R: 120, G: 50, B: 50, A: 255}
)

// coreCharged is the game's own signal for the charge reaching full:
// a plain struct, no registration. It names a transition in the game's
// vocabulary that the engine has no concept of.
type coreCharged struct{}

// launched names the countdown finishing and the core leaving. It is a
// translation of the countdown timer's completion into the game's own
// terms.
type launched struct {
	tick uint64
}

type phase int

const (
	charging phase = iota
	countdown
	flying
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(
		castrum.WithTitle("castrum - signals"),
		castrum.WithTimer(),
	)
	if err != nil {
		return err
	}

	// The signal hub and debug panel are engine wiring, like the timer
	// groups: MustResource is the right read for resources castrum.New
	// guarantees.
	hub := g.World().MustResource[*signal.Hub]()
	panel := g.World().MustResource[*debug.Panel]()

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	injector, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: injectorW / 2, Y: injectorY}},
		barSprite(runningColor),
		timer.NewTimer(injectInterval, true),
	)
	if err != nil {
		return err
	}

	// NewTimer starts running, so stop the countdown before the game loop.
	// The sequence system starts it when the core charges.
	countdownTimer, err := g.World().NewEntity(timer.NewTimer(countdownLen, false))
	if err != nil {
		return err
	}
	if err := countdownTimer.Update(g.World(), func(t *timer.Timer) { t.Running = false }); err != nil {
		return err
	}

	coreEntity, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: coreX, Y: coreY}},
		coreSprite(coreBase, runningColor),
	)
	if err != nil {
		return err
	}

	// The charge system consumes one stream and emits two game signals,
	// which still fits hub.Consumer. The annunciator consumes two streams
	// in one update, so it uses explicit readers.
	chargedFrame := hub.NewReader[coreCharged]()
	launchedFrame := hub.NewReader[launched]()

	// Signal-driven state. No value here is written by polling a timer
	// component; every write traces back to a consumed signal, and the
	// paint system's use of the state is rendering only.
	state := &sequence{
		phase:     charging,
		injector:  injector,
		countdown: countdownTimer,
		core:      coreEntity,
	}

	// The charge system links to the engine's existing signal: every
	// injector completion pumps one charge step into the core, and the
	// countdown's completion is translated into the launched signal.
	if err := g.AddSystem(core.PhaseFixed, "core.charge", hub.Consumer(func(ctx *core.Context, c timer.Completed) error {
		switch c.Entity {
		case injector.ID():
			state.flash = flashTicks
			state.injections++
			if state.phase == charging {
				state.charge++
				panel.Message(fmt.Sprintf("tick %d: charge %d/%d", ctx.Tick, state.charge, maxCharge))
				if state.charge >= maxCharge {
					hub.Emit(coreCharged{})
				}
			}
		case countdownTimer.ID():
			if state.phase == countdown {
				hub.Emit(launched{tick: ctx.Tick})
			}
		}
		return nil
	})); err != nil {
		return err
	}

	// The sequence system catches coreCharged in the same tick it was
	// emitted - it is registered after the charge system, so the signal
	// is already in the tick window - and arms the countdown.
	if err := g.AddSystem(core.PhaseFixed, "core.sequence", hub.Consumer(func(ctx *core.Context, c coreCharged) error {
		state.phase = countdown
		return state.countdown.Update(ctx.World, func(t *timer.Timer) { t.Running = true })
	})); err != nil {
		return err
	}

	// The flight system catches launched the same tick and starts the
	// core moving; the paint system renders the motion.
	if err := g.AddSystem(core.PhaseFixed, "core.flight", hub.Consumer(func(_ *core.Context, l launched) error {
		state.phase = flying
		return nil
	})); err != nil {
		return err
	}

	// The paint system renders signal-driven state. Registered last
	// among the fixed systems, it also owns the reset: when the flown
	// core clears the screen, the sequence starts a fresh cycle.
	if err := g.AddSystem(core.PhaseFixed, "core.paint", core.SystemFunc(func(ctx *core.Context) error {
		if state.flash > 0 {
			state.flash--
		}
		if err := state.injector.SetComponent(ctx.World, barSprite(state.flashColor())); err != nil {
			return err
		}

		size := coreBase + state.charge*chargeStep
		fill := runningColor
		switch state.phase {
		case countdown:
			// The blink is presentation: paint may read state and tick
			// freely; it never detects a transition.
			if (ctx.Tick/ticksPerHalfSecond)%2 == 0 {
				fill = countdownOn
			} else {
				fill = countdownOff
			}
		case flying:
			fill = chargedColor
			if err := state.core.Update(ctx.World, func(t *core.Transform) { t.Position.Y -= flightSpeed }); err != nil {
				return err
			}
		}
		if err := state.core.SetComponent(ctx.World, coreSprite(float64(size), fill)); err != nil {
			return err
		}

		// Reset: the core has left the screen, so charge and countdown
		// return to their starting state and the cycle repeats.
		transform, _ := state.core.Component[core.Transform](ctx.World)
		if state.phase == flying && transform.Position.Y < -float64(size) {
			state.phase = charging
			state.charge = 0
			if err := state.core.SetComponent(ctx.World, core.Transform{Position: geom.Vector2{X: coreX, Y: coreY}}); err != nil {
				return err
			}
			return state.countdown.Update(ctx.World, func(t *timer.Timer) { t.Restart(); t.Running = false })
		}
		return nil
	})); err != nil {
		return err
	}

	// The annunciator is a frame system: custom signals emitted during
	// fixed ticks reach it in the next display frame through independent
	// readers, and every transition it observes is narrated onto the
	// debug panel.
	if err := g.AddSystem(core.PhaseFrame, "core.announce", core.SystemFunc(func(_ *core.Context) error {
		for _, ok := chargedFrame.Next(); ok; _, ok = chargedFrame.Next() {
			panel.Message("CORE CHARGED - arming launch countdown")
		}
		for event, ok := launchedFrame.Next(); ok; event, ok = launchedFrame.Next() {
			panel.Message(fmt.Sprintf("LAUNCHED on tick %d", event.tick))
		}
		return nil
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
	if err != nil {
		return err
	}

	return g.Run(runner)
}

// sequence holds the launch sequence's signal-driven state.
type sequence struct {
	phase      phase
	charge     int
	flash      int
	injections int

	injector  *core.Entity
	countdown *core.Entity
	core      *core.Entity
}

func (s *sequence) flashColor() color.Color {
	if s.flash > 0 {
		return flashColor
	}
	return runningColor
}

func barSprite(fill color.Color) render.Sprite {
	return render.Sprite{
		Drawable: render.RectShape{Size: geom.Vector2{X: injectorW, Y: injectorH}},
		Color:    fill,
	}
}

func coreSprite(size float64, fill color.Color) render.Sprite {
	return render.Sprite{
		Drawable: render.RectShape{Size: geom.Vector2{X: size, Y: size}},
		Color:    fill,
	}
}
