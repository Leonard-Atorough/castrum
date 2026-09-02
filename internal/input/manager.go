package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/leonard-atorough/castrum/internal/core"
)

type Manager struct {
	state InputState
}

func NewManager() *Manager {
	return &Manager{
		state: NewInputState(),
	}
}

func (is *Manager) Update(world *core.World, delta float64) {

	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	pressedKeys := inpututil.AppendPressedKeys(nil)
	releasedKeys := inpututil.AppendJustReleasedKeys(nil)

	// Clear Pressed and Released on all previously-tracked keys
	// (both should only be true for one frame, not persist)
	for key := range is.state.Keyboard {
		state := is.state.Keyboard[key]
		state.Pressed = false
		state.Released = false
		is.state.Keyboard[key] = state
	}

	// Keys released this frame
	for _, key := range releasedKeys {
		is.state.Keyboard[key] = KeyState{
			Pressed:  false,
			Held:     false,
			Released: true,
			Duration: 0,
		}
	}

	// Keys pressed this frame (newly pressed, so Pressed=true for one frame only)
	for _, key := range justPressedKeys {
		old := is.state.Keyboard[key]
		is.state.Keyboard[key] = KeyState{
			Pressed:  true,
			Held:     true,
			Released: false,
			Duration: old.Duration + delta,
		}
	}

	// Keys being held (but not just pressed) - keep Held=true, Pressed already cleared
	for _, key := range pressedKeys {
		state := is.state.Keyboard[key]
		if !state.Pressed { // Skip if this was just pressed (already handled above)
			state.Held = true
			is.state.Keyboard[key] = state
		}
	}

	is.state.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	is.state.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl)
	is.state.Alt = ebiten.IsKeyPressed(ebiten.KeyAlt)

	is.state.Mouse.X, is.state.Mouse.Y = ebiten.CursorPosition()

	// Clear Pressed and Released on all mouse buttons (like keyboard keys)
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		state := is.state.Mouse.Buttons[button]
		state.Pressed = false
		state.Released = false
		is.state.Mouse.Buttons[button] = state
	}

	// Update mouse button states based on current press state
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		if ebiten.IsMouseButtonPressed(button) {
			// Button is currently pressed
			old := is.state.Mouse.Buttons[button]
			// Pressed is true only if this is the first frame (old.Held was false)
			is.state.Mouse.Buttons[button] = KeyState{
				Pressed:  !old.Held,
				Held:     true,
				Released: false,
				Duration: old.Duration + delta,
			}
		} else {
			// Button is not pressed
			old := is.state.Mouse.Buttons[button]
			is.state.Mouse.Buttons[button] = KeyState{
				Pressed:  false,
				Held:     false,
				Released: old.Held, // Released is true if it WAS held last frame
				Duration: 0,
			}
		}
	}
}

func (is *Manager) KeyPressed(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.state.Keyboard[key].Pressed &&
		(!WithCtrl || is.state.Ctrl) &&
		(!WithShift || is.state.Shift) &&
		(!WithAlt || is.state.Alt)
}

func (is *Manager) KeyHeld(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.state.Keyboard[key].Held &&
		(!WithCtrl || is.state.Ctrl) &&
		(!WithShift || is.state.Shift) &&
		(!WithAlt || is.state.Alt)
}

func (is *Manager) KeyReleased(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.state.Keyboard[key].Released &&
		(!WithCtrl || is.state.Ctrl) &&
		(!WithShift || is.state.Shift) &&
		(!WithAlt || is.state.Alt)
}

func (is *Manager) MousePressed(button ebiten.MouseButton) bool {
	return is.state.Mouse.Buttons[button].Pressed
}

func (is *Manager) MouseHeld(button ebiten.MouseButton) bool {
	return is.state.Mouse.Buttons[button].Held
}

func (is *Manager) MouseReleased(button ebiten.MouseButton) bool {
	return is.state.Mouse.Buttons[button].Released
}

func (is *Manager) MousePosition() (x int, y int) {
	return is.state.Mouse.X, is.state.Mouse.Y
}

/*
Adding an input buffer:
- Store recent input events in a buffer for later processing
- Process buffered events in the order they occurred
- Implement a fixed-size buffer to limit memory usage
- Handle buffer overflow by discarding oldest events
- Ensure that input events are processed consistently each frame
- Integrate the input buffer with the existing input manager
*/
