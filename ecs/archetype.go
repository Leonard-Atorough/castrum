package ecs

import (
	"reflect"
	"slices"
	"sort"
	"strings"
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
		return sorted[i].String() < sorted[j].String()
	})
	return sorted
}

// Len returns the number of component types in the key.
func (ak ArchetypeKey) Len() int {
	return len(ak)
}

// String returns a string representation of the ArchetypeKey.
func (ak ArchetypeKey) String() string {
	if len(ak) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, t := range ak {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(t.String())
	}
	sb.WriteString("]")
	return sb.String()
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

// Equals checks if the current ArchetypeKey is equal to another ArchetypeKey.
func (ak ArchetypeKey) Equals(other ArchetypeKey) bool {
	if len(ak) != len(other) {
		return false
	}
	for i, t := range ak {
		if t != other[i] {
			return false
		}
	}
	return true
}

// ContainsAll checks if the current ArchetypeKey contains all types from another ArchetypeKey.
func (ak ArchetypeKey) ContainsAll(other ArchetypeKey) bool {
	if len(other) == 0 {
		return true
	}
	if len(ak) < len(other) {
		return false
	}
	if len(ak) < 8 {
		for _, t := range other {
			found := slices.Contains(ak, t)
			if !found {
				return false
			}
		}
		return true
	}
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

// ContainsAny checks if the current ArchetypeKey contains any of the types from another ArchetypeKey.
func (ak ArchetypeKey) ContainsAny(other ArchetypeKey) bool {
	if len(other) == 0 {
		return false
	}

	if len(ak) == 0 {
		return false
	}
	if len(ak) < 8 {
		for _, t := range other {
			found := slices.Contains(ak, t)
			if found {
				return true
			}
		}
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

// ContainsOnly checks if the current ArchetypeKey contains exactly the same types as another ArchetypeKey.
func (ak ArchetypeKey) ContainsExactly(other ArchetypeKey) bool {
	return ak.Equals(other)
}

// Archetype represents a collection of entities that share the same set of component types.
type Archetype struct {
	ID uint64 // Unique identifier for the archetype

	componentTypes ArchetypeKey         // The set of component types for this archetype
	entities       []EntityID           // The list of entity IDs in this archetype
	componentData  map[reflect.Type]any // The actual component data, keyed by component type
}

// NewArchetype creates a new Archetype with the given ID and component types. It initializes the entities slice and component data map.
func NewArchetype(id uint64, componentTypes ArchetypeKey) *Archetype {
	return &Archetype{
		ID:             id,
		componentTypes: componentTypes,
		entities:       make([]EntityID, 0, 16),
		componentData:  make(map[reflect.Type]any),
	}
}

// Entities returns the list of entity IDs in the archetype.
func (a *Archetype) Entities() []EntityID {
	return a.entities
}

// ComponentsAtIndex returns the components of type T at the given index in the archetype.
func (a *Archetype) ComponentsAtIndex[T Component](archetypIdx int) []T {
	var comps []T
	compType := reflect.TypeFor[T]()

	// Use the provided archetypIdx directly
	idx := archetypIdx
	if idx < 0 || idx >= len(a.entities) {
		return comps
	}

	// Get the typed slice for this component type
	if rawSlice, exists := a.componentData[compType]; exists {
		// Use reflection to access the slice generically
		sliceVal := reflect.ValueOf(rawSlice)
		if sliceVal.Kind() == reflect.Slice && idx < sliceVal.Len() {
			compVal := sliceVal.Index(idx)
			if compVal.IsValid() {
				if c, ok := compVal.Interface().(T); ok {
					comps = append(comps, c)
				}
			}
		}
	}
	return comps
}

// Len returns the number of entities in the archetype.
func (a *Archetype) Len() int {
	return len(a.entities)
}

// removeEntity removes the entity at the given index using the swap-with-last-element strategy.
// It returns the ID of the entity that was moved into the removed entity's slot (if any) and a boolean indicating if a move occurred.
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

	for compType, rawSlice := range a.componentData {
		// Use reflection to handle typed slices generically
		sliceVal := reflect.ValueOf(rawSlice)
		if sliceVal.Kind() != reflect.Slice {
			continue
		}
		compLast := sliceVal.Len() - 1
		if compLast < 0 {
			continue
		}
		if idx != last && idx <= compLast {
			// Swap element at idx with element at compLast
			if sliceVal.Index(idx).IsValid() && sliceVal.Index(compLast).IsValid() {
				sliceVal.Index(idx).Set(sliceVal.Index(compLast))
			}
		}
		// Truncate the slice and update storage
		newSlice := sliceVal.Slice(0, compLast)
		a.componentData[compType] = newSlice.Interface()
	}

	return movedID, moved
}

// ArchetypeManager manages all archetypes in the ECS, providing efficient lookup and creation.
type ArchetypeManager struct {
	archetypes map[uint64]*Archetype
	keyToID    map[ArchetypeKeyHash]uint64
	nextID     uint64
	mu         sync.RWMutex
}

// NewArchetypeManager creates and initializes a new ArchetypeManager.
func NewArchetypeManager() *ArchetypeManager {
	return &ArchetypeManager{
		archetypes: make(map[uint64]*Archetype),
		keyToID:    make(map[ArchetypeKeyHash]uint64),
		nextID:     1, // start IDs from 1
	}
}

// GetOrCreateArchetype retrieves an existing archetype matching the given component types,
// or creates a new one if none exists. It returns the archetype instance.
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

// GetArchetypeByID retrieves an archetype by its unique ID. It returns the archetype and a boolean indicating if it was found.
func (am *ArchetypeManager) GetArchetypeByID(id uint64) (*Archetype, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	archetype, exists := am.archetypes[id]
	return archetype, exists
}

// GetArchetypeByKeyHash retrieves an archetype by the hash of its component types key. It returns the archetype and a boolean indicating if it was found.
func (am *ArchetypeManager) GetArchetypeByKeyHash(hash ArchetypeKeyHash) (*Archetype, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if archetypeID, exists := am.keyToID[hash]; exists {
		return am.archetypes[archetypeID], true
	}
	return nil, false
}

// CleanupEmptyArchetypes removes archetypes that have no entities.
// This should be called periodically to prevent memory bloat from archetype proliferation.
// Safe for concurrent use.
func (am *ArchetypeManager) CleanupEmptyArchetypes() {
	am.mu.Lock()
	defer am.mu.Unlock()

	for id, archetype := range am.archetypes {
		if len(archetype.entities) == 0 {
			delete(am.archetypes, id)
			// Remove from keyToID map
			delete(am.keyToID, archetype.componentTypes.Hash())
		}
	}
}
