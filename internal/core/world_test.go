package core

import (
	"reflect"
	"slices"
	"testing"
)

func TestWorldCreation(t *testing.T) {
	t.Run("NewWorld initializes correctly", func(t *testing.T) {
		w := NewWorld()
		if w == nil {
			t.Fatal("NewWorld should not return nil")
		}

		if len(w.entities) != 0 {
			t.Fatal("NewWorld should start with no entities")
		}

		if w.archetypeManager == nil {
			t.Fatal("NewWorld should initialize the archetype manager")
		}
		if w.hierarchy == nil {
			t.Fatal("NewWorld should initialize the hierarchy")
		}
		if w.destroyed == nil {
			t.Fatal("NewWorld should initialize the destroyed entities list")
		}
		if w.nextID.Load() != 0 {
			t.Fatal("NewWorld should initialize nextID to 0")
		}
	})

	t.Run("QueryByTemplate returns correct entities", func(t *testing.T) {
		w := NewWorld()
		provinceID := w.CreateEntity("Province")
		cityID := w.CreateEntity("City")
		genericID1 := w.CreateEntity("Generic")
		genericID2 := w.CreateEntity("Generic")

		provinces := w.QueryByTemplate("Province")
		if len(provinces) != 1 || provinces[0] != provinceID {
			t.Fatal("QueryByTemplate for 'Province' returned incorrect entities")
		}

		cities := w.QueryByTemplate("City")
		if len(cities) != 1 || cities[0] != cityID {
			t.Fatal("QueryByTemplate for 'City' returned incorrect entities")
		}

		generics := w.QueryByTemplate("Generic")
		if len(generics) != 2 || (generics[0] != genericID1 && generics[0] != genericID2) || (generics[1] != genericID1 && generics[1] != genericID2) {
			t.Fatal("QueryByTemplate for 'Generic' returned incorrect entities")
		}
	})

}

func TestWorld_EntityLifecycle(t *testing.T) {
	t.Run("Create, Get, and Destroy Entity", func(t *testing.T) {
		w := NewWorld()
		id := w.CreateEntity("Generic")
		if !w.HasEntity(id) {
			t.Fatal("Created entity should exist")
		}
		if _, exists := w.GetEntity(id); !exists {
			t.Fatal("GetEntity should return the entity after creation")
		}
		if exists := w.HasEntity(id); !exists {
			t.Fatal("Entity should exist before destruction")
		}
		if err := w.DestroyEntity(id, false); err != nil {
			t.Fatalf("DestroyEntity failed: %v", err)
		}
		w.Cleanup()
		if w.HasEntity(id) {
			t.Fatal("Entity should not exist after destruction and cleanup")
		}
	})

	t.Run("DestroyEntity with non-existent entity", func(t *testing.T) {
		w := NewWorld()
		err := w.DestroyEntity(999, false)
		if err == nil {
			t.Fatal("DestroyEntity should fail for non-existent entity")
		}
	})

	t.Run("CreateEntity with template tags", func(t *testing.T) {
		w := NewWorld()
		provinceID := w.CreateEntity("Province")
		cityID := w.CreateEntity("City")
		genericID := w.CreateEntity("Generic")

		exists, err := w.HasTag(provinceID, "Province")
		if err != nil {
			t.Fatalf("HasTag failed: %v", err)
		}
		if !exists {
			t.Fatal("Province entity should have 'Province' tag")
		}
		exists, err = w.HasTag(cityID, "City")
		if err != nil {
			t.Fatalf("HasTag failed: %v", err)
		}
		if !exists {
			t.Fatal("City entity should have 'City' tag")
		}
		exists, err = w.HasTag(genericID, "Generic")
		if err != nil {
			t.Fatalf("HasTag failed: %v", err)
		}
		if !exists {
			t.Fatal("Generic entity should have 'Generic' tag")
		}

		w.DestroyEntity(provinceID, false)
		w.DestroyEntity(cityID, false)
		w.DestroyEntity(genericID, false)
		w.Cleanup()
	})

	t.Run("CreateEntity with unknown template", func(t *testing.T) {
		w := NewWorld()
		id := w.CreateEntity("UnknownTemplate")
		if !w.HasEntity(id) {
			t.Fatal("Entity with unknown template should still be created")
		}
		w.DestroyEntity(id, false)
		w.Cleanup()
	})

	t.Run("CreateEntity with empty template", func(t *testing.T) {
		w := NewWorld()
		id := w.CreateEntity("")
		if !w.HasEntity(id) {
			t.Fatal("Entity with empty template should still be created")
		}
		w.DestroyEntity(id, false)
		w.Cleanup()
	})

	t.Run("Entity heirarchy management", func(t *testing.T) {
		w := NewWorld()
		parentID := w.CreateEntity("Generic")
		childID := w.CreateEntity("Generic")

		w.SetParent(childID, parentID)
		parent, ok := w.ParentOf(childID)
		if !ok || parent != parentID {
			t.Fatal("ParentOf returned incorrect parent")
		}

		children := w.ChildrenOf(parentID)
		found := slices.Contains(children, childID)
		if !found {
			t.Fatal("ChildrenOf did not return the correct child")
		}

		w.DestroyEntity(parentID, true) // cascade destroy
		w.Cleanup()                     // Ensure cleanup is called to remove destroyed entities. Entities are still associated when destroyed and only removed after cleanup.

		if w.HasEntity(childID) {
			t.Fatal("Child entity should be destroyed when parent is cascade destroyed")
		}
		if w.HasEntity(parentID) {
			t.Fatal("Parent entity should be destroyed when cascade destroyed")
		}

		// Ensure that the child is no longer listed under the parent's children
		children = w.ChildrenOf(parentID)
		found = slices.Contains(children, childID)
		if found {
			t.Fatal("ChildrenOf should not return destroyed child")
		}

	})
}

