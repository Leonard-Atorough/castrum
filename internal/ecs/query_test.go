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
		t.Fatalf("expected only e2 (X>1), got %v (e1=%d, e2=%d)", got, e1.ID, e2.ID)
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
		t.Fatalf("expected only e2 to satisfy both filters, got %v", got)
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
		t.Fatalf("expected only entities tagged scene 'a', got %v", got)
	}
}

func TestQuery_InScene_NoMatches(t *testing.T) {
	w := NewWorld()

	_, _ = w.CreateWithComponents("a1", components.SceneTag{SceneID: "a"})

	got := w.NewQuery().InScene("nonexistent").EntityIDs()
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %v", got)
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
		t.Fatalf("expected only the entity with both required components and scene tag, got %v", got)
	}
}
