package input

import "testing"

func TestNewInputSnapshot(t *testing.T) {
	snapshot := NewInputSnapshot()
	if snapshot.Keyboard == nil {
		t.Errorf("Expected Keyboard map to be initialized")
	}
	if snapshot.Mouse.Buttons == nil {
		t.Errorf("Expected Mouse Buttons map to be initialized")
	}
	if snapshot.Mouse.X != 0 || snapshot.Mouse.Y != 0 {
		t.Errorf("Expected Mouse X and Y to be initialized to 0")
	}
	if snapshot.Modifiers.Shift || snapshot.Modifiers.Ctrl || snapshot.Modifiers.Alt {
		t.Errorf("Expected Modifiers to be initialized to false")
	}
}

func TestInputSnapshotReset(t *testing.T) {
	snapshot := NewInputSnapshot()
	snapshot.Keyboard[0] = KeyState{Pressed: true, Held: true, Released: true, Duration: 1.0}
	snapshot.Modifiers.Shift = true
	snapshot.Modifiers.Ctrl = true
	snapshot.Modifiers.Alt = true
	snapshot.Mouse.X = 100
	snapshot.Mouse.Y = 200
	snapshot.Mouse.Buttons[0] = KeyState{Pressed: true, Held: true, Released: true, Duration: 1.0}

	snapshot.Reset()

	for key, state := range snapshot.Keyboard {
		if state.Pressed || state.Held || state.Released || state.Duration != 0 {
			t.Errorf("Expected Keyboard[%v] to be reset", key)
		}
	}
	if snapshot.Modifiers.Shift || snapshot.Modifiers.Ctrl || snapshot.Modifiers.Alt {
		t.Errorf("Expected Modifiers to be reset")
	}
	if snapshot.Mouse.X != 0 || snapshot.Mouse.Y != 0 {
		t.Errorf("Expected Mouse X and Y to be reset to 0")
	}
	for button, state := range snapshot.Mouse.Buttons {
		if state.Pressed || state.Held || state.Released || state.Duration != 0 {
			t.Errorf("Expected Mouse.Buttons[%v] to be reset", button)
		}
	}
}

func TestInputSnapshotClone(t *testing.T) {
	original := NewInputSnapshot()
	original.Keyboard[0] = KeyState{Pressed: true, Held: true, Released: true, Duration: 1.0}
	original.Modifiers.Shift = true
	original.Modifiers.Ctrl = true
	original.Modifiers.Alt = true
	original.Mouse.X = 100
	original.Mouse.Y = 200
	original.Mouse.Buttons[0] = KeyState{Pressed: true, Held: true, Released: true, Duration: 1.0}

	cloned := original.Clone()

	// Verify that the cloned snapshot matches the original
	for key, state := range original.Keyboard {
		clonedState, exists := cloned.Keyboard[key]
		if !exists || clonedState != state {
			t.Errorf("Expected cloned Keyboard[%v] to match original", key)
		}
	}
	if cloned.Modifiers != original.Modifiers {
		t.Errorf("Expected cloned Modifiers to match original")
	}
	if cloned.Mouse.X != original.Mouse.X || cloned.Mouse.Y != original.Mouse.Y {
		t.Errorf("Expected cloned Mouse X and Y to match original")
	}
	for button, state := range original.Mouse.Buttons {
		clonedState, exists := cloned.Mouse.Buttons[button]
		if !exists || clonedState != state {
			t.Errorf("Expected cloned Mouse.Buttons[%v] to match original", button)
		}
	}

	// Modify the cloned snapshot and verify it does not affect the original
	cloned.Keyboard[0] = KeyState{}
	cloned.Modifiers.Shift = false
	cloned.Mouse.X = 0
	cloned.Mouse.Buttons[0] = KeyState{}

	if original.Keyboard[0].Pressed == false || original.Modifiers.Shift == false || original.Mouse.X == 0 || original.Mouse.Buttons[0].Pressed == false {
		t.Errorf("Modifying cloned snapshot should not affect the original")
	}
}

