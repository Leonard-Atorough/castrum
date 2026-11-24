package ebitrun

import "fmt"

// Options configures the Ebitengine window, logical render size, and audio
// context.
type Options struct {
	// Window is the initial window size in pixels.
	Window Size
	// Resizable controls whether the window can be resized.
	Resizable bool
	// VSync enables vertical synchronization.
	VSync bool
	// Bordered controls whether the window has decorations such
	// as a title bar and border.
	Bordered bool
	// Fullscreen controls whether the window starts in fullscreen mode.
	Fullscreen bool
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
		Bordered:   true,
		SampleRate: 44100,
	}
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

// WithoutBordered disables window decorations such as the title bar and border.
// Default is on.
func WithoutBordered() option {
	return optionFunc(func(o *Options) { o.Bordered = false })
}

// WithFullscreen enables fullscreen mode. Default is windowed.
func WithFullscreen() option {
	return optionFunc(func(o *Options) { o.Fullscreen = true })
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
// corner: fps counts the display frames rendered per second, tps counts
// the fixed simulation ticks the engine advances per second, and the
// game's paused state and time scale appear whenever they depart from
// running at full speed. The overlay draws only - it never changes
// simulation timing or game state - and is off unless the option is
// passed.
func WithDebugOverlay() option {
	return optionFunc(func(o *Options) { o.DebugOverlay = true })
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