func TestWorld_TagAndTemplateQueries(t *testing.T) {
	t.Run("QueryByTag returns correct entities", func(t *testing.T) {
		w := NewWorld()
		provinceID := w.CreateEntity("Province")
		cityID := w.CreateEntity("City")
		genericID := w.CreateEntity("Generic")

		provinces := w.QueryByTag("Province")
		if len(provinces) != 1 || provinces[0] != provinceID {
			t.Fatalf("expected to find 1 Province, got %d", len(provinces))
		}

		cities := w.QueryByTag("City")
		if len(cities) != 1 || cities[0] != cityID {
			t.Fatalf("expected to find 1 City, got %d", len(cities))
		}

		generics := w.QueryByTag("Generic")
		if len(generics) != 1 || generics[0] != genericID {
			t.Fatalf("expected to find 1 Generic, got %d", len(generics))
		}

		w.DestroyEntity(provinceID, false)
		w.DestroyEntity(cityID, false)
		w.DestroyEntity(genericID, false)
		w.Cleanup()
	})

	t.Run("QueryByTemplate returns correct entities", func(t *testing.T) {
		w := NewWorld()
		provinceID := w.CreateEntity("Province")
		cityID := w.CreateEntity("City")
		genericID := w.CreateEntity("Generic")

		provinces := w.QueryByTemplate("Province")
		if len(provinces) != 1 || provinces[0] != provinceID {
			t.Fatalf("expected to find 1 Province, got %d", len(provinces))
		}

		cities := w.QueryByTemplate("City")
		if len(cities) != 1 || cities[0] != cityID {
			t.Fatalf("expected to find 1 City, got %d", len(cities))
		}

		generics := w.QueryByTemplate("Generic")
		if len(generics) != 1 || generics[0] != genericID {
			t.Fatalf("expected to find 1 Generic, got %d", len(generics))
		}

		w.DestroyEntity(provinceID, false)
		w.DestroyEntity(cityID, false)
		w.DestroyEntity(genericID, false)
		w.Cleanup()
	})

	t.Run("QueryByTag returns empty slice for non-existent tag", func(t *testing.T) {
		w := NewWorld()
		entities := w.QueryByTag("NonExistentTag")
		if len(entities) != 0 {
			t.Fatalf("expected to find 0 entities, got %d", len(entities))
		}
		w.Cleanup()
	})

	t.Run("QueryByTemplate returns empty slice for non-existent template", func(t *testing.T) {
		w := NewWorld()
		entities := w.QueryByTemplate("NonExistentTemplate")
		if len(entities) != 0 {
			t.Fatalf("expected to find 0 entities, got %d", len(entities))
		}
		w.Cleanup()
	})
}