func TestInputSnapshotCompare(t *testing.T) {
	snapshot1 := NewInputSnapshot()
	snapshot2 := snapshot1.Clone()

	if !snapshot1.Compare(&snapshot2) {
		t.Errorf("Expected snapshots to be equal")
	}

	snapshot2.Keyboard[0] = KeyState{Pressed: true}
	if snapshot1.Compare(&snapshot2) {
		t.Errorf("Expected snapshots to be different after modification")
	}
}

func TestNewInputBuffer(t *testing.T) {
	buffer := NewInputBuffer(5)
	if buffer == nil {
		t.Errorf("Expected NewInputBuffer to return a non-nil buffer")
	}
	if buffer.size != 5 {
		t.Errorf("Expected buffer size to be 5, got %d", buffer.size)
	}
	if buffer.Count() != 0 {
		t.Errorf("Expected buffer count to be 0, got %d", buffer.Count())
	}
}

func TestNewInputBuffer_DefaultSize(t *testing.T) {
	buffer := NewInputBuffer(0)
	if buffer == nil {
		t.Errorf("Expected NewInputBuffer to return a non-nil buffer")
	}
	if buffer.size != 60 {
		t.Errorf("Expected default buffer size to be 60, got %d", buffer.size)
	}
	if buffer.Count() != 0 {
		t.Errorf("Expected buffer count to be 0, got %d", buffer.Count())
	}
}

func TestInputBufferPushPop(t *testing.T) {
	buffer := NewInputBuffer(2)
	snapshot1 := NewInputSnapshot()
	snapshot2 := NewInputSnapshot()

	buffer.Push(snapshot1)
	buffer.Push(snapshot2)

	if buffer.Count() != 2 {
		t.Errorf("Expected buffer count to be 2, got %d", buffer.Count())
	}

	popped, ok := buffer.Pop()
	if !ok {
		t.Errorf("Expected to pop snapshot1")
	}
	if popped.Keyboard[0].Pressed != snapshot1.Keyboard[0].Pressed {
		t.Errorf("Expected popped snapshot to match original")
	}

	popped, ok = buffer.Pop()
	if !ok {
		t.Errorf("Expected to pop snapshot2")
	}
	if popped.Keyboard[0].Pressed != snapshot2.Keyboard[0].Pressed {
		t.Errorf("Expected popped snapshot to match original")
	}

	if !buffer.IsEmpty() {
		t.Errorf("Expected buffer to be empty")
	}
}

func TestInputBufferPeek(t *testing.T) {
	buffer := NewInputBuffer(2)
	snapshot1 := NewInputSnapshot()
	buffer.Push(snapshot1)

	peeked, ok := buffer.Peek()
	if !ok {
		t.Errorf("Expected to peek snapshot1")
	}
	if peeked.Keyboard[0].Pressed != snapshot1.Keyboard[0].Pressed {
		t.Errorf("Expected peeked snapshot to match original")
	}

	// Ensure that peeking does not remove the snapshot
	if buffer.Count() != 1 {
		t.Errorf("Expected buffer count to remain 1 after peek, got %d", buffer.Count())
	}
}
func TestInputBufferIsFull(t *testing.T) {
	buffer := NewInputBuffer(2)
	if buffer.IsFull() {
		t.Errorf("Expected buffer to not be full initially")
	}
	buffer.Push(NewInputSnapshot())
	buffer.Push(NewInputSnapshot())
	if !buffer.IsFull() {
		t.Errorf("Expected buffer to be full after pushing 2 snapshots")
	}
}

func TestInputBufferIsEmpty(t *testing.T) {
	buffer := NewInputBuffer(2)
	if !buffer.IsEmpty() {
		t.Errorf("Expected buffer to be empty initially")
	}
	buffer.Push(NewInputSnapshot())
	if buffer.IsEmpty() {
		t.Errorf("Expected buffer to not be empty after pushing a snapshot")
	}
	buffer.Pop()
	if !buffer.IsEmpty() {
		t.Errorf("Expected buffer to be empty after popping the snapshot")
	}
}

