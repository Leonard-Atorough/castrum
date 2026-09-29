package ebitrun

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
)

var keyTable = map[input.Key]ebiten.Key{
	input.KeyA: ebiten.KeyA,
	input.KeyB: ebiten.KeyB,
	input.KeyC: ebiten.KeyC,
	input.KeyD: ebiten.KeyD,
	input.KeyE: ebiten.KeyE,
	input.KeyF: ebiten.KeyF,
	input.KeyG: ebiten.KeyG,
	input.KeyH: ebiten.KeyH,
	input.KeyI: ebiten.KeyI,
	input.KeyJ: ebiten.KeyJ,
	input.KeyK: ebiten.KeyK,
	input.KeyL: ebiten.KeyL,
	input.KeyM: ebiten.KeyM,
	input.KeyN: ebiten.KeyN,
	input.KeyO: ebiten.KeyO,
	input.KeyP: ebiten.KeyP,
	input.KeyQ: ebiten.KeyQ,
	input.KeyR: ebiten.KeyR,
	input.KeyS: ebiten.KeyS,
	input.KeyT: ebiten.KeyT,
	input.KeyU: ebiten.KeyU,
	input.KeyV: ebiten.KeyV,
	input.KeyW: ebiten.KeyW,
	input.KeyX: ebiten.KeyX,
	input.KeyY: ebiten.KeyY,
	input.KeyZ: ebiten.KeyZ,
	input.Key0: ebiten.Key0,
	input.Key1: ebiten.Key1,
	input.Key2: ebiten.Key2,
	input.Key3: ebiten.Key3,
	input.Key4: ebiten.Key4,
	input.Key5: ebiten.Key5,
	input.Key6: ebiten.Key6,
	input.Key7: ebiten.Key7,
	input.Key8: ebiten.Key8,
	input.Key9: ebiten.Key9,
	// The naming seams - what the table exists for.
	input.KeyBacktick:     ebiten.KeyBackquote,
	input.KeyBracketLeft:  ebiten.KeyBracketLeft,
	input.KeyBracketRight: ebiten.KeyBracketRight,
	input.KeyISOBackslash: ebiten.KeyIntlBackslash,
	input.KeyMinus:        ebiten.KeyMinus,
	input.KeyEqual:        ebiten.KeyEqual,
	input.KeyBackslash:    ebiten.KeyBackslash,
	input.KeySemicolon:    ebiten.KeySemicolon,
	input.KeyApostrophe:   ebiten.KeyApostrophe,
	input.KeyComma:        ebiten.KeyComma,
	input.KeyPeriod:       ebiten.KeyPeriod,
	input.KeySlash:        ebiten.KeySlash,
	// Control row, modifiers, function row, navigation, arrows:
	// the names agree.
	input.KeySpace:          ebiten.KeySpace,
	input.KeyEscape:         ebiten.KeyEscape,
	input.KeyTab:            ebiten.KeyTab,
	input.KeyCapsLock:       ebiten.KeyCapsLock,
	input.KeyEnter:          ebiten.KeyEnter,
	input.KeyBackspace:      ebiten.KeyBackspace,
	input.KeyShiftLeft:      ebiten.KeyShiftLeft,
	input.KeyShiftRight:     ebiten.KeyShiftRight,
	input.KeyControlLeft:    ebiten.KeyControlLeft,
	input.KeyControlRight:   ebiten.KeyControlRight,
	input.KeyAltLeft:        ebiten.KeyAltLeft,
	input.KeyAltRight:       ebiten.KeyAltRight,
	input.KeyF1:             ebiten.KeyF1,
	input.KeyF2:             ebiten.KeyF2,
	input.KeyF3:             ebiten.KeyF3,
	input.KeyF4:             ebiten.KeyF4,
	input.KeyF5:             ebiten.KeyF5,
	input.KeyF6:             ebiten.KeyF6,
	input.KeyF7:             ebiten.KeyF7,
	input.KeyF8:             ebiten.KeyF8,
	input.KeyF9:             ebiten.KeyF9,
	input.KeyF10:            ebiten.KeyF10,
	input.KeyF11:            ebiten.KeyF11,
	input.KeyF12:            ebiten.KeyF12,
	input.KeyInsert:         ebiten.KeyInsert,
	input.KeyDelete:         ebiten.KeyDelete,
	input.KeyHome:           ebiten.KeyHome,
	input.KeyEnd:            ebiten.KeyEnd,
	input.KeyPageUp:         ebiten.KeyPageUp,
	input.KeyPageDown:       ebiten.KeyPageDown,
	input.KeyPrintScreen:    ebiten.KeyPrintScreen,
	input.KeyScrollLock:     ebiten.KeyScrollLock,
	input.KeyPause:          ebiten.KeyPause,
	input.KeyArrowUp:        ebiten.KeyArrowUp,
	input.KeyArrowDown:      ebiten.KeyArrowDown,
	input.KeyArrowLeft:      ebiten.KeyArrowLeft,
	input.KeyArrowRight:     ebiten.KeyArrowRight,
	input.KeyNumpad0:        ebiten.KeyNumpad0,
	input.KeyNumpad1:        ebiten.KeyNumpad1,
	input.KeyNumpad2:        ebiten.KeyNumpad2,
	input.KeyNumpad3:        ebiten.KeyNumpad3,
	input.KeyNumpad4:        ebiten.KeyNumpad4,
	input.KeyNumpad5:        ebiten.KeyNumpad5,
	input.KeyNumpad6:        ebiten.KeyNumpad6,
	input.KeyNumpad7:        ebiten.KeyNumpad7,
	input.KeyNumpad8:        ebiten.KeyNumpad8,
	input.KeyNumpad9:        ebiten.KeyNumpad9,
	input.KeyNumpadDecimal:  ebiten.KeyNumpadDecimal,
	input.KeyNumpadEnter:    ebiten.KeyNumpadEnter,
	input.KeyNumpadAdd:      ebiten.KeyNumpadAdd,
	input.KeyNumpadSubtract: ebiten.KeyNumpadSubtract,
	input.KeyNumpadMultiply: ebiten.KeyNumpadMultiply,
	input.KeyNumpadDivide:   ebiten.KeyNumpadDivide,
	input.KeyNumLock:        ebiten.KeyNumLock,
}

