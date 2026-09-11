package input

import "testing"

func TestInputSnapshot(t *testing.T) {
	snapshot := NewInputSnapshot()
	snapshot.Keyboard["space"] = KeyState{Pressed: true, Held: true}
	snapshot.Mouse.Buttons["left"] = KeyState{Held: true}
	snapshot.Mouse.X = 100

	clone := snapshot.Clone()
	if !snapshot.Compare(&clone) {
		t.Fatal("clone should equal original")
	}

	clone.Keyboard["space"] = KeyState{}
	if snapshot.Compare(&clone) {
		t.Fatal("changing clone should not change original")
	}

	snapshot.Reset()
	if snapshot.Keyboard["space"] != (KeyState{}) || snapshot.Mouse.Buttons["left"] != (KeyState{}) {
		t.Fatal("reset should clear input states")
	}
}

func TestInputBuffer(t *testing.T) {
	buffer := NewInputBuffer(2)
	first := NewInputSnapshot()
	first.Mouse.X = 1
	second := NewInputSnapshot()
	second.Mouse.X = 2
	third := NewInputSnapshot()
	third.Mouse.X = 3

	buffer.Push(first)
	buffer.Push(second)
	buffer.Push(third)

	got, ok := buffer.Pop()
	if !ok || got.Mouse.X != 2 {
		t.Fatalf("expected oldest retained snapshot to be second, got %+v", got)
	}
	if buffer.Count() != 1 {
		t.Fatalf("expected one snapshot after pop, got %d", buffer.Count())
	}
}