func TestInputBufferEviction(t *testing.T) {
	buffer := NewInputBuffer(2)
	snapshot1 := NewInputSnapshot()
	snapshot2 := NewInputSnapshot()
	snapshot3 := NewInputSnapshot()

	buffer.Push(snapshot1)
	buffer.Push(snapshot2)
	// Buffer is now full, next push should evict the oldest (snapshot1)
	buffer.Push(snapshot3)

	if buffer.Count() != 2 {
		t.Errorf("Expected buffer count to be 2 after eviction, got %d", buffer.Count())
	}

	peeked, ok := buffer.Peek()
	if !ok {
		t.Errorf("Expected the oldest snapshot to be snapshot2 after eviction")
	}
	if peeked.Keyboard[0].Pressed != snapshot2.Keyboard[0].Pressed {
		t.Errorf("Expected the oldest snapshot to match snapshot2 after eviction")
	}
}

func TestNewInputHandler(t *testing.T) {
	handler := New()
	if handler == nil {
		t.Errorf("Expected New to return a non-nil InputHandler")
	}
	if handler.buffer == nil {
		t.Errorf("Expected InputHandler to have a non-nil buffer")
	}
	if handler.buffer.size != 60 {
		t.Errorf("Expected default buffer size to be 60, got %d", handler.buffer.size)
	}
	if handler.currentSnapshot.Keyboard == nil {
		t.Errorf("Expected InputHandler to have a non-nil currentSnapshot.Keyboard")
	}
	if handler.currentSnapshot.Mouse.Buttons == nil {
		t.Errorf("Expected InputHandler to have a non-nil currentSnapshot.Mouse.Buttons")
	}
	if handler.currentSnapshot.Mouse.X != 0 || handler.currentSnapshot.Mouse.Y != 0 {
		t.Errorf("Expected InputHandler to have currentSnapshot.Mouse.X and currentSnapshot.Mouse.Y initialized to 0")
	}
}

func TestInputHandlerKeyPressed(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Pressed: true}

	if !handler.KeyPressed(0, Modifiers{}) {
		t.Errorf("Expected KeyPressed to return true for pressed key")
	}

	if handler.KeyPressed(1, Modifiers{}) {
		t.Errorf("Expected KeyPressed to return false for unpressed key")
	}
}

func TestInputHandlerKeyPressedWithModifiers(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Pressed: true}
	handler.currentSnapshot.Modifiers.Ctrl = true

	if !handler.KeyPressed(0, Modifiers{Ctrl: true}) {
		t.Errorf("Expected KeyPressed with Ctrl modifier to return true")
	}

	if handler.KeyPressed(0, Modifiers{Shift: true}) {
		t.Errorf("Expected KeyPressed with Shift modifier to return false when Shift not pressed")
	}

	if handler.KeyPressed(0, Modifiers{Ctrl: true, Shift: true}) {
		t.Errorf("Expected KeyPressed to return false when not all modifiers match")
	}
}

func TestInputHandlerKeyHeld(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Held: true}

	if !handler.KeyHeld(0, Modifiers{}) {
		t.Errorf("Expected KeyHeld to return true for held key")
	}

	if handler.KeyHeld(1, Modifiers{}) {
		t.Errorf("Expected KeyHeld to return false for unheld key")
	}
}

func TestInputHandlerKeyHeldWithModifiers(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Held: true}
	handler.currentSnapshot.Modifiers.Shift = true

	if !handler.KeyHeld(0, Modifiers{Shift: true}) {
		t.Errorf("Expected KeyHeld with Shift modifier to return true")
	}

	if handler.KeyHeld(0, Modifiers{Alt: true}) {
		t.Errorf("Expected KeyHeld with Alt modifier to return false when Alt not pressed")
	}
}

func TestInputHandlerKeyReleased(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Released: true}

	if !handler.KeyReleased(0, Modifiers{}) {
		t.Errorf("Expected KeyReleased to return true for released key")
	}

	if handler.KeyReleased(1, Modifiers{}) {
		t.Errorf("Expected KeyReleased to return false for non-released key")
	}
}

func TestInputHandlerKeyReleasedWithModifiers(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Released: true}
	handler.currentSnapshot.Modifiers.Alt = true

	if !handler.KeyReleased(0, Modifiers{Alt: true}) {
		t.Errorf("Expected KeyReleased with Alt modifier to return true")
	}

	if handler.KeyReleased(0, Modifiers{Ctrl: true}) {
		t.Errorf("Expected KeyReleased with Ctrl modifier to return false when Ctrl not pressed")
	}
}

