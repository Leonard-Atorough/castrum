// Package input provides types and constants for handling keyboard, mouse, and gamepad input.
// It is backend/runner free and provides a unified interface for input handling.

package input

import "fmt"

// Key represents a keyboard key. The provided constants cover standard
// alphanumeric, control, function, navigation, and numpad keys.
// it follows the ISO/IEC 9995 standard.
type Key int

// BASED ON ISO/IEC 9995 standard for keyboard layouts and categories
const (
	// KeyNone is the zero value of Key: no key at all. Modifier slots
	// read as empty, a binding with KeyNone never fires, and the
	// poller never reports it.
	KeyNone Key = iota
	// Alphanumeric keys
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyBacktick
	KeyMinus
	KeyEqual
	KeyBracketLeft
	KeyBracketRight
	KeyBackslash
	KeySemicolon
	KeyApostrophe
	KeyComma
	KeyPeriod
	KeySlash
	// Supporting ISO/IEC 9995 standard backslash key
	KeyISOBackslash
	// Space and the control row
	KeySpace
	KeyEscape
	KeyTab
	KeyCapsLock
	KeyEnter
	KeyBackspace
	// Modifiers
	KeyShiftLeft
	KeyShiftRight
	KeyControlLeft
	KeyControlRight
	KeyAltLeft
	KeyAltRight
	// Function Row
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	// Navigation cluster
	KeyInsert
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyPrintScreen
	KeyScrollLock
	KeyPause
	// Arrow keys
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	// Numpad keys
	KeyNumpad0
	KeyNumpad1
	KeyNumpad2
	KeyNumpad3
	KeyNumpad4
	KeyNumpad5
	KeyNumpad6
	KeyNumpad7
	KeyNumpad8
	KeyNumpad9
	KeyNumpadDecimal
	KeyNumpadEnter
	KeyNumpadAdd
	KeyNumpadSubtract
	KeyNumpadMultiply
	KeyNumpadDivide
	KeyNumLock
	NumKeys // sizes the number of keys
)

var keyNames = [NumKeys]string{
	"none",
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"backtick", "minus", "equal", "bracket_left", "bracket_right", "backslash", "semicolon", "apostrophe", "comma", "period", "slash",
	"iso_backslash",
	"space", "escape", "tab", "caps_lock", "enter", "backspace",
	"shift_left", "shift_right", "control_left", "control_right", "alt_left", "alt_right",
	"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
	"insert", "delete", "home", "end", "page_up", "page_down", "print_screen", "scroll_lock", "pause",
	"arrow_up", "arrow_down", "arrow_left", "arrow_right",
	"numpad_0", "numpad_1", "numpad_2", "numpad_3", "numpad_4", "numpad_5", "numpad_6", "numpad_7", "numpad_8", "numpad_9",
	"numpad_decimal", "numpad_enter", "numpad_add", "numpad_subtract", "numpad_multiply", "numpad_divide", "num_lock",
}

func (k Key) String() string {
	if k >= 0 && k < NumKeys {
		return keyNames[k]
	}
	return fmt.Sprintf("Key(%d)", int(k))
}

// MouseButton represents a mouse button. The provided constants cover the
// standard left, right, and middle buttons.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	NumMouseButtons // sizes the number of mouse buttons
)

var mouseButtonNames = [NumMouseButtons]string{
	"left", "right", "middle",
}

func (b MouseButton) String() string {
	if b >= 0 && b < NumMouseButtons {
		return mouseButtonNames[b]
	}
	return fmt.Sprintf("MouseButton(%d)", int(b))
}

// PadButton identifies a gamepad button by the standard layout's
// position names - south/east/west/north, not vendor glyphs (A/B/X/Y)
// - so bindings survive whatever controller is plugged in. The
// triggers are axes, not buttons (see [PadAxis]).
type PadButton int