var mouseButtonTable = map[input.MouseButton]ebiten.MouseButton{
	input.MouseButtonLeft:   ebiten.MouseButtonLeft,
	input.MouseButtonRight:  ebiten.MouseButtonRight,
	input.MouseButtonMiddle: ebiten.MouseButtonMiddle,
}

// ebiten names standard buttons by the W3C gamepad grid, so this table also
// translates naming schemes.
var padButtonTable = map[input.PadButton]ebiten.StandardGamepadButton{
	input.PadSouth:         ebiten.StandardGamepadButtonRightBottom,
	input.PadEast:          ebiten.StandardGamepadButtonRightRight,
	input.PadWest:          ebiten.StandardGamepadButtonRightLeft,
	input.PadNorth:         ebiten.StandardGamepadButtonRightTop,
	input.PadLeftShoulder:  ebiten.StandardGamepadButtonFrontTopLeft,
	input.PadRightShoulder: ebiten.StandardGamepadButtonFrontTopRight,
	input.PadLeftStick:     ebiten.StandardGamepadButtonLeftStick,
	input.PadRightStick:    ebiten.StandardGamepadButtonRightStick,
	input.PadBack:          ebiten.StandardGamepadButtonCenterLeft,
	input.PadStart:         ebiten.StandardGamepadButtonCenterRight,
	input.PadGuide:         ebiten.StandardGamepadButtonCenterCenter,
	input.PadDPadUp:        ebiten.StandardGamepadButtonLeftTop,
	input.PadDPadDown:      ebiten.StandardGamepadButtonLeftBottom,
	input.PadDPadLeft:      ebiten.StandardGamepadButtonLeftLeft,
	input.PadDPadRight:     ebiten.StandardGamepadButtonLeftRight,
}

