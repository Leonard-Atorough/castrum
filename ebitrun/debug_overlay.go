package ebitrun

import (
	"fmt"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
)

// overlayWindow is how long the debug overlay measures between
// updates; short enough to feel live, long enough for a stable read.
const overlayWindow = 500 * time.Millisecond

// messageRow is the screen y of the first panel message and the step
// between message rows.
const (
	firstMessageY    = 20
	messageRowHeight = 16
)

// debugOverlay samples and draws the runner's frame and tick rates, plus
// the game's pause state, time scale, and debug panel messages.
type debugOverlay struct {
	game    *castrum.Game
	panel   *debug.Panel
	started bool
	last    time.Time
	tick    uint64
	frames  uint64
	fps     float64
	tps     float64
}

func newDebugOverlay(game *castrum.Game) *debugOverlay {
	o := &debugOverlay{game: game}
	if game != nil {
		// The panel is engine wiring castrum.New guarantees.
		o.panel = game.World().MustResource[*debug.Panel]()
	}
	return o
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

// draw samples and renders the status line, the panel's status lines, and
// the panel's messages, in that order.
func (o *debugOverlay) draw(ctx *core.Context, screen *ebiten.Image) error {
	o.sample(time.Now(), ctx.Tick)
	if line := o.status(); line != "" {
		ebitenutil.DebugPrint(screen, line)
	}
	row := 0
	for _, line := range o.panel.StatusLines() {
		ebitenutil.DebugPrintAt(screen, line, 0, firstMessageY+row*messageRowHeight)
		row++
	}
	for _, message := range o.panel.Lines() {
		ebitenutil.DebugPrintAt(screen, message, 0, firstMessageY+row*messageRowHeight)
		row++
	}
	return nil
}

// status renders the overlay's first line: the measured rates the panel
// enables, then the pause state and time scale when either departs from
// its default. Paused games still read their true frame rate; the tick
// rate naturally falls to zero.
func (o *debugOverlay) status() string {
	var sections []string
	if o.panel.Enabled(debug.FPS) {
		sections = append(sections, fmt.Sprintf("fps %.1f", o.fps))
	}
	if o.panel.Enabled(debug.TPS) {
		sections = append(sections, fmt.Sprintf("tps %.1f", o.tps))
	}
	line := strings.Join(sections, "  ")
	if o.game.Paused() {
		line = appendSuffix(line, "paused")
	}
	if scale := o.game.TimeScale(); scale != 1 {
		line = appendSuffix(line, fmt.Sprintf("scale %.2g", scale))
	}
	return line
}

func appendSuffix(line, suffix string) string {
	if line == "" {
		return suffix
	}
	return line + "  " + suffix
}