// TestWorld_CountAndReset tests entity counting and world reset.
func TestWorld_CountAndReset(t *testing.T) {
	w := NewWorld()

	if w.Count() != 0 {
		t.Fatalf("expected 0 entities in new world, got %d", w.Count())
	}

	id1 := w.CreateEntity("Generic")
	w.AddComponent(id1, testComponentA{value: 1})
	w.AddTag(id1, "test")

	id2 := w.CreateEntity("Generic")
	w.CreateEntity("Generic") // third entity for testing Count

	if w.Count() != 3 {
		t.Fatalf("expected 3 entities, got %d", w.Count())
	}

	// Set up a hierarchy
	w.SetParent(id2, id1)

	// Verify query still works
	generics := w.QueryByTemplate("Generic")
	if len(generics) != 3 {
		t.Fatalf("expected 3 generic entities, got %d", len(generics))
	}

	// Reset the world
	w.Reset()

	if w.Count() != 0 {
		t.Fatalf("expected 0 entities after reset, got %d", w.Count())
	}

	generics = w.QueryByTemplate("Generic")
	if len(generics) != 0 {
		t.Fatalf("expected no entities after reset, got %d", len(generics))
	}

	// Should be able to create again
	newID := w.CreateEntity("Generic")
	if !w.HasEntity(newID) {
		t.Fatal("should be able to create new entity after reset")
	}
	if w.Count() != 1 {
		t.Fatalf("expected 1 entity after new create, got %d", w.Count())
	}
}

// TestWorld_ComponentErrors tests error handling for non-existent entities.
func TestWorld_ComponentErrors(t *testing.T) {
	w := NewWorld()
	nonExistent := EntityID(999)

	// Test AddComponent with non-existent entity
	if err := w.AddComponent(nonExistent, testComponentA{}); err == nil {
		t.Fatal("expected error adding component to non-existent entity")
	}

	// Test GetComponent with non-existent entity
	if _, err := w.GetComponent(nonExistent, reflect.TypeOf(testComponentA{})); err == nil {
		t.Fatal("expected error getting component from non-existent entity")
	}

	// Test RemoveComponent with non-existent entity
	if err := w.RemoveComponent(nonExistent, reflect.TypeOf(testComponentA{})); err == nil {
		t.Fatal("expected error removing component from non-existent entity")
	}

	// Test AddTag with non-existent entity
	if err := w.AddTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error adding tag to non-existent entity")
	}

	// Test HasTag with non-existent entity
	if _, err := w.HasTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error checking tag on non-existent entity")
	}

	// Test RemoveTag with non-existent entity
	if err := w.RemoveTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error removing tag from non-existent entity")
	}
}

