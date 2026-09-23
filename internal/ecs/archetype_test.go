package ecs

import (
	"reflect"
	"testing"
)

// Test component types
type Position struct{ X, Y int }
type Velocity struct{ X, Y int }
type Health struct{ Value int }

func TestServiceCreateAndComponent(t *testing.T) {
	svc := NewService()

	// Create entity with Position and Velocity
	err := svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 10, Y: 20},
		Velocity{X: 1, Y: 2},
	})

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test Component retrieval
	pos, ok := svc.Component(1, reflect.TypeFor[Position]())
	if !ok {
		t.Fatalf("Component failed: %v", ok)
	}
	if p, ok := pos.(Position); !ok || p.X != 10 || p.Y != 20 {
		t.Errorf("Got wrong position: %+v", pos)
	}

	// Test HasComponent
	if !svc.HasComponent(1, reflect.TypeFor[Position]()) {
		t.Error("HasComponent should return true for existing component")
	}
	if svc.HasComponent(1, reflect.TypeFor[Health]()) {
		t.Error("HasComponent should return false for non-existent component")
	}
	if svc.HasComponent(999, reflect.TypeFor[Position]()) {
		t.Error("HasComponent should return false for non-existent entity")
	}
}

func TestServiceAddComponents(t *testing.T) {
	svc := NewService()

	// Create entity
	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{
		Position{X: 10, Y: 20},
	})

	// Fast path: adding an existing component overwrites without migrating
	result := svc.AddComponents(1, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{
		Position{X: 30, Y: 40},
	})
	if result.Error != nil {
		t.Fatalf("AddComponents failed: %v", result.Error)
	}
	if result.Moved {
		t.Error("Overwriting an existing component should not migrate")
	}

	pos, _ := svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 30 || p.Y != 40 {
		t.Errorf("AddComponents didn't change position: %+v", pos)
	}

	// Migration: existing component values must carry over
	result = svc.AddComponents(1, []reflect.Type{
		reflect.TypeFor[Velocity](),
	}, []any{
		Velocity{X: 3, Y: 4},
	})
	if result.Error != nil {
		t.Fatalf("AddComponents migration failed: %v", result.Error)
	}
	if !result.Moved || result.MovedID != 1 {
		t.Errorf("Migration should report the target entity: %+v", result)
	}

	pos, _ = svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 30 || p.Y != 40 {
		t.Errorf("Existing Position lost in migration: %+v", pos)
	}
	vel, _ := svc.Component(1, reflect.TypeFor[Velocity]())
	if v, ok := vel.(Velocity); !ok || v.X != 3 || v.Y != 4 {
		t.Errorf("New Velocity not set in migration: %+v", vel)
	}

	// The emptied archetype should be cleaned up
	if svc.store.Len() != 1 {
		t.Errorf("Expected 1 archetype after migration, got %d", svc.store.Len())
	}

	// Migration into a pre-existing archetype: entity 3 joins entity 2's
	// Position+Velocity archetype
	svc.Create(2, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{Position{X: 9}, Velocity{X: 9}})
	svc.Create(3, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{Position{X: 7}})

	result = svc.AddComponents(3, []reflect.Type{
		reflect.TypeFor[Velocity](),
	}, []any{
		Velocity{X: 5, Y: 6},
	})
	if result.Error != nil {
		t.Fatalf("AddComponents to pre-existing archetype failed: %v", result.Error)
	}
	if !result.Moved {
		t.Error("Expected migration to pre-existing archetype")
	}

	pos, _ = svc.Component(3, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 7 {
		t.Errorf("Position lost migrating to pre-existing archetype: %+v", pos)
	}

	// Non-existent entity
	result = svc.AddComponents(999, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	if result.Error == nil {
		t.Error("AddComponents should fail for non-existent entity")
	}
}

func TestServiceAddComponentsSwapFixup(t *testing.T) {
	svc := NewService()

	// Three entities share one archetype; migrating the first swap-removes
	// its row, pulling the last entity into index 0. That entity's location
	// must stay readable.
	svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	svc.Create(2, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 2}})
	svc.Create(3, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 3}})

	result := svc.AddComponents(1, []reflect.Type{reflect.TypeFor[Velocity]()}, []any{Velocity{X: 0}})
	if result.Error != nil {
		t.Fatalf("AddComponents failed: %v", result.Error)
	}

	pos, ok := svc.Component(3, reflect.TypeFor[Position]())
	if !ok {
		t.Fatalf("Swapped entity 3 unreadable: %v", ok)
	}
	if p, ok := pos.(Position); !ok || p.X != 3 {
		t.Errorf("Swapped entity 3 has wrong data: %+v", pos)
	}
}

