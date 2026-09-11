package ecs

import (
	"errors"
	"slices"
	"testing"
)

func TestNewWorld(t *testing.T) {
	w := NewWorld()
	if w.Count() != 0 {
		t.Errorf("expected a new world to have 0 entities, got %d", w.Count())
	}
}

func TestWorld_Create(t *testing.T) {
	t.Run("bare entity has a template and no components", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Province")

		if !w.HasEntity(e.ID) {
			t.Error("created entity should exist in the world")
		}
		if e.Template() != "Province" {
			t.Errorf("expected template %q, got %q", "Province", e.Template())
		}
		if got := w.Components(e.ID); len(got) != 0 {
			t.Errorf("bare entity should have no components, got %#v", got)
		}
	})

	t.Run("CreateWithComponents populates components immediately", func(t *testing.T) {
		w := NewWorld()
		pos := TestPosition{X: 1, Y: 2}
		vel := TestVelocity{X: 0.5, Y: 0.5}

		e, err := w.CreateWithComponents("Unit", pos, vel)
		if err != nil {
			t.Errorf("CreateWithComponents failed: %v", err)
		}

		got, err := w.GetComponent[TestPosition](e.ID)
		if err != nil {
			t.Errorf("GetComponent failed: %v", err)
		}
		if got != pos {
			t.Errorf("expected position %#v, got %#v", pos, got)
		}
		if !w.HasComponent[TestVelocity](e.ID) {
			t.Error("expected entity to have TestVelocity")
		}
	})

	t.Run("CreateMany creates the requested number of distinct entities", func(t *testing.T) {
		w := NewWorld()
		entities := w.CreateMany("Grunt", 3)

		if len(entities) != 3 {
			t.Errorf("expected 3 entities, got %d", len(entities))
		}
		seen := make(map[EntityID]bool)
		for _, e := range entities {
			if seen[e.ID] {
				t.Errorf("duplicate entity ID %d", e.ID)
			}
			seen[e.ID] = true
			if !w.HasEntity(e.ID) {
				t.Errorf("entity %d should exist in the world", e.ID)
			}
		}
	})
}

func TestWorld_DestroyEntity(t *testing.T) {
	t.Run("destroyed entity is removed after Cleanup", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")

		if err := w.DestroyEntity(e.ID, false); err != nil {
			t.Errorf("DestroyEntity failed: %v", err)
		}
		if !w.HasEntity(e.ID) {
			t.Error("entity should still exist until Cleanup runs")
		}

		w.Cleanup()
		if w.HasEntity(e.ID) {
			t.Error("entity should be gone after Cleanup")
		}
	})

	t.Run("cascade destroys descendants", func(t *testing.T) {
		w := NewWorld()
		parent := w.Create("Generic")
		child := w.Create("Generic")
		grandchild := w.Create("Generic")
		w.SetParent(child.ID, parent.ID)
		w.SetParent(grandchild.ID, child.ID)

		if err := w.DestroyEntity(parent.ID, true); err != nil {
			t.Errorf("DestroyEntity failed: %v", err)
		}
		w.Cleanup()

		for _, id := range []EntityID{parent.ID, child.ID, grandchild.ID} {
			if w.HasEntity(id) {
				t.Errorf("entity %d should be destroyed by cascade", id)
			}
		}
	})

	t.Run("non-cascade destroy detaches children instead of destroying them", func(t *testing.T) {
		w := NewWorld()
		parent := w.Create("Generic")
		child := w.Create("Generic")
		w.SetParent(child.ID, parent.ID)

		if err := w.DestroyEntity(parent.ID, false); err != nil {
			t.Errorf("DestroyEntity failed: %v", err)
		}
		w.Cleanup()

		if !w.HasEntity(child.ID) {
			t.Error("child should survive a non-cascade destroy")
		}
		if _, ok := w.ParentOf(child.ID); ok {
			t.Error("child should be detached from the destroyed parent")
		}
	})

	t.Run("destroying an unknown entity returns an error", func(t *testing.T) {
		w := NewWorld()
		if err := w.DestroyEntity(999, false); !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("expected ErrEntityNotFound, got %v", err)
		}
	})

	t.Run("removing a middle entity keeps surviving entities' component data correct", func(t *testing.T) {
		// Regression test: archetype storage uses swap-remove, so component
		// slices must stay aligned with the entities slice after a removal
		// from the middle of an archetype.
		w := NewWorld()
		a, _ := w.CreateWithComponents("Unit", TestPosition{X: 1})
		b, _ := w.CreateWithComponents("Unit", TestPosition{X: 2})
		c, _ := w.CreateWithComponents("Unit", TestPosition{X: 3})

		if err := w.DestroyEntity(b.ID, false); err != nil {
			t.Errorf("DestroyEntity failed: %v", err)
		}
		w.Cleanup()

		gotA, err := w.GetComponent[TestPosition](a.ID)
		if err != nil {
			t.Errorf("GetComponent(a) failed: %v", err)
		}
		if gotA.X != 1 {
			t.Errorf("entity a's position corrupted after sibling removal: got %#v", gotA)
		}

		gotC, err := w.GetComponent[TestPosition](c.ID)
		if err != nil {
			t.Errorf("GetComponent(c) failed: %v", err)
		}
		if gotC.X != 3 {
			t.Errorf("entity c's position corrupted after sibling removal: got %#v", gotC)
		}
	})
}

