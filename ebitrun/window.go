package ebitrun

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum/render"
)

type window struct {
	vp                  render.Viewport
	state               render.WindowState
	oldWidth, oldHeight int
}

func (w *window) Viewport() render.Viewport {
	return w.vp
}

func (w *window) WindowState() render.WindowState {
	return w.state
}

func (w *window) OldWidth() int  { return w.oldWidth }
func (w *window) OldHeight() int { return w.oldHeight }

func (w *window) SetTitle(title string) {
	ebiten.SetWindowTitle(title)
	w.state.Title = title
}

func (w *window) SetSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	ebiten.SetWindowSize(width, height)
	w.vp.WindowWidth, w.vp.WindowHeight = width, height
}

func (w *window) SetFullscreen(fullscreen bool) {
	ebiten.SetFullscreen(fullscreen)
	w.state.FullScreen = fullscreen
}

func (w *window) SetBordered(bordered bool) {
	ebiten.SetWindowDecorated(bordered)
	w.state.Bordered = bordered
}

func (w *window) SetResizable(resizable bool) {
	mode := ebiten.WindowResizingModeDisabled
	if resizable {
		mode = ebiten.WindowResizingModeEnabled
	}
	ebiten.SetWindowResizingMode(mode)
	w.state.Resizable = resizable
}

func (w *window) SetVSync(enabled bool) {
	ebiten.SetVsyncEnabled(enabled)
	w.state.Vsync = enabled
}

func (w *window) SetCursorVisible(visible bool) {
	mode := ebiten.CursorModeVisible
	if !visible {
		mode = ebiten.CursorModeHidden
	}
	ebiten.SetCursorMode(mode)
	w.state.CursorVisible = visible
}

func (w *window) SetLogicalSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	w.vp.LogicalWidth, w.vp.LogicalHeight = width, height
}

// setWindowSize records a resize reported by Layout and reports whether the
// window size actually changed.
func (w *window) setWindowSize(width, height int) bool {
	if width <= 0 || height <= 0 || (width == w.vp.WindowWidth && height == w.vp.WindowHeight) {
		return false
	}
	w.oldWidth, w.oldHeight = w.vp.WindowWidth, w.vp.WindowHeight
	w.vp.WindowWidth, w.vp.WindowHeight = width, height
	return true
}
