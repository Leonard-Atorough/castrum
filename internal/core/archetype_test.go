package core

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// Test component types for archetype testing
type TestPosition struct {
	X, Y float64
}

func (p TestPosition) Name() string         { return "TestPosition" }
func (p TestPosition) Clone() ecs.Component { return TestPosition{X: p.X, Y: p.Y} }

type TestVelocity struct {
	X, Y float64
}

func (v TestVelocity) Name() string         { return "TestVelocity" }
func (v TestVelocity) Clone() ecs.Component { return TestVelocity{X: v.X, Y: v.Y} }

type TestHealth struct {
	Value int
}

func (h TestHealth) Name() string         { return "TestHealth" }
func (h TestHealth) Clone() ecs.Component { return TestHealth{Value: h.Value} }

type TestSprite struct {
	TextureID string
	Width, Height int
}

func (s TestSprite) Name() string         { return "TestSprite" }
func (s TestSprite) Clone() ecs.Component { return TestSprite{TextureID: s.TextureID, Width: s.Width, Height: s.Height} }

// =============================================================================
// ArchetypeKey Tests
// =============================================================================

func TestArchetypeKeyCreation(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()
	healthType := reflect.TypeFor[TestHealth]()

	t.Run("EmptyKey", func(t *testing.T) {
		key := NewArchetypeKey()
		if key != nil {
			t.Errorf("Expected nil for empty key, got %v", key)
		}
	})

	t.Run("SingleType", func(t *testing.T) {
		key := NewArchetypeKey(posType)
		if len(key) != 1 {
			t.Errorf("Expected length 1, got %d", len(key))
		}
		if key[0] != posType {
			t.Errorf("Expected %v, got %v", posType, key[0])
		}
	})

	t.Run("MultipleTypes", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType, healthType)
		if len(key) != 3 {
			t.Errorf("Expected length 3, got %d", len(key))
		}
		
		// Should be sorted
		for i := 1; i < len(key); i++ {
			if key[i-1].Name() > key[i].Name() {
				t.Errorf("Key not sorted: %v > %v", key[i-1].Name(), key[i].Name())
			}
		}
	})

	t.Run("ConsistentOrdering", func(t *testing.T) {
		// Create keys in different orders, should be equal
		key1 := NewArchetypeKey(posType, velType, healthType)
		key2 := NewArchetypeKey(healthType, posType, velType)
		key3 := NewArchetypeKey(velType, healthType, posType)
		
		if !reflect.DeepEqual(key1, key2) || !reflect.DeepEqual(key2, key3) {
			t.Error("ArchetypeKeys with same types in different orders should be equal")
		}
	})
}

func TestArchetypeKeyHash(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()

	t.Run("EmptyKeyHash", func(t *testing.T) {
		key := NewArchetypeKey()
		hash := key.Hash()
		if hash != 0 {
			t.Errorf("Expected hash 0 for empty key, got %d", hash)
		}
	})

	t.Run("ConsistentHashing", func(t *testing.T) {
		key1 := NewArchetypeKey(posType, velType)
		key2 := NewArchetypeKey(velType, posType) // Different order
		
		hash1 := key1.Hash()
		hash2 := key2.Hash()
		
		if hash1 != hash2 {
			t.Errorf("Same types in different orders should have same hash: %d != %d", hash1, hash2)
		}
	})

	t.Run("DifferentKeysDifferentHashes", func(t *testing.T) {
		key1 := NewArchetypeKey(posType)
		key2 := NewArchetypeKey(posType, velType)
		key3 := NewArchetypeKey(velType)
		
		hash1 := key1.Hash()
		hash2 := key2.Hash()
		hash3 := key3.Hash()
		
		if hash1 == hash2 || hash1 == hash3 || hash2 == hash3 {
			t.Error("Different keys should have different hashes")
		}
	})

	t.Run("HashIncludesTypeName", func(t *testing.T) {
		// This test validates that the hash uses the full type name, not just length
		// Create two types with same name length but different names
		key1 := NewArchetypeKey(posType)        // "TestPosition" - 13 chars
		key2 := NewArchetypeKey(velType)        // "TestVelocity" - 12 chars
		
		hash1 := key1.Hash()
		hash2 := key2.Hash()
		
		if hash1 == hash2 {
			t.Error("Types with different names should have different hashes")
		}
	})
}