func TestWorld_Components(t *testing.T) {
	t.Run("AddComponent migrates the entity and stores the value", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")

		if err := w.AddComponent(e.ID, TestPosition{X: 1, Y: 2}); err != nil {
			t.Errorf("AddComponent failed: %v", err)
		}
		if !w.HasComponent[TestPosition](e.ID) {
			t.Error("entity should have TestPosition after AddComponent")
		}

		got, err := w.GetComponent[TestPosition](e.ID)
		if err != nil {
			t.Errorf("GetComponent failed: %v", err)
		}
		if got != (TestPosition{X: 1, Y: 2}) {
			t.Errorf("unexpected component value: %#v", got)
		}
	})

	t.Run("AddComponent with multiple new types stores every value", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")

		err := w.AddComponent(e.ID, TestPosition{X: 1, Y: 2}, TestVelocity{X: 3, Y: 4}, TestHealth{Value: 100})
		if err != nil {
			t.Errorf("AddComponent failed: %v", err)
		}

		if !w.HasComponent[TestPosition](e.ID) {
			t.Errorf("expected entity to have TestPosition component")
		}
		if !w.HasComponent[TestVelocity](e.ID) {
			t.Errorf("expected entity to have TestVelocity component")
		}
		if !w.HasComponent[TestHealth](e.ID) {
			t.Errorf("expected entity to have TestHealth component")
		}
	})

	t.Run("AddComponent on an existing type updates the value in place", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")
		w.AddComponent(e.ID, TestHealth{Value: 100})

		if err := w.AddComponent(e.ID, TestHealth{Value: 50}); err != nil {
			t.Errorf("AddComponent failed: %v", err)
		}

		got, _ := w.GetComponent[TestHealth](e.ID)
		if got != (TestHealth{Value: 50}) {
			t.Errorf("expected updated health value, got %#v", got)
		}
	})

	t.Run("SetComponent overwrites an existing value", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")
		w.AddComponent(e.ID, TestHealth{Value: 100})

		if err := w.SetComponent(e.ID, TestHealth{Value: 10}); err != nil {
			t.Errorf("SetComponent failed: %v", err)
		}

		got, _ := w.GetComponent[TestHealth](e.ID)
		if got != (TestHealth{Value: 10}) {
			t.Errorf("expected health 10, got %#v", got)
		}
	})

	t.Run("RemoveComponent drops one type but keeps the others", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")
		w.AddComponent(e.ID, TestPosition{X: 1, Y: 2}, TestVelocity{X: 3, Y: 4})

		if err := w.RemoveComponent[TestPosition](e.ID); err != nil {
			t.Errorf("RemoveComponent failed: %v", err)
		}
		if w.HasComponent[TestPosition](e.ID) {
			t.Error("TestPosition should have been removed")
		}
		if !w.HasComponent[TestVelocity](e.ID) {
			t.Error("TestVelocity should still be present")
		}
	})

	t.Run("RemoveComponent for a type the entity doesn't have is a no-op", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")

		if err := w.RemoveComponent[TestPosition](e.ID); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetComponent for a type the entity doesn't have returns an error", func(t *testing.T) {
		w := NewWorld()
		e := w.Create("Generic")

		if _, err := w.GetComponent[TestPosition](e.ID); err == nil {
			t.Error("expected an error getting a component the entity doesn't have")
		}
	})

	t.Run("component operations on an unknown entity return an error", func(t *testing.T) {
		w := NewWorld()
		unknown := EntityID(999)

		if err := w.AddComponent(unknown, TestPosition{}); !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("AddComponent: expected ErrEntityNotFound, got %v", err)
		}
		if _, err := w.GetComponent[TestPosition](unknown); !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("GetComponent: expected ErrEntityNotFound, got %v", err)
		}
		if err := w.RemoveComponent[TestPosition](unknown); !errors.Is(err, ErrEntityNotFound) {
			t.Errorf("RemoveComponent: expected ErrEntityNotFound, got %v", err)
		}
	})
}

