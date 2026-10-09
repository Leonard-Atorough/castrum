package core

import (
	"fmt"
	"testing"
)

type position struct{ x, y float64 }
type velocity struct{ dx, dy float64 }

// rating is a validatable test component: values outside [0, 10] fail.
type rating struct{ n int }

func (r rating) Validate() error {
	if r.n < 0 || r.n > 10 {
		return fmt.Errorf("rating must be between 0 and 10")
	}
	return nil
}

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
	e, _ := w.NewEntity(position{x: 1, y: 2}, velocity{dx: 3, dy: 4})

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
	e, _ := w.NewEntity(position{})
	if !e.HasComponent[position](w) {
		t.Error("HasComponent[position] = false, want true")
	}
	if e.HasComponent[velocity](w) {
		t.Error("HasComponent[velocity] = true, want false")
	}
}

func TestSetComponent(t *testing.T) {
	w := NewWorld()
	e, _ := w.NewEntity(position{x: 1})

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

func TestEntityUpdateMutates(t *testing.T) {
	w := NewWorld()
	e, err := w.NewEntity(position{x: 1, y: 2})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := e.Update(w, func(p *position) { p.x += 5 }); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, ok := e.Component[position](w)
	if !ok || got.x != 6 {
		t.Fatalf("position = %v, ok %v, want x 6", got, ok)
	}
}

func TestEntityUpdateMissingComponent(t *testing.T) {
	w := NewWorld()
	e, err := w.NewEntity(position{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := e.Update(w, func(v *velocity) { v.dx = 1 }); err == nil {
		t.Fatal("Update on a missing component should error")
	}
}

func TestEntityUpdateValidatesOnWrite(t *testing.T) {
	w := NewWorld()
	e, err := w.NewEntity(rating{n: 5})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	err = e.Update(w, func(r *rating) { r.n = 99 })
	if err == nil {
		t.Fatal("Update writing an invalid value should error")
	}
}
