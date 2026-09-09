// Package input provides keyboard and mouse input handling with buffering for replay and rollback netcode.
package input

import (
	"maps"
	"time"

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

type Modifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
}

// InputSnapshot represents keyboard and mouse state for a single frame.
type InputSnapshot struct {
	Keyboard  map[ebiten.Key]KeyState
	Modifiers Modifiers
	Mouse     struct {
		X       int
		Y       int
		Buttons map[ebiten.MouseButton]KeyState
	}
}

// NewInputSnapshot creates a new InputSnapshot with initialized maps.
func NewInputSnapshot() InputSnapshot {
	return InputSnapshot{
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
func (i *InputSnapshot) Reset() {
	for key := range i.Keyboard {
		i.Keyboard[key] = KeyState{}
	}
	i.Modifiers.Shift = false
	i.Modifiers.Ctrl = false
	i.Modifiers.Alt = false
	for button := range i.Mouse.Buttons {
		i.Mouse.Buttons[button] = KeyState{}
	}
	i.Mouse.X = 0
	i.Mouse.Y = 0
}

// Clone returns a deep copy of the InputState.
// Maps are copied so buffered snapshots remain stable when currentState is updated.
func (i *InputSnapshot) Clone() InputSnapshot {
	cloned := *i
	cloned.Keyboard = make(map[ebiten.Key]KeyState)
	maps.Copy(cloned.Keyboard, i.Keyboard)
	cloned.Mouse.Buttons = make(map[ebiten.MouseButton]KeyState)
	maps.Copy(cloned.Mouse.Buttons, i.Mouse.Buttons)
	return cloned
}

func (i *InputSnapshot) Compare(other *InputSnapshot) bool {
	if i.Modifiers != other.Modifiers {
		return false
	}
	if i.Mouse.X != other.Mouse.X || i.Mouse.Y != other.Mouse.Y {
		return false
	}
	if len(i.Keyboard) != len(other.Keyboard) {
		return false
	}
	for key, state := range i.Keyboard {
		otherState, exists := other.Keyboard[key]
		if !exists || otherState != state {
			return false
		}
	}
	if len(i.Mouse.Buttons) != len(other.Mouse.Buttons) {
		return false
	}
	for button, state := range i.Mouse.Buttons {
		otherState, exists := other.Mouse.Buttons[button]
		if !exists || otherState != state {
			return false
		}
	}
	return true
}

// InputBuffer is a ring buffer storing up to size input snapshots.
// Used for input replay, rollback netcode, and replay recording.
type InputBuffer struct {
	states     []InputSnapshot
	head, tail int
	size       int
	count      int
}

// NewInputBuffer creates a new InputBuffer with given capacity.
//
// Returns a pointer to the newly created InputBuffer.
// If the provided size is less than or equal to zero, a default size of 60 is used.
func NewInputBuffer(size int) *InputBuffer {
	if size <= 0 {
		size = 60
	}
	return &InputBuffer{
		states: make([]InputSnapshot, size),
		size:   size,
	}
}

// Push adds state to buffer. If full, oldest state is evicted.
func (b *InputBuffer) Push(state InputSnapshot) {
	b.states[b.head] = state
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	} else {
		b.tail = (b.tail + 1) % b.size
	}
}

// Pop returns and removes the oldest buffered state. Returns false if empty.
func (b *InputBuffer) Pop() (InputSnapshot, bool) {
	if b.count == 0 {
		return InputSnapshot{}, false
	}
	state := b.states[b.tail]
	b.tail = (b.tail + 1) % b.size
	b.count--
	return state, true
}

// Peek returns the oldest buffered state without removing it. Returns false if empty.
func (b *InputBuffer) Peek() (InputSnapshot, bool) {
	if b.count == 0 {
		return InputSnapshot{}, false
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
	currentSnapshot InputSnapshot
	buffer          *InputBuffer
	lastFrameTime   time.Time
	justPressedKeys []ebiten.Key
	pressedKeys     []ebiten.Key
	releasedKeys    []ebiten.Key
}

// New creates a new InputHandler with a 60-frame input buffer.
func New() *InputHandler {
	return &InputHandler{
		currentSnapshot: NewInputSnapshot(),
		buffer:          NewInputBuffer(60),
	}
}

// Snapshot polls Ebiten once per frame, updates state, and buffers the result.
// Must be called once per frame before game logic updates.
func (ih *InputHandler) Snapshot() {
	now := time.Now()
	frameDelta := 0.0
	if !ih.lastFrameTime.IsZero() {
		frameDelta = now.Sub(ih.lastFrameTime).Seconds()
	}
	ih.lastFrameTime = now

	ih.justPressedKeys = inpututil.AppendJustPressedKeys(ih.justPressedKeys[:0])
	ih.pressedKeys = inpututil.AppendPressedKeys(ih.pressedKeys[:0])
	ih.releasedKeys = inpututil.AppendJustReleasedKeys(ih.releasedKeys[:0])

	// Reset frame-specific flags
	for key := range ih.currentSnapshot.Keyboard {
		state := ih.currentSnapshot.Keyboard[key]
		state.Pressed = false
		state.Released = false
		ih.currentSnapshot.Keyboard[key] = state
	}

	// Mark newly released keys
	for _, key := range ih.releasedKeys {
		ih.currentSnapshot.Keyboard[key] = KeyState{
			Pressed:  false,
			Held:     false,
			Released: true,
			Duration: 0,
		}
	}

	// Mark newly pressed keys
	for _, key := range ih.justPressedKeys {
		old := ih.currentSnapshot.Keyboard[key]
		ih.currentSnapshot.Keyboard[key] = KeyState{
			Pressed:  true,
			Held:     true,
			Released: false,
			Duration: old.Duration + frameDelta,
		}
	}

	// Update held keys (not just pressed)
	for _, key := range ih.pressedKeys {
		state := ih.currentSnapshot.Keyboard[key]
		if !state.Pressed {
			state.Held = true
			state.Duration += frameDelta
			ih.currentSnapshot.Keyboard[key] = state
		}
	}

	ih.currentSnapshot.Modifiers.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	ih.currentSnapshot.Modifiers.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl)
	ih.currentSnapshot.Modifiers.Alt = ebiten.IsKeyPressed(ebiten.KeyAlt)

	ih.currentSnapshot.Mouse.X, ih.currentSnapshot.Mouse.Y = ebiten.CursorPosition()

	// Reset mouse button frame-specific flags
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		state := ih.currentSnapshot.Mouse.Buttons[button]
		state.Pressed = false
		state.Released = false
		ih.currentSnapshot.Mouse.Buttons[button] = state
	}

	// Update mouse button states
	for button := ebiten.MouseButtonLeft; button <= ebiten.MouseButtonMax; button++ {
		old := ih.currentSnapshot.Mouse.Buttons[button]
		if ebiten.IsMouseButtonPressed(button) {
			ih.currentSnapshot.Mouse.Buttons[button] = KeyState{
				Pressed:  !old.Held,
				Held:     true,
				Released: false,
				Duration: old.Duration + frameDelta,
			}
		} else {
			ih.currentSnapshot.Mouse.Buttons[button] = KeyState{
				Pressed:  false,
				Held:     false,
				Released: old.Held,
				Duration: 0,
			}
		}
	}
	ih.buffer.Push(ih.currentSnapshot.Clone())
}

