package ebitrun

import (
	"testing"

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
	p.poll()
	s := p.snapshotPtr()
	if s.KeyHeld(input.KeyA) || s.MouseHeld(input.MouseButtonLeft) ||
		s.PadConnected(0) || s.PadHeld(0, input.PadSouth) ||
		s.PadAxis(0, input.PadLeftStickX) != 0 {
		t.Error("headless poll should report no input")
	}
}
