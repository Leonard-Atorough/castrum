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
// frame of device truth, as the runner's poller filled it. The fields
// are exported because the poller lives outside this package and
// writes them directly.
//
// The methods are the read face, and they carry device truth only -
// no policy: which pad wins an any-pad binding, what the deadzone
// does, and how actions fold these values are ActionMap's rules.
// They are nil-safe and bounds-safe: a nil *Snapshot reads as
// nothing held and nothing pressed (headless tests have no runner
// publishing input, and query it without guards), and an out-of-range
// key, button, or pad index reads as zero rather than panicking.
type Snapshot struct {
	Keys  [NumKeys]State
	Mouse MouseSnapshot
	Pads  [MaxPads]PadSnapshot
}

func (s *Snapshot) KeyPressed(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Pressed
}

func (s *Snapshot) KeyReleased(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Released
}

func (s *Snapshot) KeyHeld(k Key) bool {
	return s != nil && k >= 0 && k < NumKeys && s.Keys[k].Held
}

func (s *Snapshot) MousePressed(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Pressed
}

func (s *Snapshot) MouseReleased(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Released
}

func (s *Snapshot) MouseHeld(b MouseButton) bool {
	return s != nil && b >= 0 && b < NumMouseButtons && s.Mouse.Buttons[b].Held
}

func (s *Snapshot) Cursor() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Cursor
}

func (s *Snapshot) Wheel() geom.Vector2 {
	if s == nil {
		return geom.Vector2{}
	}
	return s.Mouse.Wheel
}

func (s *Snapshot) PadHeld(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Held
}

func (s *Snapshot) PadPressed(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Pressed
}

func (s *Snapshot) PadReleased(padIndex int, button PadButton) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads &&
		button >= 0 && button < NumPadButtons && s.Pads[padIndex].Buttons[button].Released
}

func (s *Snapshot) PadConnected(padIndex int) bool {
	return s != nil && padIndex >= 0 && padIndex < MaxPads && s.Pads[padIndex].Connected
}

func (s *Snapshot) PadAxis(padIndex int, axis PadAxis) float64 {
	if s != nil && padIndex >= 0 && padIndex < MaxPads && axis >= 0 && axis < NumPadAxes {
		return s.Pads[padIndex].Axes[axis]
	}
	return 0
}
