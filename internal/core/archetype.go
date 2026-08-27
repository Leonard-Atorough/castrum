package core

import (
	"reflect"
	"sort"

	"github.com/leonard-atorough/castrum/ecs"
)

type ArchetypeKey []reflect.Type

type ArchetypeKeyHash uint64

func NewArchetypeKey(components ...reflect.Type) ArchetypeKey {
	if len(components) == 0 {
		return nil
	}

	//sort for consistent ordering
	sorted := make([]reflect.Type, len(components))
	copy(sorted, components)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name() < sorted[j].Name()
	})
	return sorted
}

func (ak ArchetypeKey) Hash() ArchetypeKeyHash {
	if len(ak) == 0 {
		return 0
	}

	// Use a better hash that includes the full type name to avoid collisions
	var h ArchetypeKeyHash
	for _, t := range ak {
		// Combine type name length and actual name content
		h = h*31 + ArchetypeKeyHash(len(t.Name()))
		// Add hash of the type name string itself
		for _, c := range t.Name() {
			h = h*31 + ArchetypeKeyHash(c)
		}
		// Add the type's package path for additional uniqueness
		if t.PkgPath() != "" {
			for _, c := range t.PkgPath() {
				h = h*31 + ArchetypeKeyHash(c)
			}
		}
	}
	return h
}

// ContainsAll checks if the current ArchetypeKey contains all types from another ArchetypeKey.
func (ak ArchetypeKey) ContainsAll(other ArchetypeKey) bool {
	if len(other) == 0 {
		return true
	}

	if len(ak) < len(other) {
		return false
	}

	// Create a map for quick lookup of types in the current key
	typeSet := make(map[reflect.Type]struct{}, len(ak))
	for _, t := range ak {
		typeSet[t] = struct{}{}
	}

	for _, t := range other {
		if _, exists := typeSet[t]; !exists {
			return false
		}
	}
	return true
}

type Archetype struct {
	ID             uint64
	componentTypes ArchetypeKey
	entities       []ecs.EntityID

	componentData map[reflect.Type]any // this will replace the componentStore, storing components by type for this archetype
}

func NewArchetype(id uint64, componentTypes ArchetypeKey) *Archetype {
	return &Archetype{
		ID:             id,
		componentTypes: componentTypes,
		entities:       make([]ecs.EntityID, 0, 1024), // corresponds to 1024 entities per archetype by default which, if each entity is 16 bytes, would be 16KB per archetype, which is a reasonable default
		componentData:  make(map[reflect.Type]any),
	}
}

type ArchetypeManager struct {
	archetypes map[uint64]*Archetype
	keyToID    map[ArchetypeKeyHash]uint64 //can't use ArchetypeKey as a map key directly, so we hash it
	nextID     uint64
}

func NewArchetypeManager() *ArchetypeManager {
	return &ArchetypeManager{
		archetypes: make(map[uint64]*Archetype),
		keyToID:    make(map[ArchetypeKeyHash]uint64),
		nextID:     1, // start IDs from 1
	}
}

func (am *ArchetypeManager) GetOrCreateArchetype(componentTypes ...reflect.Type) *Archetype {
	key := NewArchetypeKey(componentTypes...)
	hash := ArchetypeKeyHash(key.Hash())

	if archetypeID, exists := am.keyToID[hash]; exists {
		return am.archetypes[archetypeID]
	}

	newArchetype := NewArchetype(am.nextID, key)
	am.archetypes[am.nextID] = newArchetype
	am.keyToID[hash] = am.nextID
	am.nextID++

	return newArchetype
}

func (am *ArchetypeManager) GetArchetypeByID(id uint64) (*Archetype, bool) {
	archetype, exists := am.archetypes[id]
	return archetype, exists
}

func (am *ArchetypeManager) GetArchetypeByKeyHash(hash ArchetypeKeyHash) (*Archetype, bool) {
	if archetypeID, exists := am.keyToID[hash]; exists {
		return am.archetypes[archetypeID], true
	}
	return nil, false
}