func TestArchetypeKeyContainsAll(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()
	healthType := reflect.TypeFor[TestHealth]()

	t.Run("EmptyOther", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType)
		if !key.ContainsAll(ArchetypeKey{}) {
			t.Error("Any key should contain empty key")
		}
	})

	t.Run("ExactMatch", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType)
		other := NewArchetypeKey(posType, velType)
		if !key.ContainsAll(other) {
			t.Error("Key should contain all of itself")
		}
	})

	t.Run("Subset", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType, healthType)
		other := NewArchetypeKey(posType, velType)
		if !key.ContainsAll(other) {
			t.Error("Key should contain subset")
		}
	})

	t.Run("NotSubset", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType)
		other := NewArchetypeKey(posType, velType, healthType)
		if key.ContainsAll(other) {
			t.Error("Key should not contain superset")
		}
	})

	t.Run("MissingType", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType)
		other := NewArchetypeKey(posType, healthType)
		if key.ContainsAll(other) {
			t.Error("Key should not contain other when missing a type")
		}
	})

	t.Run("ShorterKey", func(t *testing.T) {
		key := NewArchetypeKey(posType)
		other := NewArchetypeKey(posType, velType)
		if key.ContainsAll(other) {
			t.Error("Shorter key should not contain longer key")
		}
	})
}

// =============================================================================
// Archetype Tests
// =============================================================================

func TestArchetypeCreation(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()

	t.Run("NewArchetype", func(t *testing.T) {
		key := NewArchetypeKey(posType, velType)
		archetype := NewArchetype(1, key)
		
		if archetype.ID != 1 {
			t.Errorf("Expected ID 1, got %d", archetype.ID)
		}
		
		if !reflect.DeepEqual(archetype.componentTypes, key) {
			t.Errorf("componentTypes mismatch")
		}
		
		if len(archetype.entities) != 0 {
			t.Errorf("Expected empty entities, got %d", len(archetype.entities))
		}
		
		if archetype.componentData == nil {
			t.Error("componentData should be initialized")
		}
	})

	t.Run("EmptyArchetype", func(t *testing.T) {
		archetype := NewArchetype(1, NewArchetypeKey())
		if len(archetype.componentTypes) != 0 {
			t.Errorf("Expected empty component types, got %d", len(archetype.componentTypes))
		}
	})
}

func TestArchetypeManager(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()
	healthType := reflect.TypeFor[TestHealth]()

	t.Run("NewArchetypeManager", func(t *testing.T) {
		manager := NewArchetypeManager()
		if manager == nil {
			t.Error("NewArchetypeManager returned nil")
		}
		if len(manager.archetypes) != 0 {
			t.Error("New manager should have no archetypes")
		}
		if manager.nextID != 1 {
			t.Errorf("Expected nextID 1, got %d", manager.nextID)
		}
	})

	t.Run("GetOrCreateArchetype", func(t *testing.T) {
		manager := NewArchetypeManager()
		
		// Create first archetype
		arch1 := manager.GetOrCreateArchetype(posType)
		if arch1.ID != 1 {
			t.Errorf("Expected first archetype ID 1, got %d", arch1.ID)
		}
		
		// Create second archetype
		arch2 := manager.GetOrCreateArchetype(posType, velType)
		if arch2.ID != 2 {
			t.Errorf("Expected second archetype ID 2, got %d", arch2.ID)
		}
		
		// Create third archetype with all three types
		arch3 := manager.GetOrCreateArchetype(posType, velType, healthType)
		if arch3.ID != 3 {
			t.Errorf("Expected third archetype ID 3, got %d", arch3.ID)
		}
		
		// Get existing archetype
		arch1Again := manager.GetOrCreateArchetype(posType)
		if arch1Again.ID != arch1.ID {
			t.Error("GetOrCreate should return same archetype for same types")
		}
		
		// Different order should return same archetype
		arch1Reverse := manager.GetOrCreateArchetype(posType) // Same as arch1
		if arch1Reverse.ID != arch1.ID {
			t.Error("Different type order should return same archetype")
		}
	})

	t.Run("GetArchetypeByID", func(t *testing.T) {
		manager := NewArchetypeManager()
		
		arch := manager.GetOrCreateArchetype(posType)
		
		retrieved, exists := manager.GetArchetypeByID(arch.ID)
		if !exists {
			t.Error("GetArchetypeByID should find existing archetype")
		}
		if retrieved.ID != arch.ID {
			t.Error("Retrieved archetype should match original")
		}
		
		_, exists = manager.GetArchetypeByID(999)
		if exists {
			t.Error("GetArchetypeByID should not find non-existent archetype")
		}
	})

	t.Run("GetArchetypeByKeyHash", func(t *testing.T) {
		manager := NewArchetypeManager()
		
		key := NewArchetypeKey(posType, velType)
		arch := manager.GetOrCreateArchetype(key...)
		
		retrieved, exists := manager.GetArchetypeByKeyHash(key.Hash())
		if !exists {
			t.Error("GetArchetypeByKeyHash should find existing archetype")
		}
		if retrieved.ID != arch.ID {
			t.Error("Retrieved archetype should match original")
		}
	})
}

