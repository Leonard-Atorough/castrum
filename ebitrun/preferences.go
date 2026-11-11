package ebitrun

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/settings"
)

// DisplayPreferences holds desired startup window settings.
type DisplayPreferences struct {
	// Window is the initial window size in pixels.
	Window Size
	// Resizable permits window resizing.
	Resizable bool
	// VSync enables vertical synchronization.
	VSync bool
}

// NewDisplayPreferences returns a group with the runner's default window
// settings. Changes do not affect a runner that has already been constructed.
func NewDisplayPreferences() (*settings.Group[DisplayPreferences], error) {
	opts := defaultOptions()
	return settings.New(DisplayPreferences{Window: opts.Window, Resizable: opts.Resizable, VSync: opts.VSync}, settings.Policy[DisplayPreferences]{
		Validator: settings.ValidatorFunc[DisplayPreferences](DisplayPreferences.Validate),
	})
}

// Validate reports an error if the window size is not positive.
func (p DisplayPreferences) Validate() error {
	if p.Window.Width <= 0 || p.Window.Height <= 0 {
		return fmt.Errorf("castrum/ebiten: window size %dx%d is not positive", p.Window.Width, p.Window.Height)
	}
	return nil
}

// WithDisplayPreferences sets the runner's startup window settings.
// Later options can override these settings. Invalid values are reported by
// [New].
func WithDisplayPreferences(prefs DisplayPreferences) option {
	return optionFunc(func(o *Options) {
		o.Window, o.Resizable, o.VSync = prefs.Window, prefs.Resizable, prefs.VSync
	})
}
