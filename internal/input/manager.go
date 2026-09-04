package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/leonard-atorough/castrum/internal/core"
)

type Manager struct {
	currentState InputState
	buffer       *InputBuffer
}

func NewManager() *Manager {
	return &Manager{
		currentState: NewInputState(),
		buffer:       NewInputBuffer(64), // Example buffer size
	}
}

func (is *Manager) Update(world *core.World, delta float64) {

	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	pressedKeys := inpututil.AppendPressedKeys(nil)
	releasedKeys := inpututil.AppendJustReleasedKeys(nil)

	// Clear Pressed and Released on all previously-tracked keys
	// (both should only be true for one frame, not persist)
	for key := range is.currentState.Keyboard {
		state := is.currentState.Keyboard[key]
		state.Pressed = false
		state.Released = false
		is.currentState.Keyboard[key] = state
	}

	// Keys released this frame
	for _, key := range releasedKeys {
		is.currentState.Keyboard[key] = KeyState{
			Pressed:  false,
			Held:     false,
			Released: true,
			Duration: 0,
		}
	}

	// Keys pressed this frame (newly pressed, so Pressed=true for one frame only)
	for _, key := range justPressedKeys {
		old := is.currentState.Keyboard[key]
		is.currentState.Keyboard[key] = KeyState{
			Pressed:  true,
			Held:     true,
			Released: false,
			Duration: old.Duration + delta,
		}
	}

	// Keys being held (but not just pressed) - keep Held=true, Pressed already cleared
	for _, key := range pressedKeys {
		state := is.currentState.Keyboard[key]
		if !state.Pressed { // Skip if this was just pressed (already handled above)
			state.Held = true
			is.currentState.Keyboard[key] = state
		}
	}

	is.currentState.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	is.currentState.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl)
	is.currentState.Alt = ebiten.IsKeyPressed(ebiten.KeyAlt)

	is.currentState.Mouse.X, is.currentState.Mouse.Y = ebiten.CursorPosition()

	// Clear Pressed and Released on all mouse buttons (like keyboard keys)
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		state := is.currentState.Mouse.Buttons[button]
		state.Pressed = false
		state.Released = false
		is.currentState.Mouse.Buttons[button] = state
	}

	// Update mouse button states based on current press state
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		if ebiten.IsMouseButtonPressed(button) {
			// Button is currently pressed
			old := is.currentState.Mouse.Buttons[button]
			// Pressed is true only if this is the first frame (old.Held was false)
			is.currentState.Mouse.Buttons[button] = KeyState{
				Pressed:  !old.Held,
				Held:     true,
				Released: false,
				Duration: old.Duration + delta,
			}
		} else {
			// Button is not pressed
			old := is.currentState.Mouse.Buttons[button]
			is.currentState.Mouse.Buttons[button] = KeyState{
				Pressed:  false,
				Held:     false,
				Released: old.Held, // Released is true if it WAS held last frame
				Duration: 0,
			}
		}
	}
	// Push the current state to the buffer at the end of the update
	is.buffer.Push(is.currentState)
}

func (is *Manager) KeyPressed(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.currentState.Keyboard[key].Pressed &&
		(!WithCtrl || is.currentState.Ctrl) &&
		(!WithShift || is.currentState.Shift) &&
		(!WithAlt || is.currentState.Alt)
}

func (is *Manager) KeyHeld(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.currentState.Keyboard[key].Held &&
		(!WithCtrl || is.currentState.Ctrl) &&
		(!WithShift || is.currentState.Shift) &&
		(!WithAlt || is.currentState.Alt)
}

func (is *Manager) KeyReleased(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return is.currentState.Keyboard[key].Released &&
		(!WithCtrl || is.currentState.Ctrl) &&
		(!WithShift || is.currentState.Shift) &&
		(!WithAlt || is.currentState.Alt)
}

func (is *Manager) MousePressed(button ebiten.MouseButton) bool {
	return is.currentState.Mouse.Buttons[button].Pressed
}

func (is *Manager) MouseHeld(button ebiten.MouseButton) bool {
	return is.currentState.Mouse.Buttons[button].Held
}

func (is *Manager) MouseReleased(button ebiten.MouseButton) bool {
	return is.currentState.Mouse.Buttons[button].Released
}

func (is *Manager) MousePosition() (x int, y int) {
	return is.currentState.Mouse.X, is.currentState.Mouse.Y
}

func (is *Manager) InputBuffer() *InputBuffer {
	return is.buffer
}
