package ecs

import (
	"reflect"
	"testing"
)

func TestArchetypeKey_NewArchetypeKey(t *testing.T) {
	t.Run("nil for empty input", func(t *testing.T) {
		key := NewArchetypeKey()
		if key != nil {
			t.Errorf("expected nil for empty input, got %v", key)
		}
	})

	t.Run("single component type", func(t *testing.T) {
		aType := reflect.TypeFor[TestPosition]()
		key := NewArchetypeKey(aType)
		if len(key) != 1 {
			t.Errorf("expected length 1, got %d", len(key))
		}
		if key[0] != aType {
			t.Errorf("expected type %v, got %v", aType, key[0])
		}
	})

	t.Run("multiple component types sorted", func(t *testing.T) {
		aType := reflect.TypeFor[TestHealth]()
		bType := reflect.TypeFor[TestPosition]()
		cType := reflect.TypeFor[TestVelocity]()

		// Pass in unsorted order
		key := NewArchetypeKey(cType, aType, bType)

		if len(key) != 3 {
			t.Errorf("expected length 3, got %d", len(key))
		}

		// Verify sorted order (by type string)
		aStr := aType.String()
		bStr := bType.String()
		cStr := cType.String()

		if key[0].String() != aStr {
			t.Errorf("expected first type to be %s, got %s", aStr, key[0].String())
		}
		if key[1].String() != bStr {
			t.Errorf("expected second type to be %s, got %s", bStr, key[1].String())
		}
		if key[2].String() != cStr {
			t.Errorf("expected third type to be %s, got %s", cStr, key[2].String())
		}
	})

	t.Run("same types in different order produces same key", func(t *testing.T) {
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()

		key1 := NewArchetypeKey(aType, bType)
		key2 := NewArchetypeKey(bType, aType)

		if !key1.Equals(key2) {
			t.Errorf("expected same key for same types in different order")
		}
	})
}

