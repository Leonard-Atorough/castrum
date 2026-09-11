package input

import "maps"

// KeyState tracks the state of a physical input across frames.
type KeyState struct {
	Pressed  bool
	Held     bool
	Released bool
	Duration float64
}

// Modifiers describes modifier keys held with a physical input.
type Modifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
}

// InputSnapshot represents backend-independent physical input for one frame.
type InputSnapshot struct {
	Keyboard  map[string]KeyState
	Modifiers Modifiers
	Mouse     MouseState
}

// MouseState contains the cursor position and named mouse-button states.
type MouseState struct {
	X       int
	Y       int
	Buttons map[string]KeyState
}

// NewInputSnapshot creates an initialized physical input snapshot.
func NewInputSnapshot() InputSnapshot {
	return InputSnapshot{
		Keyboard: make(map[string]KeyState),
		Mouse: MouseState{
			Buttons: make(map[string]KeyState),
		},
	}
}

// Reset clears all physical input state while retaining allocated maps.
func (i *InputSnapshot) Reset() {
	for key := range i.Keyboard {
		i.Keyboard[key] = KeyState{}
	}
	i.Modifiers = Modifiers{}
	for button := range i.Mouse.Buttons {
		i.Mouse.Buttons[button] = KeyState{}
	}
	i.Mouse.X = 0
	i.Mouse.Y = 0
}

// Clone returns a deep copy suitable for storing in an input history buffer.
func (i *InputSnapshot) Clone() InputSnapshot {
	cloned := *i
	cloned.Keyboard = make(map[string]KeyState, len(i.Keyboard))
	maps.Copy(cloned.Keyboard, i.Keyboard)
	cloned.Mouse.Buttons = make(map[string]KeyState, len(i.Mouse.Buttons))
	maps.Copy(cloned.Mouse.Buttons, i.Mouse.Buttons)
	return cloned
}

// Compare reports whether two snapshots contain identical physical state.
func (i *InputSnapshot) Compare(other *InputSnapshot) bool {
	if i.Modifiers != other.Modifiers || i.Mouse.X != other.Mouse.X || i.Mouse.Y != other.Mouse.Y {
		return false
	}
	return compareStates(i.Keyboard, other.Keyboard) && compareStates(i.Mouse.Buttons, other.Mouse.Buttons)
}

func compareStates(left, right map[string]KeyState) bool {
	if len(left) != len(right) {
		return false
	}
	for key, state := range left {
		if right[key] != state {
			return false
		}
	}
	return true
}

// InputBuffer stores a bounded history of physical input snapshots.
type InputBuffer struct {
	states     []InputSnapshot
	head, tail int
	size       int
	count      int
}

// NewInputBuffer creates a buffer with the requested capacity, defaulting to 60.
func NewInputBuffer(size int) *InputBuffer {
	if size <= 0 {
		size = 60
	}
	return &InputBuffer{states: make([]InputSnapshot, size), size: size}
}

// Push adds a snapshot, evicting the oldest snapshot when full.
func (b *InputBuffer) Push(state InputSnapshot) {
	b.states[b.head] = state
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	} else {
		b.tail = (b.tail + 1) % b.size
	}
}

// Pop removes and returns the oldest snapshot.
func (b *InputBuffer) Pop() (InputSnapshot, bool) {
	if b.count == 0 {
		return InputSnapshot{}, false
	}
	state := b.states[b.tail]
	b.tail = (b.tail + 1) % b.size
	b.count--
	return state, true
}

// Peek returns the oldest snapshot without removing it.
func (b *InputBuffer) Peek() (InputSnapshot, bool) {
	if b.count == 0 {
		return InputSnapshot{}, false
	}
	return b.states[b.tail], true
}

// IsEmpty reports whether the buffer contains no snapshots.
func (b *InputBuffer) IsEmpty() bool { return b.count == 0 }

// IsFull reports whether the buffer has reached capacity.
func (b *InputBuffer) IsFull() bool { return b.count == b.size }

// Count reports the number of snapshots currently stored.
func (b *InputBuffer) Count() int { return b.count }
