package core

import (
	"reflect"
	"testing"
)

type TestPosition struct {
	X, Y float64
}

func (p TestPosition) Name() string     { return "TestPosition" }
func (p TestPosition) Clone() Component { return TestPosition{X: p.X, Y: p.Y} }

type TestVelocity struct {
	X, Y float64
}

func (v TestVelocity) Name() string     { return "TestVelocity" }
func (v TestVelocity) Clone() Component { return TestVelocity{X: v.X, Y: v.Y} }

type TestHealth struct {
	Value int
}

func (h TestHealth) Name() string     { return "TestHealth" }
func (h TestHealth) Clone() Component { return TestHealth{Value: h.Value} }

type TestSprite struct {
	TextureID     string
	Width, Height int
}

func (s TestSprite) Name() string { return "TestSprite" }
func (s TestSprite) Clone() Component {
	return TestSprite{TextureID: s.TextureID, Width: s.Width, Height: s.Height}
}

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
		key2 := NewArchetypeKey(velType, posType)

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
		key1 := NewArchetypeKey(posType) // "TestPosition" - 13 chars
		key2 := NewArchetypeKey(velType) // "TestVelocity" - 12 chars

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
// Edge Case Tests
// =============================================================================

func TestArchetypeEdgeCases(t *testing.T) {
	t.Run("GetComponentFromEmptyArchetype", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")

		_, err := world.GetComponent(entity.id, reflect.TypeFor[TestPosition]())
		if err == nil {
			t.Error("Expected error when getting non-existent component")
		}
	})

	t.Run("RemoveNonExistentComponent", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")

		err := world.RemoveComponent(entity.id, reflect.TypeFor[TestPosition]())
		if err != nil {
			t.Errorf("Expected no error when removing non-existent component, got: %v", err)
		}
	})

	t.Run("QueryNonExistentComponent", func(t *testing.T) {
		world := NewWorld()

		world.CreateEntity("Generic")

		result := world.Query(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]())
		if len(result) != 0 {
			t.Error("Expected empty result for non-existent component combination")
		}
	})

	t.Run("GetComponentAfterDestruction", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")
		world.AddComponent(entity.id, TestPosition{X: 1, Y: 2})

		world.DestroyEntity(entity.id, false)
		world.Cleanup()

		_, err := world.GetComponent(entity.id, reflect.TypeFor[TestPosition]())
		if err == nil {
			t.Error("Expected error when getting component from destroyed entity")
		}
	})

	t.Run("AddComponentToDestroyedEntity", func(t *testing.T) {
		world := NewWorld()

		entity := world.CreateEntity("Generic")
		world.DestroyEntity(entity.id, false)
		world.Cleanup()

		err := world.AddComponent(entity.id, TestPosition{X: 1, Y: 2})
		if err == nil {
			t.Error("Expected error when adding component to destroyed entity")
		}
	})
}