func TestArchetypeKey_Len(t *testing.T) {
	tests := []struct {
		name     string
		key      ArchetypeKey
		expected int
	}{
		{"nil key", nil, 0},
		{"empty key", ArchetypeKey{}, 0},
		{"single component", NewArchetypeKey(reflect.TypeFor[TestPosition]()), 1},
		{"multiple components", NewArchetypeKey(
			reflect.TypeFor[TestPosition](),
			reflect.TypeFor[TestVelocity](),
			reflect.TypeFor[TestHealth](),
		), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.Len(); got != tt.expected {
				t.Errorf("Len() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestArchetypeKey_String(t *testing.T) {
	tests := []struct {
		name     string
		key      ArchetypeKey
		expected string
	}{
		{"nil key", nil, "[]"},
		{"empty key", ArchetypeKey{}, "[]"},
		{"single component", NewArchetypeKey(reflect.TypeFor[TestPosition]()), "[ecs.TestPosition]"},
		{"multiple components", NewArchetypeKey(
			reflect.TypeFor[TestHealth](),
			reflect.TypeFor[TestPosition](),
		), "[ecs.TestHealth, ecs.TestPosition]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.String(); got != tt.expected {
				t.Errorf("String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestArchetypeKey_Hash(t *testing.T) {
	t.Run("nil key returns 0", func(t *testing.T) {
		var nilKey ArchetypeKey
		if h := nilKey.Hash(); h != 0 {
			t.Errorf("expected hash 0 for nil key, got %d", h)
		}
	})

	t.Run("empty key returns 0", func(t *testing.T) {
		key := ArchetypeKey{}
		if h := key.Hash(); h != 0 {
			t.Errorf("expected hash 0 for empty key, got %d", h)
		}
	})

	t.Run("same types produce same hash", func(t *testing.T) {
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()

		key1 := NewArchetypeKey(aType, bType)
		key2 := NewArchetypeKey(bType, aType)

		if h1, h2 := key1.Hash(), key2.Hash(); h1 != h2 {
			t.Errorf("expected same hash for same types regardless of order, got %d and %d", h1, h2)
		}
	})

	t.Run("different types produce different hash", func(t *testing.T) {
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()
		cType := reflect.TypeFor[TestHealth]()

		key1 := NewArchetypeKey(aType, bType)
		key2 := NewArchetypeKey(aType, cType)

		if h1, h2 := key1.Hash(), key2.Hash(); h1 == h2 {
			t.Errorf("expected different hashes for different types, both got %d", h1)
		}
	})

	t.Run("same key produces same hash", func(t *testing.T) {
		key := NewArchetypeKey(reflect.TypeFor[TestPosition]())
		h1 := key.Hash()
		h2 := key.Hash()
		if h1 != h2 {
			t.Errorf("expected same hash for same key, got %d and %d", h1, h2)
		}
	})
}

func TestArchetypeKey_Equals(t *testing.T) {
	tests := []struct {
		name     string
		key1     ArchetypeKey
		key2     ArchetypeKey
		expected bool
	}{
		{"nil equals nil", nil, nil, true},
		{"empty equals empty", ArchetypeKey{}, ArchetypeKey{}, true},
		{"nil equals empty", nil, ArchetypeKey{}, true},
		{"empty equals nil", ArchetypeKey{}, nil, true},
		{"same single type",
			NewArchetypeKey(reflect.TypeFor[TestPosition]()),
			NewArchetypeKey(reflect.TypeFor[TestPosition]()),
			true},
		{"different single type",
			NewArchetypeKey(reflect.TypeFor[TestPosition]()),
			NewArchetypeKey(reflect.TypeFor[TestVelocity]()),
			false},
		{"same types different order",
			NewArchetypeKey(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]()),
			NewArchetypeKey(reflect.TypeFor[TestVelocity](), reflect.TypeFor[TestPosition]()),
			true},
		{"different lengths",
			NewArchetypeKey(reflect.TypeFor[TestPosition]()),
			NewArchetypeKey(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]()),
			false},
		{"different types same length",
			NewArchetypeKey(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestVelocity]()),
			NewArchetypeKey(reflect.TypeFor[TestPosition](), reflect.TypeFor[TestHealth]()),
			false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key1.Equals(tt.key2); got != tt.expected {
				t.Errorf("Equals() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestArchetypeKey_ContainsAll(t *testing.T) {
	aType := reflect.TypeFor[TestPosition]()
	bType := reflect.TypeFor[TestVelocity]()
	cType := reflect.TypeFor[TestHealth]()

	tests := []struct {
		name     string
		key      ArchetypeKey
		other    ArchetypeKey
		expected bool
	}{
		{"empty other always true",
			NewArchetypeKey(aType),
			ArchetypeKey{},
			true},
		{"nil other always true",
			NewArchetypeKey(aType),
			nil,
			true},
		{"contains all - same",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType, bType),
			true},
		{"contains all - subset",
			NewArchetypeKey(aType, bType, cType),
			NewArchetypeKey(aType, bType),
			true},
		{"does not contain all - missing one",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType, bType, cType),
			false},
		{"does not contain all - different types",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType, cType),
			false},
		{"empty key with non-empty other",
			ArchetypeKey{},
			NewArchetypeKey(aType),
			false},
		{"small archetype - contains all",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType),
			true},
		{"large archetype - contains all using map", func() ArchetypeKey {
			key := ArchetypeKey{}
			for i := 0; i < 10; i++ {
				key = append(key, aType)
			}
			return key
		}(), NewArchetypeKey(aType), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.ContainsAll(tt.other); got != tt.expected {
				t.Errorf("ContainsAll() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestArchetypeKey_ContainsAny(t *testing.T) {
	aType := reflect.TypeFor[TestPosition]()
	bType := reflect.TypeFor[TestVelocity]()
	cType := reflect.TypeFor[TestHealth]()

	tests := []struct {
		name     string
		key      ArchetypeKey
		other    ArchetypeKey
		expected bool
	}{
		{"empty other always false",
			NewArchetypeKey(aType),
			ArchetypeKey{},
			false},
		{"nil other always false",
			NewArchetypeKey(aType),
			nil,
			false},
		{"empty key always false",
			ArchetypeKey{},
			NewArchetypeKey(aType),
			false},
		{"contains any - match first",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType, cType),
			true},
		{"contains any - match second",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(cType, bType),
			true},
		{"does not contain any",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(cType),
			false},
		{"contains any - multiple matches",
			NewArchetypeKey(aType, bType, cType),
			NewArchetypeKey(aType, bType),
			true},
		{"small archetype - contains any",
			NewArchetypeKey(aType),
			NewArchetypeKey(aType, bType),
			true},
		{"large archetype - contains any using map", func() ArchetypeKey {
			key := ArchetypeKey{}
			for i := 0; i < 10; i++ {
				key = append(key, aType)
			}
			return key
		}(), NewArchetypeKey(aType), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.ContainsAny(tt.other); got != tt.expected {
				t.Errorf("ContainsAny() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestArchetypeKey_ContainsExactly(t *testing.T) {
	aType := reflect.TypeFor[TestPosition]()
	bType := reflect.TypeFor[TestVelocity]()

	tests := []struct {
		name     string
		key      ArchetypeKey
		other    ArchetypeKey
		expected bool
	}{
		{"same types same order",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(aType, bType),
			true},
		{"same types different order",
			NewArchetypeKey(aType, bType),
			NewArchetypeKey(bType, aType),
			true},
		{"different types",
			NewArchetypeKey(aType),
			NewArchetypeKey(bType),
			false},
		{"different lengths",
			NewArchetypeKey(aType),
			NewArchetypeKey(aType, bType),
			false},
		{"nil equals nil", nil, nil, true},
		{"empty equals empty", ArchetypeKey{}, ArchetypeKey{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.ContainsExactly(tt.other); got != tt.expected {
				t.Errorf("ContainsExactly() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestArchetype_NewArchetype(t *testing.T) {
	key := NewArchetypeKey(reflect.TypeFor[TestPosition]())
	arch := NewArchetype(1, key)

	if arch.ID != 1 {
		t.Errorf("expected ID 1, got %d", arch.ID)
	}
	if !arch.componentTypes.Equals(key) {
		t.Errorf("expected componentTypes %v, got %v", key, arch.componentTypes)
	}
	if len(arch.entities) != 0 {
		t.Errorf("expected empty entities, got %d", len(arch.entities))
	}
	if len(arch.componentData) != 0 {
		t.Errorf("expected empty componentData, got %d", len(arch.componentData))
	}
}

func TestArchetype_Len(t *testing.T) {
	arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))

	if got := arch.Len(); got != 0 {
		t.Errorf("expected Len() 0 for new archetype, got %d", got)
	}

	arch.entities = append(arch.entities, EntityID(1))
	if got := arch.Len(); got != 1 {
		t.Errorf("expected Len() 1 after adding entity, got %d", got)
	}
}

func TestArchetype_Entities(t *testing.T) {
	arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))

	entities := arch.Entities()
	if len(entities) != 0 {
		t.Errorf("expected empty entities, got %d", len(entities))
	}

	arch.entities = []EntityID{1, 2, 3}
	entities = arch.Entities()
	if len(entities) != 3 {
		t.Errorf("expected 3 entities, got %d", len(entities))
	}
	if entities[0] != 1 || entities[1] != 2 || entities[2] != 3 {
		t.Errorf("unexpected entity IDs: %v", entities)
	}
}

func TestArchetype_ComponentsAtIndex(t *testing.T) {
	arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))

	t.Run("out of bounds - negative index", func(t *testing.T) {
		comps := arch.ComponentsAtIndex[TestPosition](-1)
		if len(comps) != 0 {
			t.Errorf("expected empty components for negative index, got %d", len(comps))
		}
	})

	t.Run("out of bounds - index >= length", func(t *testing.T) {
		comps := arch.ComponentsAtIndex[TestPosition](0)
		if len(comps) != 0 {
			t.Errorf("expected empty components for out of bounds index, got %d", len(comps))
		}
	})

	t.Run("component type not in archetype", func(t *testing.T) {
		arch.entities = append(arch.entities, EntityID(1))
		comps := arch.ComponentsAtIndex[TestVelocity](0)
		if len(comps) != 0 {
			t.Errorf("expected empty components for missing component type, got %d", len(comps))
		}
	})

	t.Run("valid component access", func(t *testing.T) {
		arch.entities = []EntityID{1, 2, 3}

		comps := []TestPosition{{X: 10.0, Y: 20}, {X: 30.0, Y: 40}, {X: 50.0, Y: 60}}
		arch.componentData[reflect.TypeFor[TestPosition]()] = comps

		result := arch.ComponentsAtIndex[TestPosition](1)
		if len(result) != 1 {
			t.Errorf("expected 1 component, got %d", len(result))
		}
		if len(result) > 0 && (result[0].X != 30.0 || result[0].Y != 40) {
			t.Errorf("expected value {X: 30.0, Y: 40}, got %+v", result[0])
		}
	})
}

func TestArchetype_RemoveEntity(t *testing.T) {
	t.Run("remove from empty archetype", func(t *testing.T) {
		arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))
		movedID, moved := arch.removeEntity(0)
		if moved || movedID != 0 {
			t.Errorf("expected no move for empty archetype, got moved=%v, movedID=%d", moved, movedID)
		}
	})

	t.Run("remove out of bounds", func(t *testing.T) {
		arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))
		arch.entities = []EntityID{1}

		movedID, moved := arch.removeEntity(-1)
		if moved || movedID != 0 {
			t.Errorf("expected no move for negative index, got moved=%v, movedID=%d", moved, movedID)
		}

		movedID, moved = arch.removeEntity(5)
		if moved || movedID != 0 {
			t.Errorf("expected no move for out of bounds index, got moved=%v, movedID=%d", moved, movedID)
		}
	})

	t.Run("remove last entity", func(t *testing.T) {
		arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))
		arch.entities = []EntityID{1, 2, 3}

		comps := []TestPosition{{X: 10.0, Y: 20}, {X: 30.0, Y: 40}, {X: 50.0, Y: 60}}
		arch.componentData[reflect.TypeFor[TestPosition]()] = comps

		movedID, moved := arch.removeEntity(2)
		if moved {
			t.Errorf("expected no move when removing last entity, got moved=%v", moved)
		}
		if movedID != 0 {
			t.Errorf("expected movedID 0 when removing last entity, got %d", movedID)
		}
		if len(arch.entities) != 2 {
			t.Errorf("expected 2 entities after removal, got %d", len(arch.entities))
		}
		if arch.entities[0] != 1 || arch.entities[1] != 2 {
			t.Errorf("unexpected entities: %v", arch.entities)
		}

		if compSlice, ok := arch.componentData[reflect.TypeFor[TestPosition]()].([]TestPosition); ok {
			if len(compSlice) != 2 {
				t.Errorf("expected component data length 2, got %d", len(compSlice))
			}
		}
	})

	t.Run("remove middle entity - swap with last", func(t *testing.T) {
		arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))
		arch.entities = []EntityID{1, 2, 3}

		comps := []TestPosition{{X: 10.0, Y: 20}, {X: 30.0, Y: 40}, {X: 50.0, Y: 60}}
		arch.componentData[reflect.TypeFor[TestPosition]()] = comps

		movedID, moved := arch.removeEntity(1)
		if !moved {
			t.Error("expected move when removing middle entity")
		}
		if movedID != 3 {
			t.Errorf("expected movedID 3, got %d", movedID)
		}
		if len(arch.entities) != 2 {
			t.Errorf("expected 2 entities after removal, got %d", len(arch.entities))
		}
		if arch.entities[0] != 1 || arch.entities[1] != 3 {
			t.Errorf("unexpected entities: %v", arch.entities)
		}

		if compSlice, ok := arch.componentData[reflect.TypeFor[TestPosition]()].([]TestPosition); ok {
			if len(compSlice) != 2 {
				t.Errorf("expected component data length 2, got %d", len(compSlice))
			}
			if compSlice[1].X != 50.0 || compSlice[1].Y != 60 {
				t.Errorf("expected component value {X: 50.0, Y: 60} at index 1, got %+v", compSlice[1])
			}
		}
	})

	t.Run("remove first entity - swap with last", func(t *testing.T) {
		arch := NewArchetype(1, NewArchetypeKey(reflect.TypeFor[TestPosition]()))
		arch.entities = []EntityID{1, 2, 3}

		comps := []TestPosition{{X: 10.0, Y: 20}, {X: 30.0, Y: 40}, {X: 50.0, Y: 60}}
		arch.componentData[reflect.TypeFor[TestPosition]()] = comps

		movedID, moved := arch.removeEntity(0)
		if !moved {
			t.Error("expected move when removing first entity")
		}
		if movedID != 3 {
			t.Errorf("expected movedID 3, got %d", movedID)
		}
		if len(arch.entities) != 2 {
			t.Errorf("expected 2 entities after removal, got %d", len(arch.entities))
		}
		if arch.entities[0] != 3 || arch.entities[1] != 2 {
			t.Errorf("unexpected entities: %v", arch.entities)
		}

		if compSlice, ok := arch.componentData[reflect.TypeFor[TestPosition]()].([]TestPosition); ok {
			if compSlice[0].X != 50.0 || compSlice[0].Y != 60 {
				t.Errorf("expected component value {X: 50.0, Y: 60} at index 0, got %+v", compSlice[0])
			}
		}
	})
}

func TestArchetypeManager_NewArchetypeManager(t *testing.T) {
	am := NewArchetypeManager()

	if len(am.archetypes) != 0 {
		t.Errorf("expected empty archetypes map, got %d entries", len(am.archetypes))
	}
	if len(am.keyToID) != 0 {
		t.Errorf("expected empty keyToID map, got %d entries", len(am.keyToID))
	}
	if am.nextID != 1 {
		t.Errorf("expected nextID 1, got %d", am.nextID)
	}
}

func TestArchetypeManager_GetOrCreateArchetype(t *testing.T) {
	t.Run("create new archetype", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()

		arch := am.GetOrCreateArchetype(aType)

		if arch == nil {
			t.Fatal("expected non-nil archetype")
		}
		if arch.ID != 1 {
			t.Errorf("expected ID 1, got %d", arch.ID)
		}
		if !arch.componentTypes.Equals(NewArchetypeKey(aType)) {
			t.Errorf("unexpected component types: %v", arch.componentTypes)
		}
		if am.nextID != 2 {
			t.Errorf("expected nextID 2, got %d", am.nextID)
		}
	})

	t.Run("get existing archetype by type", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()

		arch1 := am.GetOrCreateArchetype(aType)
		arch2 := am.GetOrCreateArchetype(aType)

		if arch1 != arch2 {
			t.Error("expected same archetype instance for same types")
		}
		if am.nextID != 2 {
			t.Errorf("expected nextID still 2 (no new archetype created), got %d", am.nextID)
		}
	})

	t.Run("different types create different archetypes", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()

		archA := am.GetOrCreateArchetype(aType)
		archB := am.GetOrCreateArchetype(bType)

		if archA == archB {
			t.Error("expected different archetype instances for different types")
		}
		if archA.ID != 1 {
			t.Errorf("expected archA ID 1, got %d", archA.ID)
		}
		if archB.ID != 2 {
			t.Errorf("expected archB ID 2, got %d", archB.ID)
		}
	})

	t.Run("same types different order create same archetype", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()

		arch1 := am.GetOrCreateArchetype(aType, bType)
		arch2 := am.GetOrCreateArchetype(bType, aType)

		if arch1 != arch2 {
			t.Error("expected same archetype for same types in different order")
		}
	})

	t.Run("multiple component types", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		bType := reflect.TypeFor[TestVelocity]()
		cType := reflect.TypeOf(TestSprite{})

		arch := am.GetOrCreateArchetype(aType, bType, cType)

		if arch == nil {
			t.Fatal("expected non-nil archetype")
		}
		expectedKey := NewArchetypeKey(aType, bType, cType)
		if !arch.componentTypes.Equals(expectedKey) {
			t.Errorf("unexpected component types: %v", arch.componentTypes)
		}
	})
}

