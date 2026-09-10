package input

import (
	"maps"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestInputSnapshot(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		s := NewInputSnapshot()
		if s.Keyboard == nil || s.Mouse.Buttons == nil {
			t.Fatal("maps not initialized")
		}
		if s.Mouse.X != 0 || s.Mouse.Y != 0 {
			t.Error("mouse position should be (0,0)")
		}
		if s.Modifiers != (Modifiers{}) {
			t.Error("modifiers should be zero-valued")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		s := NewInputSnapshot()
		s.Keyboard[ebiten.KeyA] = KeyState{Pressed: true, Held: true, Duration: 1.0}
		s.Modifiers = Modifiers{Shift: true, Ctrl: true, Alt: true}
		s.Mouse.X = 100
		s.Mouse.Y = 200
		s.Mouse.Buttons[ebiten.MouseButtonLeft] = KeyState{Held: true, Duration: 2.0}

		s.Reset()

		if state := s.Keyboard[ebiten.KeyA]; state != (KeyState{}) {
			t.Errorf("keyboard not reset: %+v", state)
		}
		if s.Modifiers != (Modifiers{}) {
			t.Error("modifiers not reset")
		}
		if s.Mouse.X != 0 || s.Mouse.Y != 0 {
			t.Error("mouse position not reset")
		}
		if state := s.Mouse.Buttons[ebiten.MouseButtonLeft]; state != (KeyState{}) {
			t.Errorf("mouse buttons not reset: %+v", state)
		}
	})

	t.Run("Clone", func(t *testing.T) {
		s1 := NewInputSnapshot()
		s1.Keyboard[ebiten.KeyA] = KeyState{Pressed: true, Held: true}
		s1.Modifiers.Shift = true
		s1.Mouse.X = 100
		s1.Mouse.Buttons[ebiten.MouseButtonLeft] = KeyState{Held: true}

		s2 := s1.Clone()

		if !s1.Compare(&s2) {
			t.Error("clone should equal original")
		}

		s2.Keyboard[ebiten.KeyA] = KeyState{}
		s2.Modifiers.Shift = false
		s2.Mouse.X = 0
		if s1.Compare(&s2) {
			t.Error("modifying clone should not affect original")
		}
	})

	t.Run("Compare", func(t *testing.T) {
		s1 := NewInputSnapshot()
		s1.Keyboard[ebiten.KeyA] = KeyState{Pressed: true}

		if !s1.Compare(&s1) {
			t.Error("snapshot should equal itself")
		}

		s2 := s1.Clone()
		if !s1.Compare(&s2) {
			t.Error("equal snapshots should compare equal")
		}

		s2.Keyboard[ebiten.KeyA] = KeyState{}
		s2.Keyboard[ebiten.KeyB] = KeyState{Pressed: true}
		if s1.Compare(&s2) {
			t.Error("different snapshots should not compare equal")
		}

		s2 = s1.Clone()
		s2.Mouse.X = 1
		if s1.Compare(&s2) {
			t.Error("different mouse position should not compare equal")
		}
	})
}

