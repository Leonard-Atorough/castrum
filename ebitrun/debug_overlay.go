package ebitrun

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
)

// overlayWindow is how long the debug overlay measures between
// updates; short enough to feel live, long enough for a stable read.
const overlayWindow = 500 * time.Millisecond

// debugOverlay samples and draws the runner's frame and tick rates, plus
// the game's pause state and time scale.
type debugOverlay struct {
	game    *castrum.Game
	started bool
	last    time.Time
	tick    uint64
	frames  uint64
	fps     float64
	tps     float64
}

func newDebugOverlay(game *castrum.Game) *debugOverlay {
	return &debugOverlay{game: game}
}

// sample records one display frame and, once a window has passed,
// the rates measured over it. tick is the engine's tick counter. The
// frame that starts a window anchors it and does not count, so a
// steady stream reads its true rate.
func (o *debugOverlay) sample(now time.Time, tick uint64) {
	if !o.started {
		o.started = true
		o.last = now
		o.tick = tick
		return
	}
	o.frames++
	elapsed := now.Sub(o.last)
	if elapsed < overlayWindow {
		return
	}
	o.fps = float64(o.frames) / elapsed.Seconds()
	o.tps = float64(tick-o.tick) / elapsed.Seconds()
	o.last, o.tick, o.frames = now, tick, 0
}

// draw samples the frame and renders the current rates.
func (o *debugOverlay) draw(ctx *core.Context, screen *ebiten.Image) error {
	o.sample(time.Now(), ctx.Tick)
	ebitenutil.DebugPrint(screen, o.status())
	return nil
}

// status renders the overlay line: frame and tick rates, then the pause
// state and time scale when either departs from its default. Paused games
// still read their true frame rate; the tick rate naturally falls to zero.
func (o *debugOverlay) status() string {
	line := fmt.Sprintf("fps %.1f  tps %.1f", o.fps, o.tps)
	if o.game.Paused() {
		line += "  paused"
	}
	if scale := o.game.TimeScale(); scale != 1 {
		line += fmt.Sprintf("  scale %.2g", scale)
	}
	return line
}