func TestWorldArchetypeIntegration(t *testing.T) {
	t.Run("EntityCreationAddsToEmptyArchetype", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")

		// Entity should be in empty archetype
		entity, exists := world.GetEntity(id)
		if !exists {
			t.Fatal("Entity should exist")
		}

		if entity.archetypeID == 0 {
			t.Error("Entity should have archetype ID")
		}

		// Check that empty archetype exists
		arch, exists := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if !exists {
			t.Fatal("Empty archetype should exist")
		}

		if len(arch.componentTypes) != 0 {
			t.Error("Empty archetype should have no component types")
		}

		if len(arch.entities) != 1 {
			t.Errorf("Empty archetype should have 1 entity, got %d", len(arch.entities))
		}
	})

	t.Run("AddComponentMigratesArchetype", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")

		// Get initial archetype
		entity, _ := world.GetEntity(id)
		initialArchetypeID := entity.archetypeID

		// Add component
		world.AddComponent(id, TestPosition{X: 1, Y: 2})

		// Entity should have migrated to new archetype
		entity, _ = world.GetEntity(id)
		if entity.archetypeID == initialArchetypeID {
			t.Error("Entity should have migrated to new archetype")
		}

		// Check new archetype
		arch, exists := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if !exists {
			t.Fatal("New archetype should exist")
		}

		if len(arch.componentTypes) != 1 {
			t.Errorf("New archetype should have 1 component type, got %d", len(arch.componentTypes))
		}
	})

	t.Run("QueryReturnsEntitiesFromArchetype", func(t *testing.T) {
		world := NewWorld()

		// Create entities with Position
		id1 := world.CreateEntity("Generic")
		id2 := world.CreateEntity("Generic")
		id3 := world.CreateEntity("Generic")

		world.AddComponent(id1, TestPosition{X: 1, Y: 2})
		world.AddComponent(id2, TestPosition{X: 3, Y: 4})
		world.AddComponent(id3, TestPosition{X: 5, Y: 6})

		// Query for Position
		posType := reflect.TypeFor[TestPosition]()
		result := world.Query(posType)

		if len(result) != 3 {
			t.Errorf("Expected 3 entities with Position, got %d", len(result))
		}

		// Check that all IDs are present
		idSet := make(map[EntityID]bool)
		for _, id := range result {
			idSet[id] = true
		}

		if !idSet[id1] || !idSet[id2] || !idSet[id3] {
			t.Error("Query should return all entities with Position")
		}
	})

	t.Run("QueryMultipleComponents", func(t *testing.T) {
		world := NewWorld()

		// Create entities with different component combinations
		id1 := world.CreateEntity("Generic") // Position only
		id2 := world.CreateEntity("Generic") // Position + Velocity
		id3 := world.CreateEntity("Generic") // Position + Velocity
		id4 := world.CreateEntity("Generic") // Velocity only

		world.AddComponent(id1, TestPosition{X: 1, Y: 2})
		world.AddComponent(id2, TestPosition{X: 3, Y: 4})
		world.AddComponent(id2, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(id3, TestPosition{X: 5, Y: 6})
		world.AddComponent(id3, TestVelocity{X: 0.2, Y: 0.2})
		world.AddComponent(id4, TestVelocity{X: 0.3, Y: 0.3})

		// Query for Position + Velocity
		posType := reflect.TypeFor[TestPosition]()
		velType := reflect.TypeFor[TestVelocity]()
		result := world.Query(posType, velType)

		if len(result) != 2 {
			t.Errorf("Expected 2 entities with Position+Velocity, got %d", len(result))
		}

		// Should only include id2 and id3
		idSet := make(map[EntityID]bool)
		for _, id := range result {
			idSet[id] = true
		}

		if !idSet[id2] || !idSet[id3] {
			t.Error("Query should return entities with both Position and Velocity")
		}

		if idSet[id1] || idSet[id4] {
			t.Error("Query should not return entities missing either component")
		}
	})

	t.Run("QuerySuperset", func(t *testing.T) {
		world := NewWorld()

		// Create entities with different component combinations
		id1 := world.CreateEntity("Generic") // Position only
		id2 := world.CreateEntity("Generic") // Position + Velocity
		id3 := world.CreateEntity("Generic") // Position + Velocity + Health

		world.AddComponent(id1, TestPosition{X: 1, Y: 2})
		world.AddComponent(id2, TestPosition{X: 3, Y: 4})
		world.AddComponent(id2, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(id3, TestPosition{X: 5, Y: 6})
		world.AddComponent(id3, TestVelocity{X: 0.2, Y: 0.2})
		world.AddComponent(id3, TestHealth{Value: 100})

		// Query for Position (superset - entities with Position AND possibly more)
		posType := reflect.TypeFor[TestPosition]()
		result := world.QuerySuperset(posType)

		if len(result) != 3 {
			t.Errorf("Expected 3 entities with at least Position, got %d", len(result))
		}

		// Should include all three entities
		idSet := make(map[EntityID]bool)
		for _, id := range result {
			idSet[id] = true
		}

		if !idSet[id1] || !idSet[id2] || !idSet[id3] {
			t.Error("QuerySuperset should return all entities with Position")
		}
	})

	t.Run("GetComponentFromArchetype", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")
		pos := TestPosition{X: 10, Y: 20}
		world.AddComponent(id, pos)

		// Get component back
		retrieved, err := world.GetComponent(id, reflect.TypeFor[TestPosition]())
		if err != nil {
			t.Fatalf("Failed to get component: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Retrieved component should not be nil")
		}

		// Type assertion and comparison
		retrievedPos, ok := retrieved.(TestPosition)
		if !ok {
			t.Fatal("Retrieved component should be TestPosition")
		}

		if retrievedPos.X != pos.X || retrievedPos.Y != pos.Y {
			t.Errorf("Retrieved component values don't match: got (%f,%f), want (%f,%f)",
				retrievedPos.X, retrievedPos.Y, pos.X, pos.Y)
		}
	})

	t.Run("HasComponent", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")

		// Entity should not have Position initially
		if world.HasComponent(id, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position initially")
		}

		// Add Position
		world.AddComponent(id, TestPosition{X: 1, Y: 2})

		// Entity should now have Position
		if !world.HasComponent(id, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should have Position after adding it")
		}

		// Entity should not have Velocity
		if world.HasComponent(id, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should not have Velocity")
		}
	})

	t.Run("RemoveComponent", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")
		world.AddComponent(id, TestPosition{X: 1, Y: 2})
		world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})

		// Entity should have both components
		if !world.HasComponent(id, reflect.TypeFor[TestPosition]()) ||
			!world.HasComponent(id, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should have both components")
		}

		// Remove Position
		world.RemoveComponent(id, reflect.TypeFor[TestPosition]())

		// Entity should no longer have Position
		if world.HasComponent(id, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position after removal")
		}

		// Entity should still have Velocity
		if !world.HasComponent(id, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should still have Velocity")
		}
	})

	t.Run("ComponentsList", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")
		world.AddComponent(id, TestPosition{X: 1, Y: 2})
		world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(id, TestHealth{Value: 100})

		comps := world.Components(id)

		if len(comps) != 3 {
			t.Errorf("Expected 3 components, got %d", len(comps))
		}

		// Check that we have all three component types
		foundPosition := false
		foundVelocity := false
		foundHealth := false

		for _, comp := range comps {
			switch comp.Name() {
			case "TestPosition":
				foundPosition = true
			case "TestVelocity":
				foundVelocity = true
			case "TestHealth":
				foundHealth = true
			}
		}

		if !foundPosition || !foundVelocity || !foundHealth {
			t.Error("Not all component types found in Components list")
		}
	})

	t.Run("EntityDestructionRemovesFromArchetype", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")
		world.AddComponent(id, TestPosition{X: 1, Y: 2})

		// Get archetype info before destruction
		entity, _ := world.GetEntity(id)
		arch, _ := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		initialCount := len(arch.entities)

		// Destroy entity
		world.DestroyEntity(id, false)
		world.Cleanup()

		// Archetype should have one less entity
		arch, _ = world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if arch != nil && len(arch.entities) != initialCount-1 {
			t.Errorf("Archetype should have %d entities after destruction, got %d",
				initialCount-1, len(arch.entities))
		}
	})

	t.Run("MultipleEntitiesInSameArchetype", func(t *testing.T) {
		world := NewWorld()

		// Create multiple entities with same components
		ids := make([]EntityID, 100)
		for i := range ids {
			ids[i] = world.CreateEntity("Generic")
			world.AddComponent(ids[i], TestPosition{X: float64(i), Y: float64(i)})
			world.AddComponent(ids[i], TestVelocity{X: 0.1, Y: 0.1})
		}

		// All should be in the same archetype
		arch, _ := world.archetypeManager.GetArchetypeByKeyHash(
			NewArchetypeKey(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]()).Hash())

		if arch == nil {
			t.Fatal("Archetype should exist")
		}

		if len(arch.entities) != 100 {
			t.Errorf("Expected 100 entities in archetype, got %d", len(arch.entities))
		}

		// Query should return all entities
		result := world.Query(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]())
		if len(result) != 100 {
			t.Errorf("Query should return 100 entities, got %d", len(result))
		}
	})

	t.Run("ArchetypeMigrationOnComponentChange", func(t *testing.T) {
		world := NewWorld()

		id := world.CreateEntity("Generic")

		// Add Position - should be in [Position] archetype
		world.AddComponent(id, TestPosition{X: 1, Y: 2})
		entity, _ := world.GetEntity(id)
		posArchetypeID := entity.archetypeID

		// Add Velocity - should migrate to [Position, Velocity] archetype
		world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
		entity, _ = world.GetEntity(id)
		posVelArchetypeID := entity.archetypeID

		if posArchetypeID == posVelArchetypeID {
			t.Error("Entity should have migrated to new archetype when adding Velocity")
		}

		// Remove Position - should migrate to [Velocity] archetype
		world.RemoveComponent(id, reflect.TypeFor[TestPosition]())
		entity, _ = world.GetEntity(id)
		velArchetypeID := entity.archetypeID

		if velArchetypeID == posVelArchetypeID {
			t.Error("Entity should have migrated to new archetype when removing Position")
		}

		// Verify final state
		if world.HasComponent(id, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position")
		}
		if !world.HasComponent(id, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should still have Velocity")
		}
	})
}