// KeyPressed reports whether key was just pressed this frame, optionally with modifiers.
func (ih *InputHandler) KeyPressed(key ebiten.Key, WithModifiers Modifiers) bool {
	return ih.currentSnapshot.Keyboard[key].Pressed &&
		(!WithModifiers.Ctrl || ih.currentSnapshot.Modifiers.Ctrl) &&
		(!WithModifiers.Shift || ih.currentSnapshot.Modifiers.Shift) &&
		(!WithModifiers.Alt || ih.currentSnapshot.Modifiers.Alt)
}

// KeyHeld reports whether key is currently held, optionally with modifiers.
func (ih *InputHandler) KeyHeld(key ebiten.Key, WithModifiers Modifiers) bool {
	return ih.currentSnapshot.Keyboard[key].Held &&
		(!WithModifiers.Ctrl || ih.currentSnapshot.Modifiers.Ctrl) &&
		(!WithModifiers.Shift || ih.currentSnapshot.Modifiers.Shift) &&
		(!WithModifiers.Alt || ih.currentSnapshot.Modifiers.Alt)
}

// KeyReleased reports whether key was just released this frame, optionally with modifiers.
func (ih *InputHandler) KeyReleased(key ebiten.Key, WithModifiers Modifiers) bool {
	return ih.currentSnapshot.Keyboard[key].Released &&
		(!WithModifiers.Ctrl || ih.currentSnapshot.Modifiers.Ctrl) &&
		(!WithModifiers.Shift || ih.currentSnapshot.Modifiers.Shift) &&
		(!WithModifiers.Alt || ih.currentSnapshot.Modifiers.Alt)
}

// MousePressed reports whether button was just pressed this frame.
func (ih *InputHandler) MousePressed(button ebiten.MouseButton) bool {
	return ih.currentSnapshot.Mouse.Buttons[button].Pressed
}

// MouseHeld reports whether button is currently held.
func (ih *InputHandler) MouseHeld(button ebiten.MouseButton) bool {
	return ih.currentSnapshot.Mouse.Buttons[button].Held
}

// MouseReleased reports whether button was just released this frame.
func (ih *InputHandler) MouseReleased(button ebiten.MouseButton) bool {
	return ih.currentSnapshot.Mouse.Buttons[button].Released
}

// MousePosition returns current cursor position.
func (ih *InputHandler) MousePosition() (x int, y int) {
	return ih.currentSnapshot.Mouse.X, ih.currentSnapshot.Mouse.Y
}

// Buffer returns the underlying input history buffer for replay or analysis.
func (ih *InputHandler) Buffer() *InputBuffer {
	return ih.buffer
}
