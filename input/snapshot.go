package input

import "github.com/Leonard-Atorough/castrum/geom"

const MaxPads = 4 // Maximum number of gamepads supported. NOTE: Review when adding configurable pad count support.

// MouseSnapshot represents the state of the mouse at a given moment.
type MouseSnapshot struct {
	Cursor  geom.Vector2
	Buttons [NumMouseButtons]State
	Wheel   geom.Vector2
}

// PadSnapshot represents the state of a gamepad at a given moment.
type PadSnapshot struct {
	Connected bool
	Buttons   [NumPadButtons]State
	Axes      [NumPadAxes]float64
}

// Snapshot is the state of all input devices at a given moment: one
// frame of device truth, as the runner's poller filled it. The
// poller writes the fields; game systems read them and treat the
// snapshot as read-only.
//
// The read methods are nil-safe - a nil *Snapshot reads as no input
// - and bounds-safe: an out-of-range key, button, or pad index reads
// as zero.
type Snapshot struct {
	// Keys holds the state of all keyboard keys in this snapshot.
	Keys [NumKeys]State
	// Mouse holds the state of the mouse in this snapshot.
	Mouse MouseSnapshot
	// Pads holds the state of all gamepads in this snapshot.
	Pads [MaxPads]PadSnapshot
}

// KeyPressed reports whether the specified key was pressed in this snapshot.
func (s *Snapshot) KeyPressed(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Pressed
}

// KeyReleased reports whether the specified key was released in this snapshot.
func (s *Snapshot) KeyReleased(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Released
}

// KeyHeld reports whether the specified key is held in this snapshot.
func (s *Snapshot) KeyHeld(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Held
}

// MousePressed reports whether the specified mouse button was pressed in this snapshot.
func (s *Snapshot) MousePressed(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Pressed
}

// MouseReleased reports whether the specified mouse button was released in this snapshot.
func (s *Snapshot) MouseReleased(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Released
}

// MouseHeld reports whether the specified mouse button is held in this snapshot.
func (s *Snapshot) MouseHeld(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Held
}

// Cursor returns the current position of the mouse cursor in this snapshot.
func (s *Snapshot) Cursor() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Cursor
}

// Wheel returns the current state of the mouse wheel in this snapshot.
func (s *Snapshot) Wheel() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Wheel
}

// PadHeld reports whether the specified gamepad button is held in this snapshot.
func (s *Snapshot) PadHeld(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Held
}

// PadPressed reports whether the specified gamepad button was pressed in this snapshot.
func (s *Snapshot) PadPressed(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Pressed
}

// PadReleased reports whether the specified gamepad button was released in this snapshot.
func (s *Snapshot) PadReleased(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Released
}

// PadConnected reports whether the specified gamepad is connected in this snapshot.
func (s *Snapshot) PadConnected(padIndex int) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads && s.Pads[padIndex].Connected
}

// PadAxis returns the current value of the specified gamepad axis in this snapshot.
func (s *Snapshot) PadAxis(padIndex int, axis PadAxis) float64 {
	if s != nil && padIndex >= 0 && padIndex < MaxPads && axis >= 0 && axis < NumPadAxes {
		return s.Pads[padIndex].Axes[axis]
	}
	return 0
}