func TestInputBuffer(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		tests := []struct {
			name     string
			size     int
			expected int
		}{
			{"DefaultSize", 0, 60},
			{"NegativeSize", -5, 60},
			{"Size10", 10, 10},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := NewInputBuffer(tt.size)
				if buf.size != tt.expected {
					t.Errorf("expected size %d, got %d", tt.expected, buf.size)
				}
				if !buf.IsEmpty() {
					t.Error("expected empty buffer")
				}
				if buf.Count() != 0 {
					t.Error("expected count 0")
				}
			})
		}
	})

	t.Run("EmptyOperations", func(t *testing.T) {
		buf := NewInputBuffer(2)

		_, ok := buf.Pop()
		if ok {
			t.Error("Pop on empty buffer should return false")
		}

		_, ok = buf.Peek()
		if ok {
			t.Error("Peek on empty buffer should return false")
		}
	})

	t.Run("PushPopPeek", func(t *testing.T) {
		buf := NewInputBuffer(3)

		s1 := NewInputSnapshot()
		s1.Mouse.X = 1
		s2 := NewInputSnapshot()
		s2.Mouse.X = 2

		buf.Push(s1)
		buf.Push(s2)

		if buf.Count() != 2 {
			t.Fatalf("expected count 2, got %d", buf.Count())
		}

		peeked, ok := buf.Peek()
		if !ok || peeked.Mouse.X != 1 {
			t.Errorf("Peek: expected X=1, got %d", peeked.Mouse.X)
		}
		if buf.Count() != 2 {
			t.Error("Peek should not change count")
		}

		p1, ok := buf.Pop()
		if !ok || p1.Mouse.X != 1 {
			t.Errorf("Pop: expected X=1, got %d", p1.Mouse.X)
		}
		if buf.Count() != 1 {
			t.Errorf("expected count 1 after pop, got %d", buf.Count())
		}

		p2, ok := buf.Pop()
		if !ok || p2.Mouse.X != 2 {
			t.Errorf("Pop: expected X=2, got %d", p2.Mouse.X)
		}

		if !buf.IsEmpty() {
			t.Error("expected buffer empty after all pops")
		}
	})

	t.Run("Eviction", func(t *testing.T) {
		buf := NewInputBuffer(2)
		s1 := NewInputSnapshot()
		s1.Mouse.X = 1
		s2 := NewInputSnapshot()
		s2.Mouse.X = 2
		s3 := NewInputSnapshot()
		s3.Mouse.X = 3

		buf.Push(s1)
		buf.Push(s2)
		buf.Push(s3) // evicts s1

		if buf.Count() != 2 {
			t.Fatalf("expected count 2 after eviction, got %d", buf.Count())
		}

		peeked, ok := buf.Peek()
		if !ok || peeked.Mouse.X != 2 {
			t.Errorf("expected oldest to be s2 with X=2, got %d", peeked.Mouse.X)
		}
	})

	t.Run("WrapAround", func(t *testing.T) {
		buf := NewInputBuffer(3)
		for i := 0; i < 5; i++ {
			s := NewInputSnapshot()
			s.Mouse.X = i
			buf.Push(s)
		}

		if buf.Count() != 3 {
			t.Fatalf("expected count 3, got %d", buf.Count())
		}

		expected := []int{2, 3, 4}
		for i, exp := range expected {
			p, ok := buf.Pop()
			if !ok {
				t.Fatalf("Pop %d failed", i)
			}
			if p.Mouse.X != exp {
				t.Errorf("Pop %d: expected X=%d, got %d", i, exp, p.Mouse.X)
			}
		}
	})

	t.Run("IsFull", func(t *testing.T) {
		buf := NewInputBuffer(2)
		if buf.IsFull() {
			t.Error("new buffer should not be full")
		}

		buf.Push(NewInputSnapshot())
		if buf.IsFull() {
			t.Error("buffer with 1/2 should not be full")
		}

		buf.Push(NewInputSnapshot())
		if !buf.IsFull() {
			t.Error("buffer with 2/2 should be full")
		}
	})
}

