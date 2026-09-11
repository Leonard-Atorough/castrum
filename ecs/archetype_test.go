package ecs

import (
	"reflect"
	"testing"
)

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

func TestArchetype_RemoveEntity(t *testing.T) {
	posType := reflect.TypeFor[TestPosition]()
	arch := NewArchetype(1, NewArchetypeKey(posType))
	arch.entities = []EntityID{10, 20, 30}
	// Store typed slice directly instead of []Component
	arch.componentData[posType] = []TestPosition{
		{X: 1}, {X: 2}, {X: 3},
	}

	t.Run("removing a middle slot swaps in the last entity and keeps component data aligned", func(t *testing.T) {
		movedID, moved := arch.removeEntity(1) // remove entity 20
		if !moved || movedID != 30 {
			t.Errorf("expected entity 30 to be moved into slot 1, got movedID=%d moved=%v", movedID, moved)
		}
		if !reflect.DeepEqual(arch.entities, []EntityID{10, 30}) {
			t.Errorf("unexpected entities after removal: %#v", arch.entities)
		}

		// Verify component data was swapped correctly using reflection
		rawSlice := arch.componentData[posType]
		sliceVal := reflect.ValueOf(rawSlice)
		if sliceVal.Kind() != reflect.Slice || sliceVal.Len() != 2 {
			t.Errorf("expected slice of length 2, got kind=%v, len=%d", sliceVal.Kind(), sliceVal.Len())
		}
		
		// Check first element (should be X:1)
		first := sliceVal.Index(0).Interface().(TestPosition)
		if first.X != 1 {
			t.Errorf("expected first element X=1, got X=%f", first.X)
		}
		
		// Check second element (should be X:3, swapped from last position)
		second := sliceVal.Index(1).Interface().(TestPosition)
		if second.X != 3 {
			t.Errorf("expected second element X=3, got X=%f", second.X)
		}
	})

	t.Run("removing the last slot needs no swap", func(t *testing.T) {
		movedID, moved := arch.removeEntity(1)
		if moved {
			t.Errorf("removing the last slot should report no move, got movedID=%d", movedID)
		}
		if !reflect.DeepEqual(arch.entities, []EntityID{10}) {
			t.Errorf("unexpected entities: %#v", arch.entities)
		}
	})

	t.Run("out-of-range index is a no-op", func(t *testing.T) {
		_, moved := arch.removeEntity(5)
		if moved {
			t.Error("out-of-range removal should report no move")
		}
	})
}

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

		arch1 := manager.GetOrCreateArchetype(posType)
		if arch1.ID != 1 {
			t.Errorf("Expected first archetype ID 1, got %d", arch1.ID)
		}

		arch2 := manager.GetOrCreateArchetype(posType, velType)
		if arch2.ID != 2 {
			t.Errorf("Expected second archetype ID 2, got %d", arch2.ID)
		}

		arch3 := manager.GetOrCreateArchetype(posType, velType, healthType)
		if arch3.ID != 3 {
			t.Errorf("Expected third archetype ID 3, got %d", arch3.ID)
		}

		arch1Again := manager.GetOrCreateArchetype(posType)
		if arch1Again.ID != arch1.ID {
			t.Error("GetOrCreate should return same archetype for same types")
		}

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
