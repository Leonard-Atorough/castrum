// Package ebitrun is the default castrum Runner, backed by Ebitengine.
// It owns the window, the draw surface, and input; the core Game owns
// configuration, schedules, and the fixed loop.
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
)

// Size is a 2D dimension in pixels, shared by window and logical resolution.
type Size struct {
	Width  int
	Height int
}

// DrawFunc renders one frame. DrawFuncs are backend-typed by design: they
// receive the runner's canvas directly, so they do not transfer across
// runners.
type DrawFunc func(ctx *core.Context, screen *ebiten.Image) error

// Options holds the runner's launch settings: window, vsync, and the
// internal render resolution.
type Options struct {
	Window    Size
	Resizable bool
	VSync     bool
	Logical   Size
	// SampleRate is the audio context's mixing rate in Hz.
	SampleRate int
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

// Runner drives a castrum.Game over Ebitengine.
//
// It implements [castrum.Runner]; start the game through
// [castrum.Game.Run].
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

// New creates the Runner for g, applying opts over defaults. The option
// constructors never fail; all validation happens here in a single pass.
// g must come from [castrum.New], which provides the game's [asset.Server];
// New wires the runner's [TextureProvider] and audio provider over it,
// both provided eagerly, and registers the engine.audio reconciler.
func New(g *castrum.Game, opts ...option) (*Runner, error) {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.validate(); err != nil {
		return nil, err
	}

	server, err := g.World().Resource[*asset.Server]()
	if err != nil {
		return nil, fmt.Errorf("castrum/ebiten: asset server: %w", err)
	}
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
	audioProvider := newAudioProvider(audioCtx, server)
	if err := g.World().ProvideEager(func(*core.World) (audio.Controller, error) {
		return audioProvider, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide audio provider: %w", err)
	}

	if err := g.AddSystem(core.PhaseFrame, "engine.audio", audio.NewAudioSystem(audioProvider, g.Mixer())); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: register engine.audio: %w", err)
	}

	g.Context().LogicalWidth = options.Logical.Width
	g.Context().LogicalHeight = options.Logical.Height

	collector := core.NewCollector(g.World())
	fonts := newFontProvider(server)
	if err := g.World().ProvideEager(func(*core.World) (*FontProvider, error) {
		return fonts, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide font provider: %w", err)
	}

	engineDraw := newEngineDrawFunc(collector, provider, fonts)

	return &Runner{g: g, opts: options, last: time.Now(), engine: engineDraw}, nil
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

// AddDraw registers a user draw system. DrawFuncs run in registration
// order, after the engine's world rendering - overlays land on top of
// the world.
func (r *Runner) AddDraw(f DrawFunc) {
	r.draws = append(r.draws, f)
}

// Run applies the window settings, runs the game's startup schedule, and
// blocks in the Ebitengine loop until the game quits. A startup failure
// returns before a window opens.
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

// Update advances the game by the elapsed platform time and reports quit
// requests to Ebitengine.
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

// Draw renders the engine's collected draw list, then the registered
// DrawFuncs, with the interpolation alpha - engine world first, user
// overlays on top. A draw error is stored and surfaces from Update on
// the next frame, since Ebitengine's Draw cannot return errors; the
// stored error names the failing layer ("engine draw" or "draw").
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

// Layout returns the internal render resolution; the window may
// differ, and Ebitengine letterboxes the difference (uniform scale,
// centered). The returned size must stay equal to the logical
// resolution New published to Context.LogicalWidth/Height: culling
// reads those, projection reads this, and the two must never diverge.
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

// WithAudioSampleRate sets the audio context's sample rate in Hz.
// Sources are resampled to it at decode time; a source already at
// this rate never resamples. Default is 44100. Invalid values are
// reported by [New].
func WithAudioSampleRate(sampleRate int) option {
	return optionFunc(func(o *Options) { o.SampleRate = sampleRate })
}