func TestInputHandler(t *testing.T) {
	newHandler := func() *InputHandler {
		h := New()
		h.lastFrameTime = time.Now().Add(-time.Second)
		return h
	}

	t.Run("New", func(t *testing.T) {
		h := New()
		if h == nil {
			t.Fatal("expected non-nil handler")
		}
		if h.buffer == nil || h.buffer.size != 60 {
			t.Errorf("expected buffer with size 60, got %d", h.buffer.size)
		}
		if h.currentSnapshot.Keyboard == nil || h.currentSnapshot.Mouse.Buttons == nil {
			t.Error("expected initialized snapshot maps")
		}
		if h.lastFrameTime != (time.Time{}) {
			t.Error("expected zero lastFrameTime")
		}
	})

	t.Run("Buffer", func(t *testing.T) {
		h := New()
		buf := h.Buffer()
		if buf == nil {
			t.Fatal("expected non-nil buffer")
		}
		if buf != h.buffer {
			t.Error("expected same buffer reference")
		}
	})

	t.Run("Snapshot", func(t *testing.T) {
		h := newHandler()
		initialCount := h.buffer.Count()
		initialTime := h.lastFrameTime

		h.Snapshot()

		if h.buffer.Count() <= initialCount {
			t.Error("expected Snapshot to push to buffer")
		}
		if h.lastFrameTime.Equal(initialTime) {
			t.Error("expected lastFrameTime to be updated")
		}

		peeked, ok := h.buffer.Peek()
		if !ok {
			t.Fatal("expected snapshot in buffer after Snapshot()")
		}
		if peeked.Keyboard == nil || peeked.Mouse.Buttons == nil {
			t.Error("expected pushed snapshot to have initialized maps")
		}
	})

	t.Run("KeyStates", func(t *testing.T) {
		tests := []struct {
			name         string
			state        KeyState
			snapshotMods Modifiers
			queryMods    Modifiers
			key          ebiten.Key
			expectP      bool
			expectH      bool
			expectR      bool
		}{
			{"NotPressed", KeyState{}, Modifiers{}, Modifiers{}, ebiten.KeyA, false, false, false},
			{"Pressed", KeyState{Pressed: true, Held: true}, Modifiers{}, Modifiers{}, ebiten.KeyA, true, true, false},
			{"Held", KeyState{Held: true}, Modifiers{}, Modifiers{}, ebiten.KeyA, false, true, false},
			{"Released", KeyState{Released: true}, Modifiers{}, Modifiers{}, ebiten.KeyA, false, false, true},
			{"PressedWithCtrl/Match", KeyState{Pressed: true, Held: true}, Modifiers{Ctrl: true}, Modifiers{Ctrl: true}, ebiten.KeyA, true, true, false},
			{"PressedWithCtrl/NoMatch", KeyState{Pressed: true, Held: true}, Modifiers{}, Modifiers{Ctrl: true}, ebiten.KeyA, false, false, false},
			{"PressedWithMultiMods/Match", KeyState{Pressed: true, Held: true}, Modifiers{Ctrl: true, Shift: true}, Modifiers{Ctrl: true, Shift: true}, ebiten.KeyA, true, true, false},
			{"PressedWithMultiMods/ExtraInSnapshot", KeyState{Pressed: true, Held: true}, Modifiers{Ctrl: true, Shift: true}, Modifiers{Ctrl: true}, ebiten.KeyA, true, true, false},
			{"PressedWithMultiMods/MissingInSnapshot", KeyState{Pressed: true, Held: true}, Modifiers{Ctrl: true}, Modifiers{Ctrl: true, Shift: true}, ebiten.KeyA, false, false, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := newHandler()
				h.currentSnapshot.Keyboard[tt.key] = tt.state
				h.currentSnapshot.Modifiers = tt.snapshotMods

				if got := h.KeyPressed(tt.key, tt.queryMods); got != tt.expectP {
					t.Errorf("KeyPressed: expected %v, got %v", tt.expectP, got)
				}
				if got := h.KeyHeld(tt.key, tt.queryMods); got != tt.expectH {
					t.Errorf("KeyHeld: expected %v, got %v", tt.expectH, got)
				}
				if got := h.KeyReleased(tt.key, tt.queryMods); got != tt.expectR {
					t.Errorf("KeyReleased: expected %v, got %v", tt.expectR, got)
				}
			})
		}
	})

	t.Run("MouseStates", func(t *testing.T) {
		tests := []struct {
			name        string
			buttonState map[ebiten.MouseButton]KeyState
			button      ebiten.MouseButton
			expectP     bool
			expectH     bool
			expectR     bool
		}{
			{"NotPressed", nil, ebiten.MouseButtonLeft, false, false, false},
			{"Pressed", map[ebiten.MouseButton]KeyState{ebiten.MouseButtonLeft: {Pressed: true, Held: true}}, ebiten.MouseButtonLeft, true, true, false},
			{"Held", map[ebiten.MouseButton]KeyState{ebiten.MouseButtonLeft: {Held: true}}, ebiten.MouseButtonLeft, false, true, false},
			{"Released", map[ebiten.MouseButton]KeyState{ebiten.MouseButtonLeft: {Released: true}}, ebiten.MouseButtonLeft, false, false, true},
			{"RightButton", map[ebiten.MouseButton]KeyState{ebiten.MouseButtonRight: {Pressed: true, Held: true}}, ebiten.MouseButtonRight, true, true, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := newHandler()
				maps.Copy(h.currentSnapshot.Mouse.Buttons, tt.buttonState)

				if got := h.MousePressed(tt.button); got != tt.expectP {
					t.Errorf("MousePressed: expected %v, got %v", tt.expectP, got)
				}
				if got := h.MouseHeld(tt.button); got != tt.expectH {
					t.Errorf("MouseHeld: expected %v, got %v", tt.expectH, got)
				}
				if got := h.MouseReleased(tt.button); got != tt.expectR {
					t.Errorf("MouseReleased: expected %v, got %v", tt.expectR, got)
				}
			})
		}
	})

	t.Run("MousePosition", func(t *testing.T) {
		h := newHandler()
		h.currentSnapshot.Mouse.X = 100
		h.currentSnapshot.Mouse.Y = 200

		x, y := h.MousePosition()
		if x != 100 || y != 200 {
			t.Errorf("expected (100,200), got (%d,%d)", x, y)
		}
	})
}
