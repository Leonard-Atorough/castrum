package core

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/geom"
)

func runCapture(t *testing.T, capture System, w *World) {
	t.Helper()
	if err := capture.Update(&Context{World: w}); err != nil {
		t.Fatalf("capture: %v", err)
	}
}

func TestPrevCaptureSnapshotsTickStartState(t *testing.T) {
	w := NewWorld()
	entity, err := w.NewEntity(Transform{Position: geom.Vector2{X: 10, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	capture := NewPrevTransformCapture()

	// Tick 1: the newborn is materialized with its spawn position.
	runCapture(t, capture, w)

	// Gameplay moves the entity, then tick 2's capture runs FIRST:
	// prev must hold the tick-start position (10), curr the moved one.
	if err := entity.SetComponent(w, Transform{Position: geom.Vector2{X: 20, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
		t.Fatalf("move: %v", err)
	}
	runCapture(t, capture, w)

	prev, ok := entity.Component[PrevTransform](w)
	if !ok {
		t.Fatal("entity missing PrevTransform after capture")
	}
	if prev.Position.X != 20 {
		t.Fatalf("prev position = %v, want 20 (the tick-start state)", prev.Position)
	}

	// The next gameplay move advances curr past prev: the render pair.
	if err := entity.SetComponent(w, Transform{Position: geom.Vector2{X: 30, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
		t.Fatalf("move: %v", err)
	}
	curr, _ := entity.Component[Transform](w)
	prev, _ = entity.Component[PrevTransform](w)
	if prev.Position.X != 20 || curr.Position.X != 30 {
		t.Fatalf("render pair = (%v, %v), want (20, 30)", prev.Position, curr.Position)
	}
}

func TestPrevCaptureMaterializesNewbornsOnce(t *testing.T) {
	w := NewWorld()
	entity, err := w.NewEntity(Transform{Position: geom.Vector2{X: 5, Y: 5}, Scale: geom.Vector2{X: 1, Y: 1}})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	capture := NewPrevTransformCapture()

	runCapture(t, capture, w)
	prev, ok := entity.Component[PrevTransform](w)
	if !ok || prev.Position.X != 5 {
		t.Fatalf("newborn prev = %+v ok=%v, want spawn position", prev, ok)
	}

	// The second run takes the update path, not a duplicate add.
	runCapture(t, capture, w)
	prev, _ = entity.Component[PrevTransform](w)
	if prev.Position.X != 5 {
		t.Fatalf("second capture changed prev to %v, want 5", prev.Position)
	}
}

func TestPrevCaptureSkipsEntitiesWithoutTransform(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(Camera{Zoom: 1}); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	capture := NewPrevTransformCapture()
	runCapture(t, capture, w) // must not error or add PrevTransform to it
}

func TestEntrySetWritesThroughPrefetchedColumn(t *testing.T) {
	w := NewWorld()
	entity, err := w.NewEntity(
		Transform{Position: geom.Vector2{X: 1, Y: 1}, Scale: geom.Vector2{X: 1, Y: 1}},
	)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := entity.AddComponent(w, PrevTransform{Position: geom.Vector2{X: 0, Y: 0}}); err != nil {
		t.Fatalf("add prev: %v", err)
	}

	q := NewQuery(w).With(Transform{}, PrevTransform{})
	for e := range q.Execute() {
		e.SetComponent(PrevTransform{Position: geom.Vector2{X: 9, Y: 9}})
	}

	prev, _ := entity.Component[PrevTransform](w)
	if prev.Position.X != 9 {
		t.Fatalf("Entry.Set did not reach storage: prev = %+v", prev)
	}
}

func TestEntrySetUnprefetchedTypePanics(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(
		Transform{Position: geom.Vector2{X: 1, Y: 1}, Scale: geom.Vector2{X: 1, Y: 1}},
		PrevTransform{Position: geom.Vector2{X: 1, Y: 1}},
	); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	q := NewQuery(w).With(Transform{})
	defer func() {
		if recover() == nil {
			t.Fatal("Set for a type not in With should panic")
		}
	}()
	for e := range q.Execute() {
		e.SetComponent(PrevTransform{})
	}
}
