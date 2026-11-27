// Package ebitrun provides the default Ebitengine-backed [Runner] for
// castrum. The runner owns the window, draw surface, and input; [castrum.Game]
// owns configuration, schedules, and the fixed loop.
package ebitrun

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

// Size is a pixel dimension used for window or logical resolution.
type Size struct {
	// Width is the horizontal dimension in pixels.
	Width int `json:"width"`
	// Height is the vertical dimension in pixels.
	Height int `json:"height"`
}

// DrawFunc renders one frame to the runner's Ebitengine canvas.
type DrawFunc func(ctx *core.Context, screen *ebiten.Image, alpha float64) error

// Runner drives a [castrum.Game] with Ebitengine.
//
// Start the game with [Runner.Run].
type Runner struct {
	g       *castrum.Game
	win     *window
	hub     *signal.Hub
	opts    Options
	draws   []DrawFunc
	engine  DrawFunc
	input   poller
	last    time.Time
	drawErr error
}

// AudioSystemName is the name of the game's audio reconciler.
const AudioSystemName = audio.SystemName

// New creates a Runner with opts applied over the defaults. It panics for an
// invalid option and performs no other setup until [Runner.Run].
func New(opts ...option) *Runner {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.validate(); err != nil {
		panic(err)
	}
	return &Runner{opts: options}
}

// Run creates a Runner with opts and runs g. It is the one-call form of
// [New] followed by [Runner.Run] and panics like both.
func Run(g *castrum.Game, opts ...option) {
	New(opts...).Run(g)
}

// bind attaches the runner to g before startup: it registers the
// [TextureProvider], [FontProvider], and [render.Window] resources, and
// attaches the audio controller. g must come from [castrum.New], which
// provides the game's [asset.Loader].
func (r *Runner) bind(g *castrum.Game) error {
	backend, err := g.World().Resource[*audio.Backend]()
	if err != nil {
		return fmt.Errorf("ebitrun: resolve audio backend: %w", err)
	}
	if backend.Controller() != nil {
		return fmt.Errorf("ebitrun: audio controller already attached")
	}
	loader := g.World().MustResource[*asset.Loader]()
	codecs := g.World().MustResource[*asset.Codecs]()
	audioCtx := ebitaudio.CurrentContext()
	if audioCtx == nil {
		audioCtx = ebitaudio.NewContext(r.opts.SampleRate)
	}
	controller, err := newAudioProvider(audioCtx, loader, codecs)
	if err != nil {
		return err
	}
	provider := newTextureProvider(loader)
	if err := g.World().ProvideEager(func(*core.World) (*TextureProvider, error) {
		return provider, nil
	}); err != nil {
		return fmt.Errorf("ebitrun: provide texture provider: %w", err)
	}
	win := &window{
		vp: render.Viewport{
			LogicalWidth:  r.opts.Logical.Width,
			LogicalHeight: r.opts.Logical.Height,
			WindowWidth:   r.opts.Window.Width,
			WindowHeight:  r.opts.Window.Height,
		},
		state: render.WindowState{
			Title:         "",
			FullScreen:    r.opts.Fullscreen,
			Bordered:      r.opts.Bordered,
			Resizable:     r.opts.Resizable,
			Vsync:         r.opts.VSync,
			CursorVisible: true,
		},
	}
	r.win = win
	r.hub = g.World().MustResource[*signal.Hub]()
	if err := g.World().ReplaceResource[render.Window](win); err != nil {
		return fmt.Errorf("ebitrun: set render window: %w", err)
	}

	collector := render.NewCollector(g.World())
	fonts := newFontProvider(loader)
	if err := g.World().ProvideEager(func(*core.World) (*FontProvider, error) {
		return fonts, nil
	}); err != nil {
		return fmt.Errorf("ebitrun: provide font provider: %w", err)
	}

	if r.opts.DebugOverlay {
		overlay := newDebugOverlay(g)
		r.AddDraw(overlay.draw)
	}
	if err := backend.Attach(controller); err != nil {
		return fmt.Errorf("ebitrun: attach audio controller: %w", err)
	}
	r.g = g
	r.engine = newEngineDrawFunc(collector, provider, fonts)
	r.last = time.Now()
	return nil
}

