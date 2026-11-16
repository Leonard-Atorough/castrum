package ebitrun

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestKeyTableComplete(t *testing.T) {
	for k := range input.NumKeys {
		if k == input.KeyNone {
			continue
		}
		if _, ok := keyTable[k]; !ok {
			t.Errorf("key %v has no ebiten mapping", k)
		}
	}
}

func TestMouseButtonTableComplete(t *testing.T) {
	for b := range input.NumMouseButtons {
		if _, ok := mouseButtonTable[b]; !ok {
			t.Errorf("mouse button %v has no ebiten mapping", b)
		}
	}
}

func TestPadTablesComplete(t *testing.T) {
	for b := range input.NumPadButtons {
		if _, ok := padButtonTable[b]; !ok {
			t.Errorf("pad button %v has no ebiten mapping", b)
		}
	}
	for a := range input.NumPadAxes {
		_, sticks := padAxisTable[a]
		_, triggers := triggerTable[a]
		if sticks == triggers {
			t.Errorf("pad axis %v: covered by both or neither of the axis tables", a)
		}
	}
}

func TestPollerSmoke(t *testing.T) {
	p := &poller{}
	p.snapshot.Mouse.Wheel = geom.Vector2{X: 0.25, Y: -1.5}
	p.poll()
	s := p.snapshotPtr()
	if s.KeyHeld(input.KeyA) || s.MouseHeld(input.MouseButtonLeft) ||
		s.PadConnected(0) || s.PadHeld(0, input.PadSouth) ||
		s.PadAxis(0, input.PadLeftStickX) != 0 || s.Wheel() != (geom.Vector2{}) {
		t.Error("headless poll should report no input")
	}
}

type fakeGamepads struct {
	ids     []ebiten.GamepadID
	buttons map[ebiten.GamepadID]map[input.PadButton]input.State
	axes    map[ebiten.GamepadID]map[input.PadAxis]float64
}

func (f *fakeGamepads) gamepads() []ebiten.GamepadID {
	return f.ids
}

func (f *fakeGamepads) button(id ebiten.GamepadID, button ebiten.StandardGamepadButton) input.State {
	for padButton, mapped := range padButtonTable {
		if mapped == button {
			return f.buttons[id][padButton]
		}
	}
	return input.State{}
}

func (f *fakeGamepads) axis(id ebiten.GamepadID, axis input.PadAxis) float64 {
	return f.axes[id][axis]
}

func newFakeGamepads(ids ...ebiten.GamepadID) *fakeGamepads {
	return &fakeGamepads{
		ids:     append([]ebiten.GamepadID(nil), ids...),
		buttons: make(map[ebiten.GamepadID]map[input.PadButton]input.State),
		axes:    make(map[ebiten.GamepadID]map[input.PadAxis]float64),
	}
}

func TestPollerHoldsSlotsForConnectionLifetime(t *testing.T) {
	pads := newFakeGamepads(10, 20)
	pads.buttons[10] = map[input.PadButton]input.State{input.PadSouth: {Held: true}}
	p := &poller{pads: pads}
	p.poll()

	if !p.snapshot.PadConnected(0) || !p.snapshot.PadConnected(1) {
		t.Fatalf("slots 0/1 connected = %v/%v, want both devices present", p.snapshot.PadConnected(0), p.snapshot.PadConnected(1))
	}
	if !p.snapshot.PadHeld(0, input.PadSouth) {
		t.Fatal("held button should be reported while connected")
	}

	pads.ids = []ebiten.GamepadID{20}
	p.poll()

	if p.snapshot.PadConnected(0) {
		t.Fatal("the disconnected device must leave its slot")
	}
	if !p.snapshot.PadConnected(1) {
		t.Fatal("the surviving device must keep its slot")
	}
	if !p.snapshot.PadReleased(0, input.PadSouth) {
		t.Fatal("a held button must release on disconnect")
	}
	if p.snapshot.PadHeld(0, input.PadSouth) || p.snapshot.PadPressed(0, input.PadSouth) {
		t.Fatal("the transition snapshot must not report the disconnected button held or pressed")
	}
	if p.snapshot.PadAxis(0, input.PadLeftStickX) != 0 {
		t.Fatal("the transition snapshot must zero the axes")
	}

	p.poll()

	if p.snapshot.PadConnected(0) || p.snapshot.PadReleased(0, input.PadSouth) {
		t.Fatal("the disconnect transition must last exactly one frame")
	}
	if !p.snapshot.PadConnected(1) {
		t.Fatal("the survivor must still hold its slot")
	}
}

func TestPollerReusesFreedSlotDeterministically(t *testing.T) {
	pads := newFakeGamepads(10, 20)
	p := &poller{pads: pads}
	p.poll()

	pads.ids = []ebiten.GamepadID{20}
	p.poll()

	pads.ids = []ebiten.GamepadID{20, 30}
	p.poll()

	if !p.snapshot.PadConnected(0) {
		t.Fatal("a new device should claim the slot freed by the disconnect")
	}
	if !p.snapshot.PadConnected(1) {
		t.Fatal("reassignment must not move the surviving device")
	}
}

func TestPollerIgnoresDevicesBeyondCapacity(t *testing.T) {
	pads := newFakeGamepads(1, 2, 3, 4, 5)
	p := &poller{pads: pads}
	p.poll()

	for slot := range input.MaxPads {
		if !p.snapshot.PadConnected(slot) {
			t.Fatalf("slot %d = disconnected, want the first four devices admitted", slot)
		}
	}

	pads.ids = []ebiten.GamepadID{1, 2, 3, 5}
	p.poll()

	if p.snapshot.PadConnected(3) {
		t.Fatal("the fifth device must wait for the departing device's transition frame")
	}
	p.poll()

	if !p.snapshot.PadConnected(3) {
		t.Fatal("the fifth device should claim the freed slot after the transition frame")
	}
	if !p.snapshot.PadConnected(0) || !p.snapshot.PadConnected(1) || !p.snapshot.PadConnected(2) {
		t.Fatal("capacity churn must not move the survivors")
	}
}
