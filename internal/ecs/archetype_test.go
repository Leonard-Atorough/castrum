package ecs

import (
	"reflect"
	"testing"
)

type archetypePosition struct{ X, Y int }
type archetypeVelocity struct{ X, Y int }

func TestKeyNormalizesAndProtectsTypes(t *testing.T) {
	position := reflect.TypeFor[archetypePosition]()
	velocity := reflect.TypeFor[archetypeVelocity]()
	key := newKey(velocity, position, position)

	if key.Len() != 2 || !key.Contains(position) || !key.Contains(velocity) {
		t.Fatalf("unexpected normalized key: %v", key)
	}
	if !key.Equals(newKey(position, velocity)) {
		t.Fatalf("equivalent keys should compare equal: %v and %v", key, newKey(position, velocity))
	}

	types := key.Types()
	types[0] = nil
	if key.Types()[0] == nil {
		t.Fatal("Types should return a copy")
	}
}

func TestKeyPredicatesAndFormatting(t *testing.T) {
	position := reflect.TypeFor[archetypePosition]()
	velocity := reflect.TypeFor[archetypeVelocity]()
	key := newKey(position, velocity)

	if key.String() == "" || key.IndexOf(position) < 0 || key.IndexOf(reflect.TypeFor[int]()) != -1 {
		t.Fatalf("unexpected key formatting/index: %q, %d", key.String(), key.IndexOf(position))
	}
	if !key.ContainsAll(position, velocity) || key.ContainsAll(reflect.TypeFor[int]()) {
		t.Fatal("ContainsAll returned an unexpected result")
	}
	if !key.ContainsAny(reflect.TypeFor[int](), velocity) || key.ContainsAny(reflect.TypeFor[int]()) {
		t.Fatal("ContainsAny returned an unexpected result")
	}
	if !key.ContainsNone(reflect.TypeFor[int]()) || key.ContainsNone(velocity) {
		t.Fatal("ContainsNone returned an unexpected result")
	}
	if !key.ContainsKey(newKey(position)) || key.ContainsKey(newKey(reflect.TypeFor[int]())) {
		t.Fatal("ContainsKey returned an unexpected result")
	}
	if !key.ContainsAnyKey(newKey(reflect.TypeFor[int](), velocity)) || key.ContainsAnyKey(newKey(reflect.TypeFor[int]())) {
		t.Fatal("ContainsAnyKey returned an unexpected result")
	}
	if !key.ContainsNoKey(newKey(reflect.TypeFor[int]())) || key.ContainsNoKey(newKey(velocity)) {
		t.Fatal("ContainsNoKey returned an unexpected result")
	}
	if !key.Equals(newKey(position, velocity)) || key.Equals(newKey(position)) {
		t.Fatal("Equals returned an unexpected result")
	}
}

func TestArchetypeSetGetAndRemove(t *testing.T) {
	positionType := reflect.TypeFor[archetypePosition]()
	velocityType := reflect.TypeFor[archetypeVelocity]()
	arch := newArchetype(1, newKey(positionType, velocityType))

	first := arch.AddEntity(10)
	second := arch.AddEntity(20)
	if !arch.Set(first, positionType, archetypePosition{X: 1}) ||
		!arch.Set(second, positionType, archetypePosition{X: 2}) {
		t.Fatal("expected component values to be set")
	}

	value, ok := arch.Get(second, positionType)
	if !ok || value.(archetypePosition).X != 2 {
		t.Fatalf("unexpected component value: %#v, %v", value, ok)
	}

	movedID, moved := arch.RemoveEntity(first)
	if !moved || movedID != 20 || arch.Len() != 1 {
		t.Fatalf("unexpected swap-remove result: id=%d moved=%v len=%d", movedID, moved, arch.Len())
	}
	value, ok = arch.Get(0, positionType)
	if !ok || value.(archetypePosition).X != 2 {
		t.Fatalf("component column was not swap-removed: %#v, %v", value, ok)
	}
}

func TestArchetypeAccessorsAndInvalidOperations(t *testing.T) {
	positionType := reflect.TypeFor[archetypePosition]()
	arch := newArchetype(3, newKey(positionType))
	if arch.ID() != 3 || arch.Key().Len() != 1 || arch.ComponentTypes()[0] != positionType {
		t.Fatal("archetype accessors returned unexpected values")
	}
	if !arch.HasComponent(positionType) || arch.HasComponent(reflect.TypeFor[int]()) {
		t.Fatal("HasComponent returned an unexpected result")
	}
	if ids := arch.EntityIDs(); len(ids) != 0 {
		t.Fatalf("new archetype should have no entities, got %v", ids)
	}
	if _, ok := arch.Get(-1, positionType); ok {
		t.Fatal("negative Get index should fail")
	}
	if arch.Set(-1, positionType, archetypePosition{}) || arch.Set(0, reflect.TypeFor[int](), 1) {
		t.Fatal("invalid Set should fail")
	}
	if moved, didMove := arch.RemoveEntity(0); didMove || moved != 0 {
		t.Fatal("invalid RemoveEntity should be a no-op")
	}
}

func TestArchetypeStoreMatchingAndMisses(t *testing.T) {
	store := newArchetypeStore()
	positionType := reflect.TypeFor[archetypePosition]()
	velocityType := reflect.TypeFor[archetypeVelocity]()
	store.GetOrCreate(positionType, velocityType).AddEntity(1)
	store.GetOrCreate(positionType).AddEntity(2)

	if _, ok := store.Get(999); ok {
		t.Fatal("missing archetype lookup should fail")
	}
	if _, ok := store.FindByTypes(reflect.TypeFor[int]()); ok {
		t.Fatal("missing archetype lookup should fail")
	}
	if got := len(store.Matching(newKey(positionType), newKey(velocityType))); got != 1 {
		t.Fatalf("unexpected matching count: got %d want 1", got)
	}
	if got := len(store.MatchingByTypes([]reflect.Type{positionType}, []reflect.Type{velocityType})); got != 1 {
		t.Fatalf("unexpected type matching count: got %d want 1", got)
	}
}

func TestArchetypeStoreCleanupRemovesEmptyNonEmptyKey(t *testing.T) {
	store := newArchetypeStore()
	arch := store.GetOrCreate(reflect.TypeFor[archetypePosition]())
	arch.AddEntity(1)
	arch.RemoveEntity(0)

	store.CleanupEmpty()
	if store.Count() != 0 {
		t.Fatalf("expected empty archetype to be removed, got %d", store.Count())
	}
	if _, ok := store.FindByTypes(reflect.TypeFor[archetypePosition]()); ok {
		t.Fatal("removed archetype remained indexed by key")
	}
}