func TestServiceRemoveComponents(t *testing.T) {
	svc := NewService()

	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 10, Y: 20},
		Velocity{X: 3, Y: 4},
	})

	// Migration: removed component gone, remaining values preserved
	result := svc.RemoveComponents(1, []reflect.Type{reflect.TypeFor[Velocity]()})
	if result.Error != nil {
		t.Fatalf("RemoveComponents failed: %v", result.Error)
	}
	if !result.Moved || result.MovedID != 1 {
		t.Errorf("Migration should report the target entity: %+v", result)
	}

	if svc.HasComponent(1, reflect.TypeFor[Velocity]()) {
		t.Error("Velocity should be removed")
	}
	pos, _ := svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 10 || p.Y != 20 {
		t.Errorf("Position lost in migration: %+v", pos)
	}

	// Fast path: removing a component the entity doesn't have is a no-op
	result = svc.RemoveComponents(1, []reflect.Type{reflect.TypeFor[Health]()})
	if result.Error != nil {
		t.Fatalf("RemoveComponents no-op failed: %v", result.Error)
	}
	if result.Moved {
		t.Error("Removing an absent component should not migrate")
	}

	// Removing all components leaves the entity in an empty archetype
	result = svc.RemoveComponents(1, []reflect.Type{reflect.TypeFor[Position]()})
	if result.Error != nil {
		t.Fatalf("RemoveComponents failed: %v", result.Error)
	}
	if svc.HasComponent(1, reflect.TypeFor[Position]()) {
		t.Error("Position should be removed")
	}
	if _, ok := svc.Component(1, reflect.TypeFor[Position]()); ok {
		t.Error("Component should fail on empty archetype")
	}
	if result := svc.Destroy(1); result.Error != nil {
		t.Error("Bare entity should still be destroyable")
	}

	// Non-existent entity
	if result := svc.RemoveComponents(999, []reflect.Type{reflect.TypeFor[Position]()}); result.Error == nil {
		t.Error("RemoveComponents should fail for non-existent entity")
	}

	// Nil type
	if result := svc.RemoveComponents(1, []reflect.Type{nil}); result.Error == nil {
		t.Error("RemoveComponents should fail for nil type")
	}
}

func TestServiceRemoveComponentsSwapFixup(t *testing.T) {
	svc := NewService()

	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{Position{X: 1}, Velocity{X: 1}})
	svc.Create(2, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{Position{X: 2}, Velocity{X: 2}})
	svc.Create(3, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{Position{X: 3}, Velocity{X: 3}})

	result := svc.RemoveComponents(1, []reflect.Type{reflect.TypeFor[Velocity]()})
	if result.Error != nil {
		t.Fatalf("RemoveComponents failed: %v", result.Error)
	}

	pos, ok := svc.Component(3, reflect.TypeFor[Position]())
	if !ok {
		t.Fatalf("Swapped entity 3 unreadable: %v", ok)
	}
	if p, ok := pos.(Position); !ok || p.X != 3 {
		t.Errorf("Swapped entity 3 has wrong data: %+v", pos)
	}
}

func TestServiceRemove(t *testing.T) {
	svc := NewService()

	// Create multiple entities
	svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	svc.Create(2, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 2}})
	svc.Create(3, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 3}})

	// Remove middle entity
	result := svc.Destroy(2)
	if result.Error != nil {
		t.Fatalf("Remove failed: %v", result.Error)
	}

	// Entity 2 should be gone
	if svc.HasComponent(2, reflect.TypeFor[Position]()) {
		t.Error("Removed entity should not have components")
	}

	// Entity 1 and 3 should still exist
	if !svc.HasComponent(1, reflect.TypeFor[Position]()) || !svc.HasComponent(3, reflect.TypeFor[Position]()) {
		t.Error("Other entities should still exist")
	}
}

