package render

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Viewport describes the logical render target and the window that presents
// it. The runner owns the current value, publishes it through
// [Window.Viewport], and updates it when the window is resized.
type Viewport struct {
	// LogicalWidth is the internal render resolution's width in pixels.
	LogicalWidth int
	// LogicalHeight is the internal render resolution's height in pixels.
	LogicalHeight int
	// WindowWidth is the window's current width in pixels.
	WindowWidth int
	// WindowHeight is the window's current height in pixels.
	WindowHeight int
}

// WindowState describes the current state of the window, including title,
// fullscreen, bordered, resizable, vsync, and cursor visibility.
type WindowState struct {
	Title                           string
	FullScreen, Bordered, Resizable bool
	Vsync, CursorVisible            bool
}

// ScaleX returns the horizontal scale from logical to window pixels, in
// window pixels per logical pixel. It returns infinity or NaN when
// LogicalWidth is zero.
func (v Viewport) ScaleX() float64 {
	return float64(v.WindowWidth) / float64(v.LogicalWidth)
}

// ScaleY returns the vertical scale from logical to window pixels, in window
// pixels per logical pixel. It returns infinity or NaN when LogicalHeight is
// zero.
func (v Viewport) ScaleY() float64 {
	return float64(v.WindowHeight) / float64(v.LogicalHeight)
}

// Size returns the logical render size as a vector, in logical pixels.
func (v Viewport) Size() geom.Vector2 {
	return geom.Vector2{X: float64(v.LogicalWidth), Y: float64(v.LogicalHeight)}
}

// Center returns the center of the logical render target, in logical pixels.
func (v Viewport) Center() geom.Vector2 {
	return geom.Vector2{X: float64(v.LogicalWidth) / 2, Y: float64(v.LogicalHeight) / 2}
}

// ToLogical converts a point from window coordinates to logical coordinates,
// for example a raw cursor position to render coordinates.
func (v Viewport) ToLogical(p geom.Vector2) geom.Vector2 {
	return geom.Vector2{X: p.X / v.ScaleX(), Y: p.Y / v.ScaleY()}
}

// ToWindow converts a point from logical coordinates to window coordinates.
func (v Viewport) ToWindow(p geom.Vector2) geom.Vector2 {
	return geom.Vector2{X: p.X * v.ScaleX(), Y: p.Y * v.ScaleY()}
}

func (v Viewport) validate() error {
	if v.LogicalWidth <= 0 || v.LogicalHeight <= 0 {
		return fmt.Errorf("logical dimensions must be positive")
	}
	if v.WindowWidth <= 0 || v.WindowHeight <= 0 {
		return fmt.Errorf("window dimensions must be positive")
	}
	return nil
}

// StaticWindow is a [Window] for headless runs and tests that build a bare
// world. Its controls do nothing. [castrum.New] provides one as the default
// window until a runner replaces it.
type StaticWindow struct {
	vp Viewport
}

// NewStaticWindow returns a [StaticWindow] reporting v. It panics if either
// the logical or the window dimensions are not positive.
func NewStaticWindow(v Viewport) *StaticWindow {
	if err := v.validate(); err != nil {
		panic(fmt.Sprintf("render: NewStaticWindow: %v", err))
	}
	return &StaticWindow{vp: v}
}

// Viewport returns the current viewport.
func (w *StaticWindow) Viewport() Viewport { return w.vp }

// WindowState returns the zero state; the controls do nothing.
func (w *StaticWindow) WindowState() WindowState { return WindowState{} }

// SetTitle does nothing.
func (w *StaticWindow) SetTitle(title string) {}

// SetSize does nothing.
func (w *StaticWindow) SetSize(width, height int) {}

// SetFullscreen does nothing.
func (w *StaticWindow) SetFullscreen(fullscreen bool) {}

// SetBordered does nothing.
func (w *StaticWindow) SetBordered(bordered bool) {}

// SetResizable does nothing.
func (w *StaticWindow) SetResizable(resizable bool) {}

// SetVSync does nothing.
func (w *StaticWindow) SetVSync(enabled bool) {}

// SetCursorVisible does nothing.
func (w *StaticWindow) SetCursorVisible(visible bool) {}

// SetLogicalSize does nothing; the viewport is fixed at construction.
func (w *StaticWindow) SetLogicalSize(width, height int) {}

// Window is the presentation surface for a game. The runner provides it as a
// world resource; systems read the current [Viewport] and request changes
// through the Set methods.
//
// Implementations must be safe to call from the game loop's thread, which is
// where systems run. Changes take effect on the runner's next window update.
type Window interface {
	// Viewport returns the current viewport. Read-only.
	Viewport() Viewport
	// WindowState returns the current window state. Read-only.
	WindowState() WindowState
	// SetTitle sets the window title.
	SetTitle(title string)
	// SetSize sets the window size in pixels.
	SetSize(width, height int)
	// SetFullscreen switches the window between fullscreen and windowed.
	SetFullscreen(fullscreen bool)
	// SetBordered controls whether the window has decorations such as a title
	// bar and border.
	SetBordered(bordered bool)
	// SetResizable controls whether the window can be resized.
	SetResizable(resizable bool)
	// SetVSync enables or disables vertical synchronization.
	SetVSync(enabled bool)
	// SetCursorVisible controls whether the cursor is visible over the window.
	SetCursorVisible(visible bool)
	// SetLogicalSize sets the logical render size in pixels.
	SetLogicalSize(width, height int)
}
