package core

import "testing"

type position struct{ x, y float64 }
type velocity struct{ dx, dy float64 }

func TestNewEntityAliveWithID(t *testing.T) {
	e := NewEntity(42)
	if e.ID() != 42 {
		t.Errorf("ID() = %d, want 42", e.ID())
	}
	if !e.IsAlive() {
		t.Error("new entity should be alive")
	}
}

func TestKillMarksEntityDead(t *testing.T) {
	e := NewEntity(0)
	e.Kill()
	if e.IsAlive() {
		t.Error("killed entity should not be alive")
	}
}

func TestComponentRoundTrip(t *testing.T) {
	w := NewWorld()
	e := w.NewEntity(position{x: 1, y: 2}, velocity{dx: 3, dy: 4})

	p, ok := e.Component[position](w)
	if !ok || p != (position{x: 1, y: 2}) {
		t.Errorf("Component[position] = %v, %v; want {1 2}, true", p, ok)
	}
	if v, ok := e.Component[velocity](w); !ok || v != (velocity{dx: 3, dy: 4}) {
		t.Errorf("Component[velocity] = %v, %v; want {3 4}, true", v, ok)
	}
	if _, ok := e.Component[struct{ tag int }](w); ok {
		t.Error("Component for an unregistered type should report false")
	}
}

func TestHasComponent(t *testing.T) {
	w := NewWorld()
	e := w.NewEntity(position{})
	if !e.HasComponent[position](w) {
		t.Error("HasComponent[position] = false, want true")
	}
	if e.HasComponent[velocity](w) {
		t.Error("HasComponent[velocity] = true, want false")
	}
}

func TestSetComponent(t *testing.T) {
	w := NewWorld()
	e := w.NewEntity(position{x: 1})

	if err := e.SetComponent(w, position{x: 9}); err != nil {
		t.Fatalf("SetComponent: %v", err)
	}
	if p, _ := e.Component[position](w); p.x != 9 {
		t.Errorf("after SetComponent, x = %v, want 9", p.x)
	}
	if err := e.SetComponent(w, velocity{dx: 1}); err == nil {
		t.Error("SetComponent for a missing component should error")
	}
}