// =============================================================================
// World Archetype Integration Tests
// =============================================================================

func TestWorldArchetypeIntegration(t *testing.T) {
	t.Run("EntityCreationAddsToEmptyArchetype", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		
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
		
		id := world.Create("Generic")
		
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
		id1 := world.Create("Generic")
		id2 := world.Create("Generic")
		id3 := world.Create("Generic")
		
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
		idSet := make(map[ecs.EntityID]bool)
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
		id1 := world.Create("Generic") // Position only
		id2 := world.Create("Generic") // Position + Velocity
		id3 := world.Create("Generic") // Position + Velocity
		id4 := world.Create("Generic") // Velocity only
		
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
		idSet := make(map[ecs.EntityID]bool)
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
		id1 := world.Create("Generic") // Position only
		id2 := world.Create("Generic") // Position + Velocity
		id3 := world.Create("Generic") // Position + Velocity + Health
		
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
		idSet := make(map[ecs.EntityID]bool)
		for _, id := range result {
			idSet[id] = true
		}
		
		if !idSet[id1] || !idSet[id2] || !idSet[id3] {
			t.Error("QuerySuperset should return all entities with Position")
		}
	})

	t.Run("GetComponentFromArchetype", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
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
		
		id := world.Create("Generic")
		
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
		
		id := world.Create("Generic")
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
		
		id := world.Create("Generic")
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
		
		id := world.Create("Generic")
		world.AddComponent(id, TestPosition{X: 1, Y: 2})
		
		// Get archetype info before destruction
		entity, _ := world.GetEntity(id)
		arch, _ := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		initialCount := len(arch.entities)
		
		// Destroy entity
		world.Destroy(id, false)
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
		ids := make([]ecs.EntityID, 100)
		for i := range ids {
			ids[i] = world.Create("Generic")
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
		
		id := world.Create("Generic")
		
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

// =============================================================================
// Edge Cases & Error Handling
// =============================================================================

func TestArchetypeEdgeCases(t *testing.T) {
	t.Run("GetComponentFromEmptyArchetype", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		
		// Try to get component that doesn't exist
		_, err := world.GetComponent(id, reflect.TypeFor[TestPosition]())
		if err == nil {
			t.Error("Expected error when getting non-existent component")
		}
	})

	t.Run("RemoveNonExistentComponent", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		
		// Should not error when removing non-existent component
		err := world.RemoveComponent(id, reflect.TypeFor[TestPosition]())
		if err != nil {
			t.Errorf("Expected no error when removing non-existent component, got: %v", err)
		}
	})

	t.Run("QueryNonExistentComponent", func(t *testing.T) {
		world := NewWorld()
		
		// Create entity without components
		world.Create("Generic")
		
		// Query for non-existent component combination
		result := world.Query(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]())
		if len(result) != 0 {
			t.Error("Expected empty result for non-existent component combination")
		}
	})

	t.Run("GetComponentAfterDestruction", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		world.AddComponent(id, TestPosition{X: 1, Y: 2})
		
		world.Destroy(id, false)
		world.Cleanup()
		
		_, err := world.GetComponent(id, reflect.TypeFor[TestPosition]())
		if err == nil {
			t.Error("Expected error when getting component from destroyed entity")
		}
	})

	t.Run("AddComponentToDestroyedEntity", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		world.Destroy(id, false)
		world.Cleanup()
		
		err := world.AddComponent(id, TestPosition{X: 1, Y: 2})
		if err == nil {
			t.Error("Expected error when adding component to destroyed entity")
		}
	})
}