func TestServiceMatch(t *testing.T) {
	svc := NewService()

	// Create entities with different component combinations
	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{Position{X: 1}, Velocity{X: 1}})

	svc.Create(2, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Health](),
	}, []any{Position{X: 2}, Health{Value: 100}})

	svc.Create(3, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{Position{X: 3}})

	// Match entities with Position
	archetypes := svc.Match(
		[]reflect.Type{reflect.TypeFor[Position]()},
		[]reflect.Type{},
	)
	if len(archetypes) != 3 {
		t.Errorf("Expected 3 archetypes with Position, got %d", len(archetypes))
	}

	// Match entities with Position but not Velocity
	archetypes = svc.Match(
		[]reflect.Type{reflect.TypeFor[Position]()},
		[]reflect.Type{reflect.TypeFor[Velocity]()},
	)
	if len(archetypes) != 2 { // Entities 2 and 3
		t.Errorf("Expected 2 archetypes with Position but not Velocity, got %d", len(archetypes))
	}
}

func TestServiceSetComponent(t *testing.T) {
	svc := NewService()

	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{Position{X: 10, Y: 20}})

	// Update existing component
	err := svc.SetComponent(1, reflect.TypeFor[Position](), Position{X: 30, Y: 40})
	if err != nil {
		t.Fatalf("SetComponent failed: %v", err)
	}

	pos, _ := svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 30 || p.Y != 40 {
		t.Errorf("SetComponent didn't update: %+v", pos)
	}

	// Try to set non-existent component
	err = svc.SetComponent(1, reflect.TypeFor[Velocity](), Velocity{X: 1, Y: 1})
	if err == nil {
		t.Error("SetComponent should fail for non-existent component type")
	}
}

func TestServiceErrorCases(t *testing.T) {
	svc := NewService()

	// Create with mismatched types/values
	err := svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 1}, // Missing second value
	})
	if err == nil {
		t.Error("Create should fail with mismatched types/values")
	}

	// Create duplicate entity
	err = svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	err = svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 2}})
	if err == nil {
		t.Error("Create should fail for duplicate entity ID")
	}

	// Component on non-existent entity
	_, ok := svc.Component(999, reflect.TypeFor[Position]())
	if ok {
		t.Error("Component should fail for non-existent entity")
	}

	// Update non-existent entity
	result := svc.AddComponents(999, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	if result.Error == nil {
		t.Error("Update should fail for non-existent entity")
	}

	// Remove non-existent entity
	result = svc.Destroy(999)
	if result.Error == nil {
		t.Error("Remove should fail for non-existent entity")
	}
}

func TestStoreGetOrCreateVerifiesCollidingHash(t *testing.T) {
	position := reflect.TypeFor[Position]()
	velocity := reflect.TypeFor[Velocity]()

	s := newArchetypes()
	positionArchetype := s.getOrCreate(position)
	// Simulate a collision: point the velocity key's hash at the wrong archetype.
	velocityHash := newKey(velocity).hash()
	s.byHash[velocityHash] = positionArchetype.ID()

	velocityArchetype := s.getOrCreate(velocity)
	if velocityArchetype == positionArchetype {
		t.Fatal("getOrCreate returned the wrong archetype for a colliding hash")
	}
	if !velocityArchetype.Key().equals(newKey(velocity)) {
		t.Fatalf("created archetype has key %v, want the velocity key", velocityArchetype.Key())
	}
	if s.getOrCreate(velocity) != velocityArchetype {
		t.Fatal("repeat getOrCreate with a colliding key should resolve to the same archetype")
	}
}

func TestStoreGetOrCreateScansPastShadowedHash(t *testing.T) {
	position := reflect.TypeFor[Position]()
	velocity := reflect.TypeFor[Velocity]()

	s := newArchetypes()
	positionArchetype := s.getOrCreate(position)
	velocityArchetype := s.getOrCreate(velocity)
	// Simulate a collision: both keys funnel through one hash slot that
	// points at the wrong archetype for position.
	s.byHash[newKey(position).hash()] = velocityArchetype.ID()

	if s.getOrCreate(position) != positionArchetype {
		t.Fatal("getOrCreate should scan past a shadowed hash registration")
	}
}

