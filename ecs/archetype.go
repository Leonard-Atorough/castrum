package ecs

import (
	"reflect"
	"sort"
	"sync"
)

// FNV-1a hash constants for 64-bit
const (
	fnvOffsetBasis64 = 14695981039346656037
	fnvPrime64       = 1099511628211
)

// ArchetypeKey represents a sorted list of component types that define an archetype.
type ArchetypeKey []reflect.Type

// ArchetypeKeyHash represents the hash of an ArchetypeKey.
type ArchetypeKeyHash uint64

// NewArchetypeKey creates a new ArchetypeKey from the given component types.
// The returned key is sorted to ensure consistent ordering. If no component types
// are provided, it returns nil.
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

// Hash computes the FNV-1a hash of the ArchetypeKey using the full type string
// (including package path) for each component type. This ensures unique hashes even
// for types with the same name in different packages, preventing hash collisions.
// Returns a 64-bit hash value that uniquely identifies the combination of component types.
func (ak ArchetypeKey) Hash() ArchetypeKeyHash {
	if len(ak) == 0 {
		return 0
	}

	var h ArchetypeKeyHash = fnvOffsetBasis64
	for _, t := range ak {
		// Use t.String() to get the fully qualified type name (e.g., "package.Name")
		typeStr := t.String()
		for i := 0; i < len(typeStr); i++ {
			h ^= ArchetypeKeyHash(typeStr[i])
			h *= fnvPrime64
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

func (ak ArchetypeKey) ContainsAny(other ArchetypeKey) bool {
	if len(other) == 0 {
		return false
	}

	if len(ak) == 0 {
		return false
	}

	typeSet := make(map[reflect.Type]struct{}, len(ak))
	for _, t := range ak {
		typeSet[t] = struct{}{}
	}

	for _, t := range other {
		if _, exists := typeSet[t]; exists {
			return true
		}
	}
	return false
}

type Archetype struct {
	ID             uint64
	componentTypes ArchetypeKey
	entities       []EntityID

	componentData map[reflect.Type]any // could switch to a byte slice for more efficient storage
}

func NewArchetype(id uint64, componentTypes ArchetypeKey) *Archetype {
	return &Archetype{
		ID:             id,
		componentTypes: componentTypes,
		entities:       make([]EntityID, 0, 16),
		componentData:  make(map[reflect.Type]any),
	}
}

func (a *Archetype) Entities() []EntityID {
	return a.entities
}

func (a *Archetype) Components[T Component](entityID EntityID) []T {
	var comps []T
	for _, compType := range a.componentTypes {
		if slice, exists := a.componentData[compType]; exists {
			compSlice := slice.([]Component)
			for _, comp := range compSlice {
				if c, ok := comp.(T); ok {
					comps = append(comps, c)
				}
			}
		}
	}
	return comps
}

// removeEntity removes the entity at slot idx using swap-with-last-element,
// which keeps every component slice aligned with the entities slice in O(1).
// It reports the EntityID that was moved into idx (if any) so the caller can
// update that entity's archetypeIdx bookkeeping.
func (a *Archetype) removeEntity(idx int) (movedID EntityID, moved bool) {
	last := len(a.entities) - 1
	if idx < 0 || idx > last {
		return 0, false
	}

	if idx != last {
		a.entities[idx] = a.entities[last]
		movedID, moved = a.entities[idx], true
	}
	a.entities = a.entities[:last]

	for compType, raw := range a.componentData {
		compSlice := raw.([]Component)
		compLast := len(compSlice) - 1
		if compLast < 0 {
			continue
		}
		if idx != last && idx <= compLast {
			compSlice[idx] = compSlice[compLast]
		}
		a.componentData[compType] = compSlice[:compLast]
	}

	return movedID, moved
}

type ArchetypeManager struct {
	archetypes map[uint64]*Archetype
	keyToID    map[ArchetypeKeyHash]uint64 //can't use ArchetypeKey as a map key directly, so we hash it
	nextID     uint64
	mu         sync.RWMutex
}

func NewArchetypeManager() *ArchetypeManager {
	return &ArchetypeManager{
		archetypes: make(map[uint64]*Archetype),
		keyToID:    make(map[ArchetypeKeyHash]uint64),
		nextID:     1, // start IDs from 1
	}
}

func (am *ArchetypeManager) GetOrCreateArchetype(componentTypes ...reflect.Type) *Archetype {
	am.mu.Lock()
	defer am.mu.Unlock()

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
	am.mu.RLock()
	defer am.mu.RUnlock()

	archetype, exists := am.archetypes[id]
	return archetype, exists
}

func (am *ArchetypeManager) GetArchetypeByKeyHash(hash ArchetypeKeyHash) (*Archetype, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if archetypeID, exists := am.keyToID[hash]; exists {
		return am.archetypes[archetypeID], true
	}
	return nil, false
}