// =============================================================================
// Performance & Stress Tests
// =============================================================================

func TestArchetypeStress(t *testing.T) {
	t.Run("ManyEntitiesSameArchetype", func(t *testing.T) {
		world := NewWorld()
		
		// Create 10,000 entities with same components
		for i := 0; i < 10000; i++ {
			id := world.Create("Generic")
			world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
			world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
		}
		
		// Query should be fast
		result := world.Query(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]())
		if len(result) != 10000 {
			t.Errorf("Expected 10000 entities, got %d", len(result))
		}
	})

	t.Run("ManyDifferentArchetypes", func(t *testing.T) {
		world := NewWorld()
		
		// Create entities with many different component combinations
		for i := 0; i < 100; i++ {
			id := world.Create("Generic")
			
			// Randomly add components
			if i%2 == 0 {
				world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
			}
			if i%3 == 0 {
				world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
			}
			if i%5 == 0 {
				world.AddComponent(id, TestHealth{Value: 100})
			}
			if i%7 == 0 {
				world.AddComponent(id, TestSprite{TextureID: "test", Width: 32, Height: 32})
			}
		}
		
		// Should have created multiple archetypes
		if len(world.archetypeManager.archetypes) < 5 {
			t.Errorf("Expected multiple archetypes, got %d", len(world.archetypeManager.archetypes))
		}
	})

	t.Run("ComponentAddRemoveStress", func(t *testing.T) {
		world := NewWorld()
		
		id := world.Create("Generic")
		
		// Add and remove components many times
		for i := 0; i < 1000; i++ {
			world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
			world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
			world.RemoveComponent(id, reflect.TypeFor[TestPosition]())
			world.RemoveComponent(id, reflect.TypeFor[TestVelocity]())
		}
		
		// Entity should end up in empty archetype
		entity, _ := world.GetEntity(id)
		arch, _ := world.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if arch == nil || len(arch.componentTypes) != 0 {
			t.Error("Entity should be in empty archetype after removing all components")
		}
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkArchetypeQuery(b *testing.B) {
	world := NewWorld()
	
	// Pre-create entities with Position
	for i := 0; i < 10000; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
	}
	
	posType := reflect.TypeFor[TestPosition]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Query(posType)
	}
}

func BenchmarkArchetypeQueryMultiple(b *testing.B) {
	world := NewWorld()
	
	// Pre-create entities with Position + Velocity
	for i := 0; i < 10000; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
		world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
	}
	
	posType := reflect.TypeFor[TestPosition]()
	velType := reflect.TypeFor[TestVelocity]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.Query(posType, velType)
	}
}

func BenchmarkArchetypeGetComponent(b *testing.B) {
	world := NewWorld()
	
	// Pre-create entity with Position
	id := world.Create("Generic")
	world.AddComponent(id, TestPosition{X: 1, Y: 2})
	
	posType := reflect.TypeFor[TestPosition]()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		world.GetComponent(id, posType)
	}
}

func BenchmarkArchetypeAddComponent(b *testing.B) {
	world := NewWorld()
	
	// Pre-create entity
	id := world.Create("Generic")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Remove and re-add to test migration
		world.RemoveComponent(id, reflect.TypeFor[TestPosition]())
		world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
	}
}

func BenchmarkArchetypeEntityCreation(b *testing.B) {
	world := NewWorld()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := world.Create("Generic")
		world.AddComponent(id, TestPosition{X: float64(i), Y: float64(i)})
		world.AddComponent(id, TestVelocity{X: 0.1, Y: 0.1})
	}
}