func TestArchetypeManager_GetArchetypeByID(t *testing.T) {
	t.Run("get existing archetype", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		arch := am.GetOrCreateArchetype(aType)

		result, exists := am.GetArchetypeByID(arch.ID)
		if !exists {
			t.Error("expected archetype to exist")
		}
		if result != arch {
			t.Error("expected same archetype instance")
		}
	})

	t.Run("get non-existing archetype", func(t *testing.T) {
		am := NewArchetypeManager()

		_, exists := am.GetArchetypeByID(999)
		if exists {
			t.Error("expected archetype to not exist")
		}
	})
}

func TestArchetypeManager_GetArchetypeByKeyHash(t *testing.T) {
	t.Run("get existing archetype by key hash", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		arch := am.GetOrCreateArchetype(aType)

		hash := arch.componentTypes.Hash()
		result, exists := am.GetArchetypeByKeyHash(hash)
		if !exists {
			t.Error("expected archetype to exist")
		}
		if result != arch {
			t.Error("expected same archetype instance")
		}
	})

	t.Run("get non-existing archetype by key hash", func(t *testing.T) {
		am := NewArchetypeManager()

		key := NewArchetypeKey(reflect.TypeFor[TestPosition]())
		hash := key.Hash()

		_, exists := am.GetArchetypeByKeyHash(hash)
		if exists {
			t.Error("expected archetype to not exist")
		}
	})
}