func TestWorld_Query(t *testing.T) {
	w := NewWorld()
	posOnly := w.Create("Generic")
	w.AddComponent(posOnly.ID, TestPosition{X: 1})

	posAndVel := w.Create("Generic")
	w.AddComponent(posAndVel.ID, TestPosition{X: 2}, TestVelocity{X: 1})

	velOnly := w.Create("Generic")
	w.AddComponent(velOnly.ID, TestVelocity{X: 3})

	t.Run("NewQuery Builder returns entities that have every requested component", func(t *testing.T) {
		got := w.NewQuery().WithRequiredComponents(TestPosition{}, TestVelocity{}).EntityIDs()
		if !idsMatchUnordered(got, []EntityID{posAndVel.ID}) {
			t.Errorf("expected only the entity with both components, got %#v", got)
		}
	})

	t.Run("Query required component matches archetypes with extra components", func(t *testing.T) {
		got := w.NewQuery().WithRequiredComponents(TestPosition{}).EntityIDs()
		if !idsMatchUnordered(got, []EntityID{posOnly.ID, posAndVel.ID}) {
			t.Errorf("expected entities with TestPosition including supersets, got %#v", got)
		}
	})

	t.Run("Query excluded component filters out matching entities", func(t *testing.T) {
		got := w.NewQuery().WithExcludedComponents(TestVelocity{}).EntityIDs()
		if !idsMatchUnordered(got, []EntityID{posOnly.ID}) {
			t.Errorf("expected entities without TestVelocity, got %#v", got)
		}
	})

	t.Run("Query for a component nobody has returns nothing", func(t *testing.T) {
		if got := w.NewQuery().WithRequiredComponents(TestHealth{}).EntityIDs(); len(got) != 0 {
			t.Errorf("expected no matches, got %#v", got)
		}
	})
}

func TestWorld_TypedComponentHelpers(t *testing.T) {
	w := NewWorld()
	e := w.Create("Generic")
	w.AddComponent(e.ID, TestPosition{X: 1, Y: 1})

	t.Run("SetComponent updates an existing typed component", func(t *testing.T) {
		if err := w.SetComponent(e.ID, TestPosition{X: 5, Y: 6}); err != nil {
			t.Errorf("SetComponent failed: %v", err)
		}

		got, err := w.GetComponent[TestPosition](e.ID)
		if err != nil {
			t.Errorf("GetComponent failed: %v", err)
		}
		if got != (TestPosition{X: 5, Y: 6}) {
			t.Errorf("unexpected value: %#v", got)
		}
	})

	t.Run("HasComponent reflects presence of the typed component", func(t *testing.T) {
		if !w.HasComponent[TestPosition](e.ID) {
			t.Error("expected entity to have TestPosition")
		}
		if w.HasComponent[TestVelocity](e.ID) {
			t.Error("did not expect entity to have TestVelocity")
		}
	})

	t.Run("GetComponent returns an error for a type the entity doesn't have", func(t *testing.T) {
		if _, err := w.GetComponent[TestVelocity](e.ID); err == nil {
			t.Error("expected an error")
		}
	})
}

