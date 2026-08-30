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
		provinceEntity := w.CreateEntity("Province")
		cityEntity := w.CreateEntity("City")
		genericEntity1 := w.CreateEntity("Generic")
		genericEntity2 := w.CreateEntity("Generic")

		provinces := w.QueryByTemplate("Province")
		if len(provinces) != 1 || provinces[0] != provinceEntity.ID {
			t.Fatal("QueryByTemplate for 'Province' returned incorrect entities")
		}

		cities := w.QueryByTemplate("City")
		if len(cities) != 1 || cities[0] != cityEntity.ID {
			t.Fatal("QueryByTemplate for 'City' returned incorrect entities")
		}

		generics := w.QueryByTemplate("Generic")
		if len(generics) != 2 || (generics[0] != genericEntity1.ID && generics[0] != genericEntity2.ID) || (generics[1] != genericEntity1.ID && generics[1] != genericEntity2.ID) {
			t.Fatal("QueryByTemplate for 'Generic' returned incorrect entities")
		}
	})

}

func TestWorld_EntityLifecycle(t *testing.T) {
	t.Run("Create, Get, and Destroy Entity", func(t *testing.T) {
		w := NewWorld()
		entity := w.CreateEntity("Generic")
		if !w.HasEntity(entity.ID) {
			t.Fatal("Created entity should exist")
		}
		if _, exists := w.GetEntity(entity.ID); !exists {
			t.Fatal("GetEntity should return the entity after creation")
		}
		if exists := w.HasEntity(entity.ID); !exists {
			t.Fatal("Entity should exist before destruction")
		}
		if err := w.DestroyEntity(entity.ID, false); err != nil {
			t.Fatalf("DestroyEntity failed: %v", err)
		}
		w.Cleanup()
		if w.HasEntity(entity.ID) {
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

	t.Run("CreateEntity with unknown template", func(t *testing.T) {
		w := NewWorld()
		entity := w.CreateEntity("UnknownTemplate")
		if !w.HasEntity(entity.ID) {
			t.Fatal("Entity with unknown template should still be created")
		}
		w.DestroyEntity(entity.ID, false)
		w.Cleanup()
	})

	t.Run("CreateEntity with empty template", func(t *testing.T) {
		w := NewWorld()
		entity := w.CreateEntity("")
		if !w.HasEntity(entity.ID) {
			t.Fatal("Entity with empty template should still be created")
		}
		w.DestroyEntity(entity.ID, false)
		w.Cleanup()
	})

	t.Run("Entity heirarchy management", func(t *testing.T) {
		w := NewWorld()
		parent := w.CreateEntity("Generic")
		child := w.CreateEntity("Generic")

		w.SetParent(child.ID, parent.ID)
		parentID, ok := w.ParentOf(child.ID)
		if !ok || parentID != parent.ID {
			t.Fatal("ParentOf returned incorrect parent")
		}

		children := w.ChildrenOf(parent.ID)
		found := slices.Contains(children, child.ID)
		if !found {
			t.Fatal("ChildrenOf did not return the correct child")
		}

		w.DestroyEntity(parent.ID, true) // cascade destroy
		w.Cleanup()                      // Ensure cleanup is called to remove destroyed entities. Entities are still associated when destroyed and only removed after cleanup.

		if w.HasEntity(child.ID) {
			t.Fatal("Child entity should be destroyed when parent is cascade destroyed")
		}
		if w.HasEntity(parent.ID) {
			t.Fatal("Parent entity should be destroyed when cascade destroyed")
		}

		// Ensure that the child is no longer listed under the parent's children
		children = w.ChildrenOf(parent.ID)
		found = slices.Contains(children, child.ID)
		if found {
			t.Fatal("ChildrenOf should not return destroyed child")
		}

	})
}

func TestWorld_TagAndTemplateQueries(t *testing.T) {
	t.Run("QueryByTag returns correct entities", func(t *testing.T) {
		w := NewWorld()
		provinceEntity := w.CreateEntity("Province")
		cityEntity := w.CreateEntity("City")
		genericEntity := w.CreateEntity("Generic")

		provinces := w.QueryByTag("Province")
		if len(provinces) != 1 || provinces[0] != provinceEntity.ID {
			t.Fatalf("expected to find 1 Province, got %d", len(provinces))
		}

		cities := w.QueryByTag("City")
		if len(cities) != 1 || cities[0] != cityEntity.ID {
			t.Fatalf("expected to find 1 City, got %d", len(cities))
		}

		generics := w.QueryByTag("Generic")
		if len(generics) != 1 || generics[0] != genericEntity.ID {
			t.Fatalf("expected to find 1 Generic, got %d", len(generics))
		}

		w.DestroyEntity(provinceEntity.ID, false)
		w.DestroyEntity(cityEntity.ID, false)
		w.DestroyEntity(genericEntity.ID, false)
		w.Cleanup()
	})

	t.Run("QueryByTemplate returns correct entities", func(t *testing.T) {
		w := NewWorld()
		provinceEntity := w.CreateEntity("Province")
		cityEntity := w.CreateEntity("City")
		genericEntity := w.CreateEntity("Generic")

		provinces := w.QueryByTemplate("Province")
		if len(provinces) != 1 || provinces[0] != provinceEntity.ID {
			t.Fatalf("expected to find 1 Province, got %d", len(provinces))
		}

		cities := w.QueryByTemplate("City")
		if len(cities) != 1 || cities[0] != cityEntity.ID {
			t.Fatalf("expected to find 1 City, got %d", len(cities))
		}

		generics := w.QueryByTemplate("Generic")
		if len(generics) != 1 || generics[0] != genericEntity.ID {
			t.Fatalf("expected to find 1 Generic, got %d", len(generics))
		}

		w.DestroyEntity(provinceEntity.ID, false)
		w.DestroyEntity(cityEntity.ID, false)
		w.DestroyEntity(genericEntity.ID, false)
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

	entity1 := w.CreateEntity("Generic")
	w.AddComponent(entity1.ID, testComponentA{value: 1})

	entity2 := w.CreateEntity("Generic")
	w.CreateEntity("Generic") // third entity for testing Count

	if w.Count() != 3 {
		t.Fatalf("expected 3 entities, got %d", w.Count())
	}

	// Set up a hierarchy
	w.SetParent(entity2.ID, entity1.ID)

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
	newEntity := w.CreateEntity("Generic")
	if !w.HasEntity(newEntity.ID) {
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
}

func TestWorldArchetypeIntegration(t *testing.T) {
	t.Run("EntityCreationAddsToEmptyArchetype", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")

		// Entity should be in empty archetype
		gotEntity, exists := world.GetEntity(entity.ID)
		if !exists {
			t.Fatal("Entity should exist")
		}

		if gotEntity.archetypeID == 0 {
			t.Error("Entity should have archetype ID")
		}

		// Check that empty archetype exists
		arch, exists := world.archetypeManager.GetArchetypeByID(gotEntity.archetypeID)
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

		entity := world.CreateEntity("Generic")

		// Get initial archetype
		gotEntity, _ := world.GetEntity(entity.ID)
		initialArchetypeID := gotEntity.archetypeID

		// Add component
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})

		// Entity should have migrated to new archetype
		gotEntity, _ = world.GetEntity(entity.ID)
		if gotEntity.archetypeID == initialArchetypeID {
			t.Error("Entity should have migrated to new archetype")
		}

		// Check new archetype
		arch, exists := world.archetypeManager.GetArchetypeByID(gotEntity.archetypeID)
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
		entity1 := world.CreateEntity("Generic")
		entity2 := world.CreateEntity("Generic")
		entity3 := world.CreateEntity("Generic")

		world.AddComponent(entity1.ID, TestPosition{X: 1, Y: 2})
		world.AddComponent(entity2.ID, TestPosition{X: 3, Y: 4})
		world.AddComponent(entity3.ID, TestPosition{X: 5, Y: 6})

		// Query for Position
		posType := reflect.TypeFor[TestPosition]()
		result := world.Query(posType)

		if len(result) != 3 {
			t.Errorf("Expected 3 entities with Position, got %d", len(result))
		}

		// Check that all IDs are present
		idSet := make(map[EntityID]bool)
		for _, ID := range result {
			idSet[ID] = true
		}

		if !idSet[entity1.ID] || !idSet[entity2.ID] || !idSet[entity3.ID] {
			t.Error("Query should return all entities with Position")
		}
	})

	t.Run("QueryMultipleComponents", func(t *testing.T) {
		world := NewWorld()

		// Create entities with different component combinations
		entity1 := world.CreateEntity("Generic") // Position only
		entity2 := world.CreateEntity("Generic") // Position + Velocity
		entity3 := world.CreateEntity("Generic") // Position + Velocity
		entity4 := world.CreateEntity("Generic") // Velocity only

		world.AddComponent(entity1.ID, TestPosition{X: 1, Y: 2})
		world.AddComponent(entity2.ID, TestPosition{X: 3, Y: 4})
		world.AddComponent(entity2.ID, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(entity3.ID, TestPosition{X: 5, Y: 6})
		world.AddComponent(entity3.ID, TestVelocity{X: 0.2, Y: 0.2})
		world.AddComponent(entity4.ID, TestVelocity{X: 0.3, Y: 0.3})

		// Query for Position + Velocity
		posType := reflect.TypeFor[TestPosition]()
		velType := reflect.TypeFor[TestVelocity]()
		result := world.Query(posType, velType)

		if len(result) != 2 {
			t.Errorf("Expected 2 entities with Position+Velocity, got %d", len(result))
		}

		// Should only include entity2 and entity3
		idSet := make(map[EntityID]bool)
		for _, ID := range result {
			idSet[ID] = true
		}

		if !idSet[entity2.ID] || !idSet[entity3.ID] {
			t.Error("Query should return entities with both Position and Velocity")
		}

		if idSet[entity1.ID] || idSet[entity4.ID] {
			t.Error("Query should not return entities missing either component")
		}
	})

	t.Run("QuerySuperset", func(t *testing.T) {
		world := NewWorld()

		// Create entities with different component combinations
		entity1 := world.CreateEntity("Generic") // Position only
		entity2 := world.CreateEntity("Generic") // Position + Velocity
		entity3 := world.CreateEntity("Generic") // Position + Velocity + Health

		world.AddComponent(entity1.ID, TestPosition{X: 1, Y: 2})
		world.AddComponent(entity2.ID, TestPosition{X: 3, Y: 4})
		world.AddComponent(entity2.ID, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(entity3.ID, TestPosition{X: 5, Y: 6})
		world.AddComponent(entity3.ID, TestVelocity{X: 0.2, Y: 0.2})
		world.AddComponent(entity3.ID, TestHealth{Value: 100})

		// Query for Position (superset - entities with Position AND possibly more)
		posType := reflect.TypeFor[TestPosition]()
		result := world.QuerySuperset(posType)

		if len(result) != 3 {
			t.Errorf("Expected 3 entities with at least Position, got %d", len(result))
		}

		// Should include all three entities
		idSet := make(map[EntityID]bool)
		for _, ID := range result {
			idSet[ID] = true
		}

		if !idSet[entity1.ID] || !idSet[entity2.ID] || !idSet[entity3.ID] {
			t.Error("QuerySuperset should return all entities with Position")
		}
	})

	t.Run("GetComponentFromArchetype", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")
		pos := TestPosition{X: 10, Y: 20}
		world.AddComponent(entity.ID, pos)

		// Get component back
		retrieved, err := world.GetComponent(entity.ID, reflect.TypeFor[TestPosition]())
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

		entity := world.CreateEntity("Generic")

		// Entity should not have Position initially
		if world.HasComponent(entity.ID, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position initially")
		}

		// Add Position
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})

		// Entity should now have Position
		if !world.HasComponent(entity.ID, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should have Position after adding it")
		}

		// Entity should not have Velocity
		if world.HasComponent(entity.ID, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should not have Velocity")
		}
	})

	t.Run("RemoveComponent", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})
		world.AddComponent(entity.ID, TestVelocity{X: 0.1, Y: 0.1})

		// Entity should have both components
		if !world.HasComponent(entity.ID, reflect.TypeFor[TestPosition]()) ||
			!world.HasComponent(entity.ID, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should have both components")
		}

		// Remove Position
		world.RemoveComponent(entity.ID, reflect.TypeFor[TestPosition]())

		// Entity should no longer have Position
		if world.HasComponent(entity.ID, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position after removal")
		}

		// Entity should still have Velocity
		if !world.HasComponent(entity.ID, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should still have Velocity")
		}
	})

	t.Run("ComponentsList", func(t *testing.T) {
		world := NewWorld()
		RegisterMultiple(
			reflect.TypeFor[TestPosition](),
			reflect.TypeFor[TestVelocity](),
			reflect.TypeFor[TestHealth](),
		)

		entity := world.CreateEntity("Generic")
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})
		world.AddComponent(entity.ID, TestVelocity{X: 0.1, Y: 0.1})
		world.AddComponent(entity.ID, TestHealth{Value: 100})

		comps := world.Components(entity.ID)

		if len(comps) != 3 {
			t.Errorf("Expected 3 components, got %d", len(comps))
		}

		// Check that we have all three component types
		foundPosition := false
		foundVelocity := false
		foundHealth := false

		for _, comp := range comps {
			typ := reflect.TypeOf(comp)
			switch GetTypeInfo(typ).Name {
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

		entity := world.CreateEntity("Generic")
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})

		// Get archetype info before destruction
		entity, _ = world.GetEntity(entity.ID)
		arch, _ := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		initialCount := len(arch.entities)

		// Destroy entity
		world.DestroyEntity(entity.ID, false)
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
		entities := make([]*Entity, 100)
		for i := range entities {
			entities[i] = world.CreateEntity("Generic")
			world.AddComponent(entities[i].ID, TestPosition{X: float64(i), Y: float64(i)})
			world.AddComponent(entities[i].ID, TestVelocity{X: 0.1, Y: 0.1})
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

		entity := world.CreateEntity("Generic")

		// Add Position - should be in [Position] archetype
		world.AddComponent(entity.ID, TestPosition{X: 1, Y: 2})
		entity, _ = world.GetEntity(entity.ID)
		posArchetypeID := entity.archetypeID

		// Add Velocity - should migrate to [Position, Velocity] archetype
		world.AddComponent(entity.ID, TestVelocity{X: 0.1, Y: 0.1})
		entity, _ = world.GetEntity(entity.ID)
		posVelArchetypeID := entity.archetypeID

		if posArchetypeID == posVelArchetypeID {
			t.Error("Entity should have migrated to new archetype when adding Velocity")
		}

		// Remove Position - should migrate to [Velocity] archetype
		world.RemoveComponent(entity.ID, reflect.TypeFor[TestPosition]())
		entity, _ = world.GetEntity(entity.ID)
		velArchetypeID := entity.archetypeID

		if velArchetypeID == posVelArchetypeID {
			t.Error("Entity should have migrated to new archetype when removing Position")
		}

		// Verify final state
		if world.HasComponent(entity.ID, reflect.TypeFor[TestPosition]()) {
			t.Error("Entity should not have Position")
		}
		if !world.HasComponent(entity.ID, reflect.TypeFor[TestVelocity]()) {
			t.Error("Entity should still have Velocity")
		}
	})
}
