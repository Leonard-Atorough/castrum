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

	// Tick 1: the pair exists by construction; the capture snapshots it.
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

func TestAddComponentTransformCompletesPair(t *testing.T) {
	w := NewWorld()
	entity, err := w.NewEntity(Camera{Zoom: 1})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	// A mid-game Transform attach also attaches the pair's prev, at
	// the attach position: the entity interpolates from where it
	// gained its Transform, on its first rendered frame.
	pos := geom.Vector2{X: 7, Y: 9}
	if err := entity.AddComponent(w, Transform{Position: pos, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
		t.Fatalf("AddComponent(Transform): %v", err)
	}
	prev, ok := entity.Component[PrevTransform](w)
	if !ok {
		t.Fatal("AddComponent(Transform) must attach the pair's PrevTransform")
	}
	if prev.Position != pos {
		t.Fatalf("prev = %v, want the attach position %v", prev.Position, pos)
	}

	// The capture then snapshots like any other entity.
	runCapture(t, NewPrevTransformCapture(), w)
	if err := entity.SetComponent(w, Transform{Position: geom.Vector2{X: 20, Y: 0}, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
		t.Fatalf("move: %v", err)
	}
	runCapture(t, NewPrevTransformCapture(), w)
	prev, _ = entity.Component[PrevTransform](w)
	if prev.Position.X != 20 {
		t.Fatalf("prev after second capture = %v, want the tick-start position", prev.Position)
	}
}

func TestAddComponentTransformKeepsExistingPrev(t *testing.T) {
	w := NewWorld()
	// An explicit prev is authoritative: a later Transform attach
	// must not overwrite it.
	entity, err := w.NewEntity(PrevTransform{Position: geom.Vector2{X: 1, Y: 2}})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := entity.AddComponent(w, Transform{Position: geom.Vector2{X: 8, Y: 8}, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
		t.Fatalf("AddComponent(Transform): %v", err)
	}
	prev, _ := entity.Component[PrevTransform](w)
	if prev.Position.X != 1 || prev.Position.Y != 2 {
		t.Fatalf("prev = %v, want the explicit (1, 2) preserved", prev.Position)
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