func TestArchetypeManager_CleanupEmptyArchetypes(t *testing.T) {
	t.Run("cleanup removes empty archetypes", func(t *testing.T) {
		am := NewArchetypeManager()

		aType := reflect.TypeFor[TestPosition]()
		arch := am.GetOrCreateArchetype(aType)

		_, exists := am.GetArchetypeByID(arch.ID)
		if !exists {
			t.Fatal("expected archetype to exist before cleanup")
		}

		am.CleanupEmptyArchetypes()

		_, exists = am.GetArchetypeByID(arch.ID)
		if exists {
			t.Error("expected empty archetype to be removed after cleanup")
		}
	})

	t.Run("cleanup preserves non-empty archetypes", func(t *testing.T) {
		am := NewArchetypeManager()

		aType := reflect.TypeFor[TestPosition]()
		arch := am.GetOrCreateArchetype(aType)
		arch.entities = append(arch.entities, EntityID(1))

		am.CleanupEmptyArchetypes()

		_, exists := am.GetArchetypeByID(arch.ID)
		if !exists {
			t.Error("expected non-empty archetype to be preserved after cleanup")
		}
	})

	t.Run("cleanup removes only empty archetypes", func(t *testing.T) {
		am := NewArchetypeManager()

		aType := reflect.TypeFor[TestPosition]()
		emptyArch := am.GetOrCreateArchetype(aType)

		bType := reflect.TypeFor[TestVelocity]()
		nonEmptyArch := am.GetOrCreateArchetype(bType)
		nonEmptyArch.entities = append(nonEmptyArch.entities, EntityID(1))

		am.CleanupEmptyArchetypes()

		_, emptyExists := am.GetArchetypeByID(emptyArch.ID)
		if emptyExists {
			t.Error("expected empty archetype to be removed")
		}

		_, nonEmptyExists := am.GetArchetypeByID(nonEmptyArch.ID)
		if !nonEmptyExists {
			t.Error("expected non-empty archetype to be preserved")
		}
	})
}

func TestArchetypeManager_Concurrency(t *testing.T) {
	t.Run("concurrent GetOrCreateArchetype", func(t *testing.T) {
		am := NewArchetypeManager()

		done := make(chan bool, 10)
		for range 10 {
			go func() {
				am.GetOrCreateArchetype(reflect.TypeFor[TestPosition]())
				done <- true
			}()
		}

		for range 10 {
			<-done
		}

		if len(am.archetypes) != 1 {
			t.Errorf("expected 1 archetype after concurrent calls, got %d", len(am.archetypes))
		}
	})

	t.Run("concurrent read and write", func(t *testing.T) {
		am := NewArchetypeManager()
		aType := reflect.TypeFor[TestPosition]()
		arch := am.GetOrCreateArchetype(aType)

		done := make(chan bool, 20)

		for range 10 {
			go func() {
				am.GetOrCreateArchetype(aType)
				done <- true
			}()
		}

		for range 10 {
			go func() {
				am.GetArchetypeByID(arch.ID)
				done <- true
			}()
		}

		for range 20 {
			<-done
		}

		if len(am.archetypes) != 1 {
			t.Errorf("expected 1 archetype after concurrent operations, got %d", len(am.archetypes))
		}
	})
}
