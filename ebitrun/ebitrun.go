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
	"github.com/Leonard-Atorough/castrum/render"
)

// Size is a pixel dimension used for window or logical resolution.
type Size struct {
	// Width is the horizontal dimension in pixels.
	Width int
	// Height is the vertical dimension in pixels.
	Height int
}

// DrawFunc renders one frame to the runner's Ebitengine canvas.
type DrawFunc func(ctx *core.Context, screen *ebiten.Image) error

// Options configures the Ebitengine window, logical render size, and audio
// context.
type Options struct {
	// Window is the initial window size in pixels.
	Window Size
	// Resizable controls whether the window can be resized.
	Resizable bool
	// VSync enables vertical synchronization.
	VSync bool
	// Logical is the internal render size in pixels.
	Logical Size
	// SampleRate is the audio context's mixing rate in Hz, used when the
	// runner creates the context.
	SampleRate int
	// DebugOverlay draws the frame and tick rates in the window's
	// top-left corner. Default is off.
	DebugOverlay bool
}

type option interface {
	apply(*Options)
}

type optionFunc func(*Options)

func (f optionFunc) apply(opts *Options) {
	f(opts)
}

func defaultOptions() Options {
	return Options{
		Window:     Size{Width: 1280, Height: 720},
		Logical:    Size{Width: 1280, Height: 720},
		VSync:      true,
		SampleRate: 44100,
	}
}

// Runner drives a [castrum.Game] with Ebitengine.
//
// It implements [castrum.Runner]. Start the game with [castrum.Game.Run].
type Runner struct {
	g       *castrum.Game
	opts    Options
	draws   []DrawFunc
	engine  DrawFunc
	input   poller
	last    time.Time
	drawErr error
}

var _ castrum.Runner = (*Runner)(nil)

// AudioSystemName is the name the runner registers the audio
// reconciler under in the frame schedule.
const AudioSystemName = "engine.audio"

// New creates a Runner for g, applying opts over the defaults. It returns an
// error for invalid options or if the runner's resources or audio system
// cannot be registered.
//
// g must come from [castrum.New], which provides the game's [asset.Server].
// New registers the [TextureProvider], [FontProvider], and audio provider as
// eager resources, and adds the audio reconciler to the frame schedule.
func New(g *castrum.Game, opts ...option) (*Runner, error) {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.validate(); err != nil {
		return nil, err
	}

	server := g.World().MustResource[*asset.Server]()
	provider := newTextureProvider(server)
	if err := g.World().ProvideEager(func(*core.World) (*TextureProvider, error) {
		return provider, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide texture provider: %w", err)
	}

	audioCtx := ebitaudio.CurrentContext()
	if audioCtx == nil {
		audioCtx = ebitaudio.NewContext(options.SampleRate)
	}
	controller := newAudioProvider(audioCtx, server)
	if err := g.World().ProvideEager(func(*core.World) (audio.Controller, error) {
		return controller, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide audio provider: %w", err)
	}

	if err := g.AddSystem(core.PhaseFrame, AudioSystemName, audio.NewAudioSystem(controller, g.World().MustResource[*audio.Mixer]())); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: register %s: %w", AudioSystemName, err)
	}

	g.Context().LogicalWidth = options.Logical.Width
	g.Context().LogicalHeight = options.Logical.Height

	collector := render.NewCollector(g.World())
	fonts := newFontProvider(server)
	if err := g.World().ProvideEager(func(*core.World) (*FontProvider, error) {
		return fonts, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide font provider: %w", err)
	}

	engineDraw := newEngineDrawFunc(collector, provider, fonts)

	runner := &Runner{g: g, opts: options, last: time.Now(), engine: engineDraw}
	if options.DebugOverlay {
		overlay := newDebugOverlay()
		runner.AddDraw(overlay.draw)
	}
	return runner, nil
}

func (o *Options) validate() error {
	if o.Window.Width <= 0 || o.Window.Height <= 0 {
		return fmt.Errorf("castrum/ebiten: window size %dx%d is not positive", o.Window.Width, o.Window.Height)
	}
	if o.Logical.Width <= 0 || o.Logical.Height <= 0 {
		return fmt.Errorf("castrum/ebiten: logical size %dx%d is not positive", o.Logical.Width, o.Logical.Height)
	}
	if o.SampleRate <= 0 {
		return fmt.Errorf("castrum/ebiten: audio sample rate %d is not positive", o.SampleRate)
	}
	return nil
}

// AddDraw appends f to the user draw callbacks. They run in registration
// order, after the engine renders the world.
func (r *Runner) AddDraw(f DrawFunc) {
	r.draws = append(r.draws, f)
}

// Run applies the window settings, starts the game, and blocks in the
// Ebitengine loop until it quits. A startup error is returned before the
// window opens.
func (r *Runner) Run() error {
	ebiten.SetWindowTitle(r.g.Options().Title)
	ebiten.SetWindowSize(r.opts.Window.Width, r.opts.Window.Height)
	mode := ebiten.WindowResizingModeDisabled
	if r.opts.Resizable {
		mode = ebiten.WindowResizingModeEnabled
	}
	ebiten.SetWindowResizingMode(mode)
	ebiten.SetVsyncEnabled(r.opts.VSync)
	// The core accumulator owns pacing: Ebitengine must not pace Update
	// itself, or fixed ticks and interpolation alpha break.
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := r.g.Startup(); err != nil {
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
	// device state through the published snapshot.
	r.input.poll()
	r.g.Context().Input = r.input.snapshotPtr()

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
	ctx.Alpha = r.g.Alpha()

	if err := r.engine(ctx, screen); err != nil && r.drawErr == nil {
		r.drawErr = fmt.Errorf("engine draw: %w", err)
	}
	for _, f := range r.draws {
		if err := f(ctx, screen); err != nil && r.drawErr == nil {
			r.drawErr = fmt.Errorf("draw: %w", err)
		}
	}
}

// Layout returns the logical render size, independent of the window size.
func (r *Runner) Layout(outsideWidth, outsideHeight int) (int, int) {
	return r.opts.Logical.Width, r.opts.Logical.Height
}

// WithWindowSize sets the window size in pixels. Default is 1280x720.
// Invalid values are reported by [New].
func WithWindowSize(width, height int) option {
	return optionFunc(func(o *Options) { o.Window = Size{Width: width, Height: height} })
}

// WithLogicalSize sets the internal render resolution in pixels.
// Default is 1280x720. Invalid values are reported by [New].
func WithLogicalSize(width, height int) option {
	return optionFunc(func(o *Options) { o.Logical = Size{Width: width, Height: height} })
}

// WithResizable enables window resizing. Default is a fixed-size window.
func WithResizable() option {
	return optionFunc(func(o *Options) { o.Resizable = true })
}

// WithoutVSync disables vsync. Default is on.
func WithoutVSync() option {
	return optionFunc(func(o *Options) { o.VSync = false })
}

// WithAudioSampleRate sets the sample rate in Hz for the audio context created
// by the runner. An existing Ebitengine audio context is reused instead.
// Audio sources are resampled to this rate when decoded. The default is
// 44100; invalid values are reported by [New].
func WithAudioSampleRate(sampleRate int) option {
	return optionFunc(func(o *Options) { o.SampleRate = sampleRate })
}

// WithDebugOverlay draws a development overlay in the window's top-left
// corner: fps counts the display frames rendered per second, and tps counts
// the fixed simulation ticks the engine advances per second. The overlay
// draws only - it never changes simulation timing or game state - and is
// off unless the option is passed.
func WithDebugOverlay() option {
	return optionFunc(func(o *Options) { o.DebugOverlay = true })
}
