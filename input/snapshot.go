package input

import "github.com/Leonard-Atorough/castrum/geom"

// MaxPads is the number of gamepad slots in a [Snapshot].
const MaxPads = 4

// MouseSnapshot is the mouse state in one [Snapshot].
type MouseSnapshot struct {
	// Cursor is the cursor position in the runner's coordinate system.
	Cursor geom.Vector2
	// Buttons stores the state of each [MouseButton].
	Buttons [NumMouseButtons]State
	// Wheel is the scroll movement reported for this snapshot.
	Wheel geom.Vector2
}

// PadSnapshot is the state of one gamepad slot in a [Snapshot].
type PadSnapshot struct {
	// Connected reports whether a gamepad occupies this slot.
	Connected bool
	// Buttons stores the state of each [PadButton].
	Buttons [NumPadButtons]State
	// Axes stores the value of each [PadAxis].
	Axes [NumPadAxes]float64
}

// Snapshot is the state of all input devices for one frame. The runner
// populates it; game systems should treat it as read-only.
//
// Its read methods are nil-safe and return the zero value for invalid keys,
// buttons, axes, or pad indexes.
type Snapshot struct {
	// Keys holds the state of all keyboard keys in this snapshot.
	Keys [NumKeys]State
	// Mouse holds the state of the mouse in this snapshot.
	Mouse MouseSnapshot
	// Pads holds the state of all gamepads in this snapshot.
	Pads [MaxPads]PadSnapshot
}

// KeyPressed reports whether k was pressed in this snapshot.
func (s *Snapshot) KeyPressed(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Pressed
}

// KeyReleased reports whether k was released in this snapshot.
func (s *Snapshot) KeyReleased(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Released
}

// KeyHeld reports whether k is held in this snapshot.
func (s *Snapshot) KeyHeld(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Held
}

// MousePressed reports whether b was pressed in this snapshot.
func (s *Snapshot) MousePressed(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Pressed
}

// MouseReleased reports whether b was released in this snapshot.
func (s *Snapshot) MouseReleased(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Released
}

// MouseHeld reports whether b is held in this snapshot.
func (s *Snapshot) MouseHeld(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Held
}

// Cursor returns the cursor position in this snapshot.
func (s *Snapshot) Cursor() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Cursor
}

// Wheel returns the scroll movement in this snapshot.
func (s *Snapshot) Wheel() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Wheel
}

// PadHeld reports whether button is held on padIndex in this snapshot.
func (s *Snapshot) PadHeld(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Held
}

// PadPressed reports whether button was pressed on padIndex in this snapshot.
func (s *Snapshot) PadPressed(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Pressed
}

// PadReleased reports whether button was released on padIndex in this snapshot.
func (s *Snapshot) PadReleased(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Released
}

// PadConnected reports whether a gamepad occupies padIndex in this snapshot.
func (s *Snapshot) PadConnected(padIndex int) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads && s.Pads[padIndex].Connected
}

// PadAxis returns the value of axis on padIndex in this snapshot.
func (s *Snapshot) PadAxis(padIndex int, axis PadAxis) float64 {
	if s != nil && padIndex >= 0 && padIndex < MaxPads && axis >= 0 && axis < NumPadAxes {
		return s.Pads[padIndex].Axes[axis]
	}
	return 0
}
