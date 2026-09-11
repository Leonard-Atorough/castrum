package ecs

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
)

func TestQuery_WithFilter(t *testing.T) {
	w := NewWorld()

	e1, _ := w.CreateWithComponents("e1", TestPosition{X: 1})
	e2, _ := w.CreateWithComponents("e2", TestPosition{X: 2})

	got := w.NewQuery().
		WithRequiredComponents(TestPosition{}).
		WithFilter(func(r ResultEntry) bool {
			pos, err := r.Get[TestPosition]()
			return err == nil && pos.X > 1
		}).
		EntityIDs()

	if len(got) != 1 || got[0] != e2.ID {
		t.Errorf("expected only e2 (X>1), got %v (e1=%d, e2=%d)", got, e1.ID, e2.ID)
	}
}

func TestQuery_WithFilter_Composes(t *testing.T) {
	w := NewWorld()

	_, _ = w.CreateWithComponents("e1", TestPosition{X: 1}, TestHealth{Value: 10})
	e2, _ := w.CreateWithComponents("e2", TestPosition{X: 5}, TestHealth{Value: 20})
	_, _ = w.CreateWithComponents("e3", TestPosition{X: 5}, TestHealth{Value: 5})

	got := w.NewQuery().
		WithRequiredComponents(TestPosition{}, TestHealth{}).
		WithFilter(func(r ResultEntry) bool {
			pos, _ := r.Get[TestPosition]()
			return pos.X >= 5
		}).
		WithFilter(func(r ResultEntry) bool {
			hp, _ := r.Get[TestHealth]()
			return hp.Value >= 10
		}).
		EntityIDs()

	if len(got) != 1 || got[0] != e2.ID {
		t.Errorf("expected only e2 to satisfy both filters, got %v", got)
	}
}

func TestQuery_InScene(t *testing.T) {
	w := NewWorld()

	a1, _ := w.CreateWithComponents("a1", TestPosition{}, components.SceneTag{SceneID: "a"})
	_, _ = w.CreateWithComponents("a2", TestPosition{}, components.SceneTag{SceneID: "b"})
	_, _ = w.CreateWithComponents("a3", TestPosition{})

	got := w.NewQuery().
		WithRequiredComponents(TestPosition{}).
		InScene("a").
		EntityIDs()

	if len(got) != 1 || got[0] != a1.ID {
		t.Errorf("expected only entities tagged scene 'a', got %v", got)
	}
}

func TestQuery_InScene_NoMatches(t *testing.T) {
	w := NewWorld()

	_, _ = w.CreateWithComponents("a1", components.SceneTag{SceneID: "a"})

	got := w.NewQuery().InScene("nonexistent").EntityIDs()
	if len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestQuery_InScene_ComposesWithOtherRequiredComponents(t *testing.T) {
	w := NewWorld()

	// Same scene, but only one has the extra required component.
	tagged, _ := w.CreateWithComponents("tagged", TestPosition{}, TestVelocity{}, components.SceneTag{SceneID: "level"})
	_, _ = w.CreateWithComponents("untagged-velocity", TestPosition{}, components.SceneTag{SceneID: "level"})

	got := w.NewQuery().
		WithRequiredComponents(TestPosition{}, TestVelocity{}).
		InScene("level").
		EntityIDs()

	if len(got) != 1 || got[0] != tagged.ID {
		t.Errorf("expected only the entity with both required components and scene tag, got %v", got)
	}
}

func TestQuery_ResultMethods(t *testing.T) {
	w := NewWorld()

	firstEntity, _ := w.CreateWithComponents("first", TestPosition{X: 1})
	secondEntity, _ := w.CreateWithComponents("second", TestPosition{X: 2})
	_, _ = w.CreateWithComponents("excluded", TestPosition{X: 0})

	query := w.NewQuery().
		WithRequiredComponents(TestPosition{}).
		WithFilter(func(r ResultEntry) bool {
			position, _ := r.Get[TestPosition]()
			return position.X > 0
		})

	all := query.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d results, want 2", len(all))
	}
	if all[0].EntityID != firstEntity.ID || all[1].EntityID != secondEntity.ID {
		t.Errorf("All() returned entity IDs %v, want [%d %d]", []EntityID{all[0].EntityID, all[1].EntityID}, firstEntity.ID, secondEntity.ID)
	}

	first, ok := query.First()
	if !ok {
		t.Error("First() reported no result for a non-empty query")
	}
	if first.EntityID != firstEntity.ID {
		t.Errorf("First() returned entity ID %d, want %d", first.EntityID, firstEntity.ID)
	}
	if !query.Any() {
		t.Error("Any() returned false for a non-empty query")
	}
	if got := query.Count(); got != 2 {
		t.Errorf("Count() returned %d, want 2", got)
	}
}

func TestQuery_ResultMethods_Empty(t *testing.T) {
	query := NewWorld().NewQuery().WithRequiredComponents(TestPosition{})

	if got := query.All(); len(got) != 0 {
		t.Errorf("All() returned %d results, want 0", len(got))
	}
	if _, ok := query.First(); ok {
		t.Error("First() reported a result for an empty query")
	}
	if query.Any() {
		t.Error("Any() returned true for an empty query")
	}
	if got := query.Count(); got != 0 {
		t.Errorf("Count() returned %d, want 0", got)
	}
}
