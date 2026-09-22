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
	result := svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 10, Y: 20},
		Velocity{X: 1, Y: 2},
	})

	if !result.Success {
		t.Fatalf("Create failed: %v", result.Error)
	}

	// Test Component retrieval
	pos, error := svc.Component(1, reflect.TypeFor[Position]())
	if error != nil {
		t.Fatalf("Component failed: %v", error)
	}
	if p, ok := pos.(Position); !ok || p.X != 10 || p.Y != 20 {
		t.Errorf("Got wrong position: %+v", pos)
	}

	// Test HasComponent
	has, error := svc.HasComponent(1, reflect.TypeFor[Position]())
	if error != nil || !has {
		t.Error("HasComponent should return true for existing component")
	}
	has, error = svc.HasComponent(1, reflect.TypeFor[Health]())
	if error != nil || has {
		t.Error("HasComponent should return false for non-existent component")
	}
}

func TestServiceUpdate(t *testing.T) {
	svc := NewService()

	// Create entity
	svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{
		Position{X: 10, Y: 20},
	})

	// Update with same archetype (same component types)
	result := svc.Update(1, []reflect.Type{
		reflect.TypeFor[Position](),
	}, []any{
		Position{X: 30, Y: 40},
	})
	if !result.Success {
		t.Fatalf("Update failed: %v", result.Error)
	}

	// Verify update
	pos, _ := svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 30 || p.Y != 40 {
		t.Errorf("Update didn't change position: %+v", pos)
	}

	// Update with different archetype (different component types)
	result = svc.Update(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 50, Y: 60},
		Velocity{X: 3, Y: 4},
	})
	if !result.Success {
		t.Fatalf("Update to new archetype failed: %v", result.Error)
	}
}

func TestServiceRemove(t *testing.T) {
	svc := NewService()

	// Create multiple entities
	svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	svc.Create(2, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 2}})
	svc.Create(3, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 3}})

	// Remove middle entity
	result := svc.Remove(2)
	if !result.Success {
		t.Fatalf("Remove failed: %v", result.Error)
	}

	// Entity 2 should be gone
	has, error := svc.HasComponent(2, reflect.TypeFor[Position]())
	if error == nil && has {
		t.Error("Removed entity should not have components")
	}

	// Entity 1 and 3 should still exist
	has1, error1 := svc.HasComponent(1, reflect.TypeFor[Position]())
	has3, error3 := svc.HasComponent(3, reflect.TypeFor[Position]())
	if error1 != nil || error3 != nil || !has1 || !has3 {
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
	success, error := svc.SetComponent(1, reflect.TypeFor[Position](), Position{X: 30, Y: 40})
	if !success {
		t.Fatalf("SetComponent failed: %v", error)
	}

	pos, _ := svc.Component(1, reflect.TypeFor[Position]())
	if p, ok := pos.(Position); !ok || p.X != 30 || p.Y != 40 {
		t.Errorf("SetComponent didn't update: %+v", pos)
	}

	// Try to set non-existent component
	success, error = svc.SetComponent(1, reflect.TypeFor[Velocity](), Velocity{X: 1, Y: 1})
	if success {
		t.Error("SetComponent should fail for non-existent component type")
	}
}

func TestServiceErrorCases(t *testing.T) {
	svc := NewService()

	// Create with mismatched types/values
	result := svc.Create(1, []reflect.Type{
		reflect.TypeFor[Position](),
		reflect.TypeFor[Velocity](),
	}, []any{
		Position{X: 1}, // Missing second value
	})
	if result.Success {
		t.Error("Create should fail with mismatched types/values")
	}

	// Create duplicate entity
	svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	result = svc.Create(1, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 2}})
	if result.Success {
		t.Error("Create should fail for duplicate entity ID")
	}

	// Component on non-existent entity
	_, error := svc.Component(999, reflect.TypeFor[Position]())
	if error == nil {
		t.Error("Component should fail for non-existent entity")
	}

	// Update non-existent entity
	result = svc.Update(999, []reflect.Type{reflect.TypeFor[Position]()}, []any{Position{X: 1}})
	if result.Success {
		t.Error("Update should fail for non-existent entity")
	}

	// Remove non-existent entity
	result = svc.Remove(999)
	if result.Success {
		t.Error("Remove should fail for non-existent entity")
	}
}
