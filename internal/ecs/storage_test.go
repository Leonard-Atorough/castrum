package ecs

import (
	"reflect"
	"testing"
)

func TestStorageCreateReadWriteAndDelete(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()
	velocityType := reflect.TypeFor[archetypeVelocity]()
	location, err := storage.CreateEntity(7,
		[]reflect.Type{positionType, velocityType},
		[]any{archetypePosition{X: 3}, archetypeVelocity{Y: 4}},
	)
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}

	value, ok := storage.Get(location, positionType)
	if !ok || value.(archetypePosition).X != 3 {
		t.Fatalf("unexpected stored position: %#v, %v", value, ok)
	}
	if !storage.Set(location, positionType, archetypePosition{X: 9}) {
		t.Fatal("expected Set to succeed")
	}
	value, ok = storage.Get(location, positionType)
	if !ok || value.(archetypePosition).X != 9 {
		t.Fatalf("unexpected updated position: %#v, %v", value, ok)
	}

	result, ok := storage.DeleteEntity(location)
	if !ok || result.Moved {
		t.Fatalf("unexpected delete result for sole entity: %#v, ok=%v", result, ok)
	}
}

func TestStorageMoveWithinSameArchetypeUpdatesInPlace(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()
	location, err := storage.CreateEntity(7, []reflect.Type{positionType}, []any{archetypePosition{X: 1}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}

	updated, result, ok := storage.MoveEntity(7, location, []reflect.Type{positionType}, []any{archetypePosition{X: 8}})
	if !ok || result.Moved {
		t.Fatalf("unexpected same-archetype move result: %#v, ok=%v", result, ok)
	}
	if updated != location {
		t.Fatalf("same-archetype move changed location: got %#v want %#v", updated, location)
	}
	value, ok := storage.Get(updated, positionType)
	if !ok || value.(archetypePosition).X != 8 {
		t.Fatalf("same-archetype move did not update value: %#v, %v", value, ok)
	}
}

func TestStorageRejectsMismatchedComponentDataWithoutMutation(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()

	if _, err := storage.CreateEntity(1, []reflect.Type{positionType}, nil); err == nil {
		t.Fatal("expected mismatched create data to fail")
	}
	if storage.archetypes.Count() != 0 {
		t.Fatal("invalid create mutated storage")
	}

	location, err := storage.CreateEntity(1, []reflect.Type{positionType}, []any{archetypePosition{}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	if _, _, ok := storage.MoveEntity(1, location, []reflect.Type{positionType}, []any{}); ok {
		t.Fatal("expected mismatched move data to fail")
	}
	if _, err := storage.CreateEntity(2, []reflect.Type{nil}, []any{nil}); err == nil {
		t.Fatal("expected nil component type to fail")
	}
	if _, err := storage.CreateEntity(2, []reflect.Type{positionType}, []any{archetypeVelocity{}}); err == nil {
		t.Fatal("expected incompatible component value to fail")
	}
}

func TestStorageRejectsInvalidDeleteLocation(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()
	location, err := storage.CreateEntity(1, []reflect.Type{positionType}, []any{archetypePosition{}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}

	result, ok := storage.DeleteEntity(EntityLocation{ArchetypeID: location.ArchetypeID, Index: 1})
	if ok || result.Moved {
		t.Fatalf("expected invalid delete to be rejected: %#v, ok=%v", result, ok)
	}
	if _, ok := storage.Get(location, positionType); !ok {
		t.Fatal("invalid delete removed the entity")
	}
}

func TestStorageMoveReportsDisplacedEntity(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()
	first, err := storage.CreateEntity(1, []reflect.Type{positionType}, []any{archetypePosition{X: 1}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	second, err := storage.CreateEntity(2, []reflect.Type{positionType}, []any{archetypePosition{X: 2}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}

	_, result, ok := storage.MoveEntity(1, first, nil, nil)
	if !ok || !result.Moved || result.MovedID != 2 {
		t.Fatalf("expected entity 2 to be reported as displaced: %#v, ok=%v", result, ok)
	}
	if _, ok := storage.Get(second, positionType); ok {
		t.Fatal("old location should no longer be valid for the displaced entity")
	}
}

func TestStorageRejectsMissingLocationsAndComponents(t *testing.T) {
	storage := NewStorage()
	positionType := reflect.TypeFor[archetypePosition]()
	location, err := storage.CreateEntity(1, []reflect.Type{positionType}, []any{archetypePosition{}})
	if err != nil {
		t.Fatalf("CreateEntity failed: %v", err)
	}
	missing := EntityLocation{ArchetypeID: 999, Index: 0}
	if _, ok := storage.Get(missing, positionType); ok || storage.Set(missing, positionType, archetypePosition{}) || storage.Has(missing, positionType) {
		t.Fatal("missing location should fail Get, Set, and Has")
	}
	if _, ok := storage.Get(location, reflect.TypeFor[int]()); ok || storage.Set(location, reflect.TypeFor[int](), 1) || storage.Has(location, reflect.TypeFor[int]()) {
		t.Fatal("missing component should fail Get, Set, and Has")
	}
	if _, _, ok := storage.MoveEntity(1, missing, nil, nil); ok {
		t.Fatal("move from a missing location should fail")
	}
}