func TestInputHandlerMousePressed(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Mouse.Buttons[0] = KeyState{Pressed: true}

	if !handler.MousePressed(0) {
		t.Errorf("Expected MousePressed to return true for pressed button")
	}

	if handler.MousePressed(1) {
		t.Errorf("Expected MousePressed to return false for unpressed button")
	}
}

func TestInputHandlerMouseHeld(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Mouse.Buttons[1] = KeyState{Held: true}

	if !handler.MouseHeld(1) {
		t.Errorf("Expected MouseHeld to return true for held button")
	}

	if handler.MouseHeld(2) {
		t.Errorf("Expected MouseHeld to return false for unheld button")
	}
}

func TestInputHandlerMouseReleased(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Mouse.Buttons[2] = KeyState{Released: true}

	if !handler.MouseReleased(2) {
		t.Errorf("Expected MouseReleased to return true for released button")
	}

	if handler.MouseReleased(1) {
		t.Errorf("Expected MouseReleased to return false for non-released button")
	}
}

func TestInputHandlerMousePosition(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Mouse.X = 100
	handler.currentSnapshot.Mouse.Y = 200

	x, y := handler.MousePosition()
	if x != 100 || y != 200 {
		t.Errorf("Expected MousePosition to return (100, 200), got (%d, %d)", x, y)
	}
}

func TestInputHandlerBuffer(t *testing.T) {
	handler := New()
	buf := handler.Buffer()

	if buf == nil {
		t.Errorf("Expected Buffer to return non-nil buffer")
	}

	if buf != handler.buffer {
		t.Errorf("Expected Buffer to return the internal buffer")
	}
}

func TestInputBufferPopEmpty(t *testing.T) {
	buffer := NewInputBuffer(2)

	_, ok := buffer.Pop()
	if ok {
		t.Errorf("Expected Pop on empty buffer to return false")
	}
}

func TestInputBufferPeekEmpty(t *testing.T) {
	buffer := NewInputBuffer(2)

	_, ok := buffer.Peek()
	if ok {
		t.Errorf("Expected Peek on empty buffer to return false")
	}
}

func TestInputBufferWrapAround(t *testing.T) {
	buffer := NewInputBuffer(3)
	s1 := NewInputSnapshot()
	s1.Mouse.X = 1
	s2 := NewInputSnapshot()
	s2.Mouse.X = 2
	s3 := NewInputSnapshot()
	s3.Mouse.X = 3
	s4 := NewInputSnapshot()
	s4.Mouse.X = 4

	buffer.Push(s1)
	buffer.Push(s2)
	buffer.Push(s3)
	buffer.Push(s4) // evicts s1

	p1, _ := buffer.Pop()
	if p1.Mouse.X != 2 {
		t.Errorf("Expected first pop to be s2 with X=2, got X=%d", p1.Mouse.X)
	}

	p2, _ := buffer.Pop()
	if p2.Mouse.X != 3 {
		t.Errorf("Expected second pop to be s3 with X=3, got X=%d", p2.Mouse.X)
	}

	p3, _ := buffer.Pop()
	if p3.Mouse.X != 4 {
		t.Errorf("Expected third pop to be s4 with X=4, got X=%d", p3.Mouse.X)
	}
}

func TestModifiersZeroValue(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Pressed: true}

	// Zero-value Modifiers{} should match when no modifiers pressed
	if !handler.KeyPressed(0, Modifiers{}) {
		t.Errorf("Expected zero-value Modifiers to work")
	}
}

func TestModifiersCombinations(t *testing.T) {
	handler := New()
	handler.currentSnapshot.Keyboard[0] = KeyState{Held: true}
	handler.currentSnapshot.Modifiers.Ctrl = true
	handler.currentSnapshot.Modifiers.Shift = true

	if !handler.KeyHeld(0, Modifiers{Ctrl: true, Shift: true}) {
		t.Errorf("Expected multiple modifiers to match together")
	}

	if handler.KeyHeld(0, Modifiers{Ctrl: true, Alt: true}) {
		t.Errorf("Expected partial modifier mismatch to fail")
	}

	if handler.KeyHeld(0, Modifiers{Alt: true}) {
		t.Errorf("Expected wrong modifier to fail")
	}
}