// AddDraw appends f to the user draw callbacks. They run in registration
// order, after the engine renders the world.
func (r *Runner) AddDraw(f DrawFunc) {
	r.draws = append(r.draws, f)
}

// Run binds g, applies the window settings, starts the game, and blocks in
// the Ebitengine loop until it quits. It panics on a running failure.
func (r *Runner) Run(g *castrum.Game) {
	if err := r.run(g); err != nil {
		panic(err)
	}
}

func (r *Runner) run(g *castrum.Game) error {
	if err := g.BeginRun(); err != nil {
		return err
	}
	if err := r.bind(g); err != nil {
		return err
	}
	r.win.SetTitle(g.Options().Title)
	r.win.SetSize(r.opts.Window.Width, r.opts.Window.Height)
	r.win.SetFullscreen(r.opts.Fullscreen)
	r.win.SetBordered(r.opts.Bordered)
	r.win.SetResizable(r.opts.Resizable)
	r.win.SetVSync(r.opts.VSync)
	// The core accumulator owns pacing: Ebitengine must not pace Update
	// itself, or fixed ticks and interpolation alpha break.
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := g.Startup(); err != nil {
		return err
	}
	r.last = time.Now()
	return ebiten.RunGame(r)
}

// Update advances the game by elapsed platform time and reports quit
// requests to Ebitengine. Errors from [Runner.Draw] are returned here on the
// next frame because Ebitengine's Draw callback cannot return an error.
func (r *Runner) Update() error {
	if r.drawErr != nil {
		err := r.drawErr
		r.drawErr = nil
		return err
	}
	now := time.Now()
	elapsed := now.Sub(r.last)
	r.last = now

	// Poll before Advance: PhaseFrame systems read this frame's
	// device state through the snapshot resource.
	r.input.poll()
	snapshot, err := r.g.World().Resource[*input.Snapshot]()
	if err != nil {
		return err
	}
	*snapshot = *r.input.snapshotPtr()
	// Report the cursor in logical render coordinates so ScreenToWorld and
	// camera math work at any window size.
	if r.win != nil {
		snapshot.Mouse.Cursor = r.win.Viewport().ToLogical(snapshot.Mouse.Cursor)
	}

	if err := r.g.Advance(elapsed); err != nil {
		return err
	}
	if r.g.Quitting() {
		return ebiten.Termination
	}
	return nil
}

// Draw renders the engine's world, then the registered user callbacks, with
// the current interpolation alpha. It runs every callback even if one fails;
// the first error is returned by [Runner.Update] on the next frame.
func (r *Runner) Draw(screen *ebiten.Image) {
	ctx := r.g.Context()
	alpha := r.g.Alpha()

	if err := r.engine(ctx, screen, alpha); err != nil && r.drawErr == nil {
		r.drawErr = fmt.Errorf("engine draw: %w", err)
	}
	for _, f := range r.draws {
		if err := f(ctx, screen, alpha); err != nil && r.drawErr == nil {
			r.drawErr = fmt.Errorf("draw: %w", err)
		}
	}
}

// Layout returns the current logical render size, independent of the window
// size, and records the window size in the [render.Window] resource.
func (r *Runner) Layout(outsideWidth, outsideHeight int) (int, int) {
	if r.win != nil {
		if r.win.setWindowSize(outsideWidth, outsideHeight) && r.hub != nil {
			r.hub.Emit(ViewportChanged{
				OldWidth:  r.win.OldWidth(),
				OldHeight: r.win.OldHeight(),
				NewWidth:  outsideWidth,
				NewHeight: outsideHeight,
			})
		}
		vp := r.win.Viewport()
		return vp.LogicalWidth, vp.LogicalHeight
	}
	return r.opts.Logical.Width, r.opts.Logical.Height
}

// ViewportChanged is emitted when the window's viewport changes size. It contains
// the old and new logical dimensions of the viewport.
//
// It is emitted by the [Runner] when the window's layout changes.
type ViewportChanged struct {
	OldWidth  int
	OldHeight int
	NewWidth  int
	NewHeight int
}
