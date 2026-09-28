package input

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

// The nil contract: gameplay queries ctx.Input without guards, and
// headless tests have no runner publishing input - every query on a
// nil snapshot must read zero instead of panicking.
func TestSnapshotNilReadsZero(t *testing.T) {
	var s *Snapshot
	for _, q := range []struct {
		name string
		read func() bool
	}{
		{"KeyPressed", func() bool { return s.KeyPressed(KeyA) }},
		{"KeyHeld", func() bool { return s.KeyHeld(KeyA) }},
		{"KeyReleased", func() bool { return s.KeyReleased(KeyA) }},
		{"MousePressed", func() bool { return s.MousePressed(MouseButtonLeft) }},
		{"MouseHeld", func() bool { return s.MouseHeld(MouseButtonLeft) }},
		{"MouseReleased", func() bool { return s.MouseReleased(MouseButtonLeft) }},
		{"PadPressed", func() bool { return s.PadPressed(0, PadSouth) }},
		{"PadHeld", func() bool { return s.PadHeld(0, PadSouth) }},
		{"PadReleased", func() bool { return s.PadReleased(0, PadSouth) }},
		{"PadConnected", func() bool { return s.PadConnected(0) }},
		{"PadAxis", func() bool { return s.PadAxis(0, PadLeftStickX) != 0 }},
		{"Cursor", func() bool { return s.Cursor() != (geom.Vector2{}) }},
		{"Wheel", func() bool { return s.Wheel() != (geom.Vector2{}) }},
	} {
		if q.read() {
			t.Errorf("nil snapshot: %s should read zero", q.name)
		}
	}
}

// The bounds contract: the poller's device tables are the component
// that could hand a bad index, and the read path must return zero
// rather than panic.
func TestSnapshotBoundsReadZero(t *testing.T) {
	s := &Snapshot{}
	for _, q := range []struct {
		name string
		read func() bool
	}{
		{"negative key", func() bool { return s.KeyPressed(Key(-1)) }},
		{"key at NumKeys", func() bool { return s.KeyPressed(Key(NumKeys)) }},
		{"negative mouse button", func() bool { return s.MousePressed(MouseButton(-1)) }},
		{"mouse button at NumMouseButtons", func() bool { return s.MousePressed(MouseButton(NumMouseButtons)) }},
		{"negative pad index", func() bool { return s.PadPressed(-1, PadSouth) }},
		{"pad index at MaxPads", func() bool { return s.PadPressed(MaxPads, PadSouth) }},
		{"negative pad button", func() bool { return s.PadPressed(0, PadButton(-1)) }},
		{"pad button at NumPadButtons", func() bool { return s.PadPressed(0, PadButton(NumPadButtons)) }},
		{"negative pad axis", func() bool { return s.PadAxis(0, PadAxis(-1)) != 0 }},
		{"pad axis at NumPadAxes", func() bool { return s.PadAxis(0, PadAxis(NumPadAxes)) != 0 }},
	} {
		if q.read() {
			t.Errorf("out-of-range %s should read zero", q.name)
		}
	}
}

// The readback: every query must return what was set, indexing the
// right slot - an indexing slip (PadPressed(1, ...) reading Pads[0])
// compiles fine and only this test sees it. The zero value reading
// all-off rides along at the end.
func TestSnapshotReadback(t *testing.T) {
	s := Snapshot{}
	s.Keys[KeyW] = State{Pressed: true, Held: true}
	s.Keys[KeySpace] = State{Released: true}
	s.Mouse = MouseSnapshot{
		Cursor: geom.Vector2{X: 320, Y: 240},
		Wheel:  geom.Vector2{X: 0, Y: -2},
	}
	s.Mouse.Buttons[MouseButtonLeft] = State{Held: true}
	s.Pads[1] = PadSnapshot{Connected: true}
	s.Pads[1].Buttons[PadSouth] = State{Held: true}
	s.Pads[1].Axes[PadLeftStickX] = -0.75

	p := &s
	if !p.KeyPressed(KeyW) || !p.KeyHeld(KeyW) || p.KeyReleased(KeyW) {
		t.Errorf("KeyW = pressed %v, held %v, released %v, want true/true/false",
			p.KeyPressed(KeyW), p.KeyHeld(KeyW), p.KeyReleased(KeyW))
	}
	if !p.KeyReleased(KeySpace) || p.KeyHeld(KeySpace) {
		t.Errorf("KeySpace released/held = %v/%v, want true/false",
			p.KeyReleased(KeySpace), p.KeyHeld(KeySpace))
	}
	if !p.MouseHeld(MouseButtonLeft) || p.MousePressed(MouseButtonLeft) {
		t.Errorf("left mouse held/pressed = %v/%v, want true/false",
			p.MouseHeld(MouseButtonLeft), p.MousePressed(MouseButtonLeft))
	}
	if p.Cursor() != (geom.Vector2{X: 320, Y: 240}) {
		t.Errorf("cursor = %v, want (320, 240)", p.Cursor())
	}
	if p.Wheel() != (geom.Vector2{X: 0, Y: -2}) {
		t.Errorf("wheel = %v, want (0, -2)", p.Wheel())
	}
	if p.PadConnected(0) || !p.PadConnected(1) {
		t.Errorf("pad 0/1 connected = %v/%v, want false/true",
			p.PadConnected(0), p.PadConnected(1))
	}
	if !p.PadHeld(1, PadSouth) || p.PadPressed(1, PadSouth) {
		t.Errorf("pad 1 south held/pressed = %v/%v, want true/false",
			p.PadHeld(1, PadSouth), p.PadPressed(1, PadSouth))
	}
	if p.PadAxis(1, PadLeftStickX) != -0.75 {
		t.Errorf("pad 1 left stick X = %v, want -0.75", p.PadAxis(1, PadLeftStickX))
	}

	// The zero value reads as everything off and every pad
	// disconnected.
	empty := &Snapshot{}
	if empty.KeyHeld(KeyA) || empty.MouseHeld(MouseButtonLeft) ||
		empty.PadConnected(0) || empty.PadAxis(0, PadLeftStickX) != 0 ||
		empty.Cursor() != (geom.Vector2{}) {
		t.Error("zero Snapshot should read as all input off")
	}
}