// Stick axes map to ebiten's standard axes. ebiten models triggers
// as analog buttons, so they fill through triggerTable instead: the
// input contract (triggers are 0..1 axes) is unchanged, the poller
// translates.
var padAxisTable = map[input.PadAxis]ebiten.StandardGamepadAxis{
	input.PadLeftStickX:  ebiten.StandardGamepadAxisLeftStickHorizontal,
	input.PadLeftStickY:  ebiten.StandardGamepadAxisLeftStickVertical,
	input.PadRightStickX: ebiten.StandardGamepadAxisRightStickHorizontal,
	input.PadRightStickY: ebiten.StandardGamepadAxisRightStickVertical,
}

var triggerTable = map[input.PadAxis]ebiten.StandardGamepadButton{
	input.PadLeftTrigger:  ebiten.StandardGamepadButtonFrontBottomLeft,
	input.PadRightTrigger: ebiten.StandardGamepadButtonFrontBottomRight,
}

// poller builds the frame's Snapshot and owns it across frames. The
// runner polls before Advance, so PhaseFrame systems read this
// frame's device state.
type poller struct {
	snapshot input.Snapshot
	ids      []ebiten.GamepadID
}

// poll clears and updates the input snapshot for the current frame.
// the snapshot is reused across frames, so it must be cleared and repopulated each frame.
func (p *poller) poll() {
	p.snapshot = input.Snapshot{}
	p.pollKeys()
	p.pollMouse()
	p.pollPads()
}

// snapshotPtr returns a pointer to the current input snapshot.
func (p *poller) snapshotPtr() *input.Snapshot {
	return &p.snapshot
}

func (p *poller) pollKeys() {
	for key, ekey := range keyTable {
		p.snapshot.Keys[key] = input.State{
			Pressed:  inpututil.IsKeyJustPressed(ekey),
			Held:     ebiten.IsKeyPressed(ekey),
			Released: inpututil.IsKeyJustReleased(ekey),
		}
	}
}

func (p *poller) pollMouse() {
	x, y := ebiten.CursorPosition()
	p.snapshot.Mouse.Cursor = geom.Vector2{X: float64(x), Y: float64(y)}
	wheelX, wheelY := ebiten.Wheel()
	p.snapshot.Mouse.Wheel = geom.Vector2{X: wheelX, Y: wheelY}
	for button, ebutton := range mouseButtonTable {
		p.snapshot.Mouse.Buttons[button] = input.State{
			Pressed:  inpututil.IsMouseButtonJustPressed(ebutton),
			Held:     ebiten.IsMouseButtonPressed(ebutton),
			Released: inpututil.IsMouseButtonJustReleased(ebutton),
		}
	}
}

func (p *poller) pollPads() {
	p.ids = ebiten.AppendGamepadIDs(p.ids[:0])
	for slot, id := range p.ids {
		if slot >= input.MaxPads {
			break
		}
		pad := &p.snapshot.Pads[slot]
		pad.Connected = true
		for button, ebutton := range padButtonTable {
			pad.Buttons[button] = input.State{
				Pressed:  inpututil.IsStandardGamepadButtonJustPressed(id, ebutton),
				Held:     ebiten.IsStandardGamepadButtonPressed(id, ebutton),
				Released: inpututil.IsStandardGamepadButtonJustReleased(id, ebutton),
			}
		}
		for axis, eaxis := range padAxisTable {
			pad.Axes[axis] = ebiten.StandardGamepadAxisValue(id, eaxis)
		}
		for axis, ebutton := range triggerTable {
			pad.Axes[axis] = ebiten.StandardGamepadButtonValue(id, ebutton)
		}
	}
}
