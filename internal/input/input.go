// Package input provides keyboard and mouse input handling with buffering for replay and rollback netcode.
package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// KeyState tracks the state of a key or mouse button across frames.
type KeyState struct {
	Pressed  bool    // True only on first frame of press
	Held     bool    // True while key is down
	Released bool    // True only on release frame
	Duration float64 // Seconds held (accumulates while Held is true)
}

// InputState represents keyboard and mouse state for a single frame.
type InputState struct {
	Keyboard map[ebiten.Key]KeyState
	Shift    bool
	Ctrl     bool
	Alt      bool
	Mouse    struct {
		X       int
		Y       int
		Buttons map[ebiten.MouseButton]KeyState
	}
}

// NewInputState creates a new InputState with initialized maps.
func NewInputState() InputState {
	return InputState{
		Keyboard: make(map[ebiten.Key]KeyState),
		Mouse: struct {
			X       int
			Y       int
			Buttons map[ebiten.MouseButton]KeyState
		}{
			Buttons: make(map[ebiten.MouseButton]KeyState),
		},
	}
}

// Reset clears all key and button states.
func (i *InputState) Reset() {
	for key := range i.Keyboard {
		i.Keyboard[key] = KeyState{}
	}
	i.Shift = false
	i.Ctrl = false
	i.Alt = false
	for button := range i.Mouse.Buttons {
		i.Mouse.Buttons[button] = KeyState{}
	}
}

// InputBuffer is a ring buffer storing up to size input snapshots.
// Used for input replay, rollback netcode, and replay recording.
type InputBuffer struct {
	states     []InputState
	head, tail int
	size       int
	count      int
}

// NewInputBuffer creates a new InputBuffer with given capacity.
func NewInputBuffer(size int) *InputBuffer {
	return &InputBuffer{
		states: make([]InputState, size),
		size:   size,
	}
}

// Push adds state to buffer. If full, oldest state is evicted.
func (b *InputBuffer) Push(state InputState) {
	b.states[b.head] = state
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	} else {
		b.tail = (b.tail + 1) % b.size
	}
}

// Pop returns and removes the oldest buffered state. Returns false if empty.
func (b *InputBuffer) Pop() (InputState, bool) {
	if b.count == 0 {
		return InputState{}, false
	}
	state := b.states[b.tail]
	b.tail = (b.tail + 1) % b.size
	b.count--
	return state, true
}

// Peek returns the oldest buffered state without removing it. Returns false if empty.
func (b *InputBuffer) Peek() (InputState, bool) {
	if b.count == 0 {
		return InputState{}, false
	}
	return b.states[b.tail], true
}

// IsEmpty reports whether the buffer contains no states.
func (b *InputBuffer) IsEmpty() bool {
	return b.count == 0
}

// IsFull reports whether the buffer is at capacity.
func (b *InputBuffer) IsFull() bool {
	return b.count == b.size
}

// Count returns the number of buffered states.
func (b *InputBuffer) Count() int {
	return b.count
}

// InputHandler polls and tracks keyboard and mouse input, buffering for replay.
type InputHandler struct {
	currentState InputState
	buffer       *InputBuffer
}

// New creates a new InputHandler with a 60-frame input buffer.
func New() *InputHandler {
	return &InputHandler{
		currentState: NewInputState(),
		buffer:       NewInputBuffer(60),
	}
}