const (
	PadSouth PadButton = iota
	PadEast
	PadWest
	PadNorth
	PadLeftShoulder
	PadRightShoulder
	PadLeftStick
	PadRightStick
	PadBack
	PadStart
	PadGuide
	PadDPadUp
	PadDPadDown
	PadDPadLeft
	PadDPadRight
	NumPadButtons // sizes the number of pad buttons
)

var padButtonNames = [NumPadButtons]string{
	"south", "east", "west", "north",
	"left_shoulder", "right_shoulder",
	"left_stick", "right_stick",
	"back", "start", "guide",
	"dpad_up", "dpad_down", "dpad_left", "dpad_right",
}

func (b PadButton) String() string {
	if b >= 0 && b < NumPadButtons {
		return padButtonNames[b]
	}
	return fmt.Sprintf("PadButton(%d)", int(b))
}

// PadAxis identifies a gamepad axis: sticks span -1..1 with Y
// positive downward - the screen convention - and triggers span 0..1.
// The poller normalizes backend values to this contract in its
// device table.
type PadAxis int

const (
	PadLeftStickX PadAxis = iota
	PadLeftStickY
	PadRightStickX
	PadRightStickY
	PadLeftTrigger
	PadRightTrigger
	NumPadAxes // sizes the number of pad axes
)

var padAxisNames = [NumPadAxes]string{
	"left_stick_x", "left_stick_y",
	"right_stick_x", "right_stick_y",
	"left_trigger", "right_trigger",
}

func (a PadAxis) String() string {
	if a >= 0 && a < NumPadAxes {
		return padAxisNames[a]
	}
	return fmt.Sprintf("PadAxis(%d)", int(a))
}

// State is one frame of one physical input: Pressed and Released are
// this-frame edges, Held is the level. Snapshots carry one per input;
// duration is an action-level accumulation, not stored here.
type State struct{ Pressed, Held, Released bool }

// Input is the interface implemented by all bindable physical inputs.
// Every type that can be bound to an action implements this interface.
type Input interface{ isInput() }

// KeyInput binds an action to a keyboard key, with optional modifier
// keys that must be held. Ctrl+S is
// KeyInput{Key: KeyS, Modifiers: [3]Key{KeyControlLeft}}.
type KeyInput struct {
	// Key is the trigger: Pressed, Held, and Released follow this key
	// while every modifier is held. A Key of KeyNone never fires.
	Key Key
	// Modifiers lists keys that must be held for Key to count.
	// Empty slots (KeyNone) are ignored, so the zero value binds the
	// plain key. A modifier dropped mid-hold stops the action without
	// a release edge. A modifier equal to Key suppresses the release
	// edge: the key is no longer held on its own release frame.
	Modifiers [3]Key
}

// MouseButtonInput binds an action to a mouse button.
type MouseButtonInput struct{ Button MouseButton }

// PadButtonInput binds an action to a gamepad button.
type PadButtonInput struct {
	// Pad is the player number: 0 - the zero value - means any
	// connected pad, 1 the first pad, 2 the second.
	Pad int
	// Button is the gamepad button that triggers the action.
	Button PadButton
}

// PadAxisInput binds an action to a gamepad axis.
type PadAxisInput struct {
	// Pad is the player number: 0 - the zero value - means any
	// connected pad, 1 the first pad, 2 the second.
	Pad int
	// Axis is the gamepad axis that triggers the action.
	Axis PadAxis
	// Direction gates which deflection drives the binding: 0 -
	// the zero value - either direction, -1 only the negative half
	// (screen-up for a Y axis), +1 only the positive half.
	Direction int
}

// KeyPairInput binds an axis from two keyboard keys,
// one for the negative direction and one for the positive direction.
type KeyPairInput struct{ Negative, Positive Key }

func (KeyInput) isInput()         {}
func (MouseButtonInput) isInput() {}
func (PadButtonInput) isInput()   {}
func (PadAxisInput) isInput()     {}
func (KeyPairInput) isInput()     {}
