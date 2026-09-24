// Package ebitrun is the default castrum Runner, backed by Ebitengine.
// It owns the window, the draw surface, and input; the core Game owns
// configuration, schedules, and the fixed loop.
//
// The package name is ebitrun so no import alias is needed alongside
// Ebitengine itself:
//
//	import (
//		"github.com/Leonard-Atorough/castrum"
//		"github.com/Leonard-Atorough/castrum/ebiten"
//		"github.com/hajimehoshi/ebiten/v2"
//	)
package ebitrun

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
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
	Window     Size
	Resizable  bool
	VSync      bool
	Logical    Size
	Filesystem fs.FS
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
		Window:  Size{Width: 1280, Height: 720},
		Logical: Size{Width: 1280, Height: 720},
		VSync:   true,
	}
}

// Runner drives a castrum.Game over Ebitengine.
type Runner struct {
	g       *castrum.Game
	opts    Options
	draws   []DrawFunc
	last    time.Time
	drawErr error
}

// New creates the Runner for g, applying opts over defaults. The option
// constructors never fail; all validation happens here in a single pass.
// New also wires the asset pipeline into g's world: an [asset.Server]
// over the configured filesystem, provided as a resource, and the
// runner's [TextureProvider], provided eagerly. A game that already
// provided its own server owns the conflict; New returns the error.
func New(g *castrum.Game, opts ...option) (*Runner, error) {
	options := defaultOptions()
	for _, o := range opts {
		o.apply(&options)
	}
	if err := options.validate(); err != nil {
		return nil, err
	}

	server := asset.New(options.Filesystem)
	if err := g.World().Provide(func(*core.World) (*asset.Server, error) {
		return server, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide asset server: %w", err)
	}
	provider := newTextureProvider(server)
	if err := g.World().ProvideEager(func(*core.World) (*TextureProvider, error) {
		return provider, nil
	}); err != nil {
		return nil, fmt.Errorf("castrum/ebiten: provide texture provider: %w", err)
	}

	// The logical resolution is static per game: publish it to the
	// context at construction, where culling and camera projection
	// read it. (Alpha is per-frame and set in Draw.)
	g.Context().LogicalWidth = options.Logical.Width
	g.Context().LogicalHeight = options.Logical.Height

	return &Runner{g: g, opts: options, last: time.Now()}, nil
}

// validate checks the converged options. It is the single owner of
// validation: the option constructors are dumb setters.
func (o *Options) validate() error {
	if o.Window.Width <= 0 || o.Window.Height <= 0 {
		return fmt.Errorf("castrum/ebiten: window size %dx%d is not positive", o.Window.Width, o.Window.Height)
	}
	if o.Logical.Width <= 0 || o.Logical.Height <= 0 {
		return fmt.Errorf("castrum/ebiten: logical size %dx%d is not positive", o.Logical.Width, o.Logical.Height)
	}
	return nil
}

// AddDraw registers a draw system. DrawFuncs run in registration order.
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
	if err := r.g.Advance(elapsed); err != nil {
		return err
	}
	if r.g.Quitting() {
		return ebiten.Termination
	}
	return nil
}

// Draw runs the registered DrawFuncs with the interpolation alpha. A
// DrawFunc error is stored and surfaces from Update on the next frame,
// since Ebitengine's Draw cannot return errors.
func (r *Runner) Draw(screen *ebiten.Image) {
	ctx := r.g.Context()
	ctx.Alpha = r.g.Alpha()
	for _, f := range r.draws {
		if err := f(ctx, screen); err != nil && r.drawErr == nil {
			r.drawErr = fmt.Errorf("draw: %w", err)
		}
	}
}

// Layout returns the internal render resolution; the window may differ.
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

// WithFilesystem sets the filesystem asset paths resolve against. Any
// fs.FS works: embed.FS for single-binary distribution, os.DirFS for
// development layouts, fstest.MapFS for tests. Default is the game's
// working directory.
func WithFilesystem(fs fs.FS) option {
	return optionFunc(func(o *Options) { o.Filesystem = fs })
}
