package ecs

import "testing"

func TestWorld_SpawnAndDestroy(t *testing.T) {
	w := NewWorld()

	// Spawn a Province entity
	provinceID := w.Spawn("Province")
	provinceEntity, exists := w.entities[provinceID]
	if !exists {
		t.Fatalf("Province entity with ID %d not found in world", provinceID)
	}
	if provinceEntity.Template() != "Province" {
		t.Fatalf("Expected template 'Province', got %q", provinceEntity.Template())
	}
	if !provinceEntity.HasTag("Province") {
		t.Fatalf("Expected entity to have tag 'Province'")
	}

	// Spawn a City entity
	cityID := w.Spawn("City")
	cityEntity, exists := w.entities[cityID]
	if !exists {
		t.Fatalf("City entity with ID %d not found in world", cityID)
	}
	if cityEntity.Template() != "City" {
		t.Fatalf("Expected template 'City', got %q", cityEntity.Template())
	}
	if !cityEntity.HasTag("City") {
		t.Fatalf("Expected entity to have tag 'City'")
	}

	// Destroy the Province entity
	w.Destroy(provinceID, false)
	w.Cleanup()

	if _, exists := w.entities[provinceID]; exists {
		t.Fatalf("Province entity with ID %d should have been removed from world", provinceID)
	}
}

func TestWorld_DestroyNonExistentEntity(t *testing.T) {
	w := NewWorld()
	w.Destroy(999, false) // Attempt to destroy a non-existent entity
	if len(w.destroyed) != 0 {
		t.Fatalf("Expected no destroyed entities, got %d", len(w.destroyed))
	}
	w.Cleanup() // Ensure cleanup doesn't panic or remove anything
}

func TestWorld_DestroyWithCascade(t *testing.T) {
	type ComponentA struct{}
	
	w := NewWorld()

	// Spawn a Province entity
	provinceID := w.Spawn("Province")

	// Spawn a City entity and set its parent to the Province
	cityID := w.Spawn("City")
	w.SetParent(cityID, provinceID)

	// Destroy the Province entity with cascade
	w.Destroy(provinceID, true)
	w.Cleanup()

	if _, exists := w.entities[provinceID]; exists {
		t.Fatalf("Province entity with ID %d should have been removed from world", provinceID)
	}
	if _, exists := w.entities[cityID]; exists {
		t.Fatalf("City entity with ID %d should have been removed from world due to cascade", cityID)
	}
}

func TestWorld_DestroyWithoutCascade(t *testing.T) {
	w := NewWorld()

	// Spawn a Province entity
	provinceID := w.Spawn("Province")

	// Spawn a City entity and set its parent to the Province
	cityID := w.Spawn("City")
	w.SetParent(cityID, provinceID)

	// Destroy the Province entity without cascade
	w.Destroy(provinceID, false)
	w.Cleanup()

	if _, exists := w.entities[provinceID]; exists {
		t.Fatalf("Province entity with ID %d should have been removed from world", provinceID)
	}
	if _, exists := w.entities[cityID]; !exists {
		t.Fatalf("City entity with ID %d should still exist in world", cityID)
	}
}