// Snapshot polls Ebiten once per frame, updates state, and buffers the result.
// Must be called once per frame before game logic updates.
func (ih *InputHandler) Snapshot() {
	justPressedKeys := inpututil.AppendJustPressedKeys(nil)
	pressedKeys := inpututil.AppendPressedKeys(nil)
	releasedKeys := inpututil.AppendJustReleasedKeys(nil)

	// Reset frame-specific flags
	for key := range ih.currentState.Keyboard {
		state := ih.currentState.Keyboard[key]
		state.Pressed = false
		state.Released = false
		ih.currentState.Keyboard[key] = state
	}

	// Mark newly released keys
	for _, key := range releasedKeys {
		ih.currentState.Keyboard[key] = KeyState{
			Pressed:  false,
			Held:     false,
			Released: true,
			Duration: 0,
		}
	}

	// Mark newly pressed keys
	for _, key := range justPressedKeys {
		old := ih.currentState.Keyboard[key]
		ih.currentState.Keyboard[key] = KeyState{
			Pressed:  true,
			Held:     true,
			Released: false,
			Duration: old.Duration + 1.0/60.0,
		}
	}

	// Update held keys (not just pressed)
	for _, key := range pressedKeys {
		state := ih.currentState.Keyboard[key]
		if !state.Pressed {
			state.Held = true
			state.Duration += 1.0 / 60.0
			ih.currentState.Keyboard[key] = state
		}
	}

	ih.currentState.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	ih.currentState.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl)
	ih.currentState.Alt = ebiten.IsKeyPressed(ebiten.KeyAlt)

	ih.currentState.Mouse.X, ih.currentState.Mouse.Y = ebiten.CursorPosition()

	// Reset mouse button frame-specific flags
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		state := ih.currentState.Mouse.Buttons[button]
		state.Pressed = false
		state.Released = false
		ih.currentState.Mouse.Buttons[button] = state
	}

	// Update mouse button states
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		old := ih.currentState.Mouse.Buttons[button]
		if ebiten.IsMouseButtonPressed(button) {
			ih.currentState.Mouse.Buttons[button] = KeyState{
				Pressed:  !old.Held,
				Held:     true,
				Released: false,
				Duration: old.Duration + 1.0/60.0,
			}
		} else {
			ih.currentState.Mouse.Buttons[button] = KeyState{
				Pressed:  false,
				Held:     false,
				Released: old.Held,
				Duration: 0,
			}
		}
	}
	ih.buffer.Push(ih.currentState)
}

// KeyPressed reports whether key was just pressed this frame, optionally with modifiers.
func (ih *InputHandler) KeyPressed(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return ih.currentState.Keyboard[key].Pressed &&
		(!WithCtrl || ih.currentState.Ctrl) &&
		(!WithShift || ih.currentState.Shift) &&
		(!WithAlt || ih.currentState.Alt)
}

// KeyHeld reports whether key is currently held, optionally with modifiers.
func (ih *InputHandler) KeyHeld(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return ih.currentState.Keyboard[key].Held &&
		(!WithCtrl || ih.currentState.Ctrl) &&
		(!WithShift || ih.currentState.Shift) &&
		(!WithAlt || ih.currentState.Alt)
}

// KeyReleased reports whether key was just released this frame, optionally with modifiers.
func (ih *InputHandler) KeyReleased(key ebiten.Key, WithCtrl bool, WithShift bool, WithAlt bool) bool {
	return ih.currentState.Keyboard[key].Released &&
		(!WithCtrl || ih.currentState.Ctrl) &&
		(!WithShift || ih.currentState.Shift) &&
		(!WithAlt || ih.currentState.Alt)
}

// MousePressed reports whether button was just pressed this frame.
func (ih *InputHandler) MousePressed(button ebiten.MouseButton) bool {
	return ih.currentState.Mouse.Buttons[button].Pressed
}

// MouseHeld reports whether button is currently held.
func (ih *InputHandler) MouseHeld(button ebiten.MouseButton) bool {
	return ih.currentState.Mouse.Buttons[button].Held
}

// MouseReleased reports whether button was just released this frame.
func (ih *InputHandler) MouseReleased(button ebiten.MouseButton) bool {
	return ih.currentState.Mouse.Buttons[button].Released
}

// MousePosition returns current cursor position.
func (ih *InputHandler) MousePosition() (x int, y int) {
	return ih.currentState.Mouse.X, ih.currentState.Mouse.Y
}

// Buffer returns the underlying input history buffer for replay or analysis.
func (ih *InputHandler) Buffer() *InputBuffer {
	return ih.buffer
}