func TestStoreCleanupEmptyPreservesCollidingRegistration(t *testing.T) {
	position := reflect.TypeFor[Position]()
	velocity := reflect.TypeFor[Velocity]()

	s := newArchetypes()
	positionArchetype := s.getOrCreate(position) // empty: eligible for cleanup
	velocityArchetype := s.getOrCreate(velocity)
	velocityArchetype.insertEntity(1) // survives cleanup
	positionHash := newKey(position).hash()
	s.byHash[positionHash] = velocityArchetype.ID() // collision: slot names another archetype

	s.cleanupEmpty()

	if _, ok := s.get(positionArchetype.ID()); ok {
		t.Error("empty archetype should have been removed")
	}
	if s.byHash[positionHash] != velocityArchetype.ID() {
		t.Error("cleanup must not unregister a hash slot naming another archetype")
	}
}

func TestArchetypeColumnAliasesStorage(t *testing.T) {
	position := reflect.TypeFor[Position]()
	a := newArchetype(1, newKey(position))
	index := a.insertEntity(7)
	a.setComponent(index, position, Position{X: 3})

	col := a.Column(position)
	if len(col) != 1 || col[index].(Position).X != 3 {
		t.Fatalf("column = %v, want the stored value", col)
	}
	col[index] = Position{X: 9}
	if c := a.component(index, position); c.(Position).X != 9 {
		t.Fatal("column must alias archetype storage without copying")
	}
}

func TestArchetypeEachEntity(t *testing.T) {
	position := reflect.TypeFor[Position]()
	a := newArchetype(1, newKey(position))
	a.insertEntity(3)
	a.insertEntity(5)
	a.insertEntity(7)

	var seen []EntityID
	a.EachEntity(func(index int, entityID EntityID) bool {
		if index != len(seen) {
			t.Fatalf("index = %d, want %d", index, len(seen))
		}
		seen = append(seen, entityID)
		return true
	})
	if len(seen) != 3 || seen[0] != 3 || seen[1] != 5 || seen[2] != 7 {
		t.Fatalf("eachEntity visited %v, want [3 5 7]", seen)
	}

	// Yielding false stops iteration.
	seen = seen[:0]
	a.EachEntity(func(index int, entityID EntityID) bool {
		seen = append(seen, entityID)
		return len(seen) < 2
	})
	if len(seen) != 2 {
		t.Fatalf("eachEntity should stop on a false yield, visited %v", seen)
	}
}

func TestServiceMatchIntoReusesCapacity(t *testing.T) {
	position := reflect.TypeFor[Position]()
	velocity := reflect.TypeFor[Velocity]()
	health := reflect.TypeFor[Health]()

	svc := NewService()
	if err := svc.Create(1, []reflect.Type{position}, []any{Position{}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Create(2, []reflect.Type{position, velocity}, []any{Position{}, Velocity{}}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	buf := svc.MatchInto(nil, []reflect.Type{position}, nil)
	if len(buf) != 2 {
		t.Fatalf("MatchInto found %d matches, want 2", len(buf))
	}
	first := &buf[0]

	buf = svc.MatchInto(buf, []reflect.Type{position}, nil)
	if len(buf) != 2 {
		t.Fatalf("MatchInto found %d matches on reuse, want 2", len(buf))
	}
	if &buf[0] != first {
		t.Fatal("MatchInto must reuse the buffer's backing array when capacity fits")
	}

	// A match with no results truncates the buffer correctly.
	buf = svc.MatchInto(buf, []reflect.Type{position, health}, nil)
	if len(buf) != 0 {
		t.Fatalf("MatchInto found %d matches, want 0", len(buf))
	}

	// Match stays consistent with MatchInto for one-off use.
	if got := svc.Match([]reflect.Type{position}, nil); len(got) != 2 {
		t.Fatalf("Match found %d matches, want 2", len(got))
	}
}