func TestWorld_Hierarchy(t *testing.T) {
	w := NewWorld()
	parent := w.Create("Generic")
	child := w.Create("Generic")

	t.Run("SetParent establishes a parent/child relationship", func(t *testing.T) {
		w.SetParent(child.ID, parent.ID)

		gotParent, ok := w.ParentOf(child.ID)
		if !ok || gotParent != parent.ID {
			t.Errorf("expected parent %d, got %d (ok=%v)", parent.ID, gotParent, ok)
		}
		if !slices.Contains(w.ChildrenOf(parent.ID), child.ID) {
			t.Error("expected child to be listed under parent")
		}
	})

	t.Run("Detach removes the relationship", func(t *testing.T) {
		w.Detach(child.ID)

		if _, ok := w.ParentOf(child.ID); ok {
			t.Error("expected child to have no parent after Detach")
		}
	})
}

func TestWorld_Reset(t *testing.T) {
	w := NewWorld()
	parent := w.Create("Generic")
	child := w.Create("Generic")
	w.AddComponent(child.ID, TestPosition{X: 1})
	w.SetParent(child.ID, parent.ID)

	w.Reset()

	if w.Count() != 0 {
		t.Errorf("expected 0 entities after reset, got %d", w.Count())
	}
	if w.HasEntity(child.ID) {
		t.Error("entity should not exist after reset")
	}

	fresh := w.Create("Generic")
	if !w.HasEntity(fresh.ID) {
		t.Error("should be able to create entities after reset")
	}
}

// =============================================================================
// Component Edge Case Tests
// =============================================================================

func TestWorld_ComponentEdgeCases(t *testing.T) {
	t.Run("GetComponentFromEmptyArchetype", func(t *testing.T) {
		world := NewWorld()

		entity := world.Create("Generic")

		_, err := world.GetComponent[TestPosition](entity.ID)
		if err == nil {
			t.Error("Expected error when getting non-existent component")
		}
	})

	t.Run("RemoveNonExistentComponent", func(t *testing.T) {
		world := NewWorld()

		entity := world.Create("Generic")

		err := world.RemoveComponent[TestPosition](entity.ID)
		if err != nil {
			t.Errorf("Expected no error when removing non-existent component, got: %v", err)
		}
	})

	t.Run("QueryNonExistentComponent", func(t *testing.T) {
		world := NewWorld()

		world.Create("Generic")

		result := world.NewQuery().WithRequiredComponents(TestPosition{}, TestVelocity{}).EntityIDs()
		if len(result) != 0 {
			t.Error("Expected empty result for non-existent component combination")
		}
	})

	t.Run("GetComponentAfterDestruction", func(t *testing.T) {
		world := NewWorld()

		entity := world.Create("Generic")
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})

		world.DestroyEntity(entity.ID, false)
		world.Cleanup()

		_, err := world.GetComponent[TestPosition](entity.ID)
		if err == nil {
			t.Error("Expected error when getting component from destroyed entity")
		}
	})

	t.Run("AddComponentToDestroyedEntity", func(t *testing.T) {
		world := NewWorld()

		entity := world.Create("Generic")
		world.DestroyEntity(entity.ID, false)
		world.Cleanup()

		err := world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})
		if err == nil {
			t.Error("Expected error when adding component to destroyed entity")
		}
	})
}
