package ecs

import (
	"reflect"
	"slices"
	"sort"
	"strings"
)

// FNV-1a hash constants for 64-bit
const (
	fnvOffsetBasis64 = 14695981039346656037
	fnvPrime64       = 1099511628211
)

// ArchetypeID represents a unique identifier for an archetype in the ECS system.
type ArchetypeID uint64

// Key represents a unique combination of component types that define an archetype.
type Key []reflect.Type

// A KeyHash represents a hash value for a Key, used for efficient lookups and comparisons.
type KeyHash uint64

func newKey(types ...reflect.Type) Key {
	if len(types) == 0 {
		return nil
	}

	// deduplicate and clean nil
	uniqueTypes := make(map[reflect.Type]struct{})
	cleanedTypes := make([]reflect.Type, 0, len(types))
	for _, t := range types {
		if t == nil {
			continue
		}
		if _, exists := uniqueTypes[t]; !exists {
			uniqueTypes[t] = struct{}{}
			cleanedTypes = append(cleanedTypes, t)
		}
	}
	types = cleanedTypes

	sortedTypes := make([]reflect.Type, len(types))
	copy(sortedTypes, types)
	// Sort the types to ensure consistent ordering for the key.
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].String() < sortedTypes[j].String()
	})
	return Key(sortedTypes)
}

// Hash computes the FNV-1a hash of the key for efficient lookups.
func (k Key) Hash() KeyHash {
	if len(k) == 0 || k == nil {
		return 0
	}
	var hash uint64 = fnvOffsetBasis64
	for _, t := range k {
		for i := 0; i < len(t.String()); i++ {
			hash ^= uint64(t.String()[i])
			hash *= fnvPrime64
		}
	}
	return KeyHash(hash)
}

// Equals checks if two keys are equal by comparing their component types.
func (k Key) Equals(other Key) bool {
	if len(k) != len(other) {
		return false
	}
	for i, typ := range k {
		if typ != other[i] {
			return false
		}
	}
	return true
}

// String returns a string representation of the key, with component types separated by semicolons.
func (k Key) String() string {
	var str strings.Builder
	for _, t := range k {
		str.WriteString(t.String())
		str.WriteString(";")
	}
	return str.String()
}

// Len returns the number of component types in the key.
func (k Key) Len() int {
	return len(k)
}

// Contains checks if the key contains the specified component type.
func (k Key) Contains(t reflect.Type) bool {
	return slices.Contains(k, t)
}

// ContainsAll checks if the key contains all of the specified component types.
func (k Key) ContainsAll(types ...reflect.Type) bool {
	for _, t := range types {
		if !k.Contains(t) {
			return false
		}
	}
	return true
}

// ContainsAny checks if the key contains any of the specified component types.
func (k Key) ContainsAny(types ...reflect.Type) bool {
	return slices.ContainsFunc(types, k.Contains)
}

// ContainsNone checks if the key contains none of the specified component types.
func (k Key) ContainsNone(types ...reflect.Type) bool {
	return !k.ContainsAny(types...)
}

// ContainsKey checks if the key contains all component types of another key.
func (k Key) ContainsKey(other Key) bool {
	return k.ContainsAll(other...)
}

// ContainsAnyKey checks if the key contains any component type of another key.
func (k Key) ContainsAnyKey(other Key) bool {
	return k.ContainsAny(other...)
}

// ContainsNoKey checks if the key contains none of the component types of another key.
func (k Key) ContainsNoKey(other Key) bool {
	return k.ContainsNone(other...)
}



// IndexOf returns the index of the specified component type in the key, or -1 if not found.
func (k Key) IndexOf(t reflect.Type) int {
	for i, typ := range k {
		if typ == t {
			return i
		}
	}
	return -1
}

// Types returns a slice of all component types in the key.
func (k Key) Types() []reflect.Type {
	return slices.Clone(k)
}

type Archetype struct {
	id        ArchetypeID
	key       Key
	entityIDs []uint64
	columns   map[reflect.Type]any
}

func newArchetype(id ArchetypeID, key Key) *Archetype {
	columns := make(map[reflect.Type]any, key.Len())
	for _, typ := range key {
		columns[typ] = reflect.MakeSlice(reflect.SliceOf(typ), 0, 16).Interface()
	}
	return &Archetype{
		id:        id,
		key:       key,
		entityIDs: make([]uint64, 0, 16),
		columns:   columns,
	}
}

func (a *Archetype) ID() ArchetypeID {
	return a.id
}

func (a *Archetype) Key() Key {
	return a.key.Types()
}

// Len returns the number of entities in the archetype.
func (a *Archetype) Len() int {
	return len(a.entityIDs)
}

// EntityIDs returns a slice of all entity IDs in the archetype.
func (a *Archetype) EntityIDs() []uint64 {
	return slices.Clone(a.entityIDs)
}

// HasComponent checks if the archetype has a column for the specified component type.
func (a *Archetype) HasComponent(t reflect.Type) bool {
	_, exists := a.columns[t]
	return exists
}

// ComponentTypes returns a slice of all component types in the archetype.
func (a *Archetype) ComponentTypes() []reflect.Type {
	return a.key.Types()
}

// AddEntity adds an entity ID to the archetype and returns its index.
func (a *Archetype) AddEntity(id uint64) int {
	a.entityIDs = append(a.entityIDs, id)
	return len(a.entityIDs) - 1
}

// RemoveEntity removes the entity at the specified index from the archetype.
// It returns the ID of the entity that was moved to fill the gap (if any) and a boolean indicating if a move occurred.
func (a *Archetype) RemoveEntity(idx int) (movedID uint64, moved bool) {
	lastIdx := len(a.entityIDs) - 1
	if idx < 0 || idx > lastIdx {
		return 0, false
	}
	if idx != lastIdx {
		a.entityIDs[idx] = a.entityIDs[lastIdx]
		movedID, moved = a.entityIDs[idx], true
	}
	a.entityIDs = a.entityIDs[:lastIdx]

	for typ, col := range a.columns {
		v := reflect.ValueOf(col)
		if v.Kind() != reflect.Slice {
			continue
		}
		compLast := v.Len() - 1
		if compLast < 0 {
			continue
		}
		if idx != compLast && idx < compLast {
			v.Index(idx).Set(v.Index(compLast))
		}
		a.columns[typ] = v.Slice(0, compLast).Interface()
	}
	return movedID, moved
}

// Get retrieves the component value of the specified type for the entity at the given index.
// It returns the value and a boolean indicating if the value was found.
func (a *Archetype) Get(idx int, typ reflect.Type) (any, bool) {
	col, exists := a.columns[typ]
	if !exists {
		return nil, false
	}
	v := reflect.ValueOf(col)
	if v.Kind() != reflect.Slice || idx < 0 || idx >= v.Len() {
		return nil, false
	}
	return v.Index(idx).Interface(), true
}

// Set sets the component value of the specified type for the entity at the given index.
// It returns a boolean indicating if the operation was successful.
func (a *Archetype) Set(idx int, typ reflect.Type, value any) bool {
	col, exists := a.columns[typ]
	if !exists {
		return false
	}
	colVal := reflect.ValueOf(col)
	if colVal.Kind() != reflect.Slice || idx < 0 {
		return false
	}
	valueVal := reflect.ValueOf(value)
	if !valueVal.IsValid() || !valueVal.Type().AssignableTo(colVal.Type().Elem()) {
		return false
	}
	if idx >= colVal.Len() {
		newLen := idx + 1
		if colVal.Cap() < newLen {
			newCap := max(colVal.Cap()*2, newLen)
			newSlice := reflect.MakeSlice(colVal.Type(), newLen, newCap)
			reflect.Copy(newSlice, colVal)
			colVal = newSlice
		} else {
			colVal = colVal.Slice(0, newLen)
		}
		a.columns[typ] = colVal.Interface()
	}
	colVal.Index(idx).Set(valueVal)
	return true
}

type ArchetypeStore struct {
	archetypes map[ArchetypeID]*Archetype
	byKey      map[KeyHash][]ArchetypeID
	nextID     ArchetypeID
}

func newArchetypeStore() *ArchetypeStore {
	return &ArchetypeStore{
		archetypes: make(map[ArchetypeID]*Archetype),
		byKey:      make(map[KeyHash][]ArchetypeID),
		nextID:     0,
	}
}

// GetOrCreate retrieves an existing archetype matching the specified component types,
// or creates a new one if none exists.
func (s *ArchetypeStore) GetOrCreate(types ...reflect.Type) *Archetype {
	key := newKey(types...)
	hash := key.Hash()

	if ids, exists := s.byKey[hash]; exists {
		for _, id := range ids {
			arch := s.archetypes[id]
			if arch.Key().Equals(key) {
				return arch
			}
		}
	}

	id := s.nextID
	s.nextID++
	arch := newArchetype(id, key)
	s.archetypes[id] = arch
	s.byKey[hash] = append(s.byKey[hash], id)
	return arch
}

// Get retrieves the archetype with the specified ID.
// It returns the archetype and a boolean indicating if it was found.
func (s *ArchetypeStore) Get(id ArchetypeID) (*Archetype, bool) {
	arch, exists := s.archetypes[id]
	return arch, exists
}

// Find retrieves the archetype matching the specified key.
// It returns the archetype and a boolean indicating if it was found.
func (s *ArchetypeStore) Find(key Key) (*Archetype, bool) {
	hash := key.Hash()
	if ids, exists := s.byKey[hash]; exists {
		for _, id := range ids {
			arch := s.archetypes[id]
			if arch.Key().Equals(key) {
				return arch, true
			}
		}
	}
	return nil, false
}

// FindByTypes retrieves the archetype matching the specified component types.
// It returns the archetype and a boolean indicating if it was found.
func (s *ArchetypeStore) FindByTypes(types ...reflect.Type) (*Archetype, bool) {
	key := newKey(types...)
	return s.Find(key)
}

func (s *ArchetypeStore) Matching(required Key, excluded Key) []*Archetype {
	var result []*Archetype
	for _, arch := range s.archetypes {
		if arch.Key().ContainsAll(required...) && arch.Key().ContainsNone(excluded...) {
			result = append(result, arch)
		}
	}
	return result
}

// MatchingByTypes retrieves all archetypes that match the specified required and excluded component types.
// It returns a slice of matching archetypes.
func (s *ArchetypeStore) MatchingByTypes(requiredTypes []reflect.Type, excludedTypes []reflect.Type) []*Archetype {
	requiredKey := newKey(requiredTypes...)
	excludedKey := newKey(excludedTypes...)
	return s.Matching(requiredKey, excludedKey)
}

// CleanupEmpty removes all archetypes that have no entities.
func (s *ArchetypeStore) CleanupEmpty() {
	for id, arch := range s.archetypes {
		if arch.Len() == 0 {
			delete(s.archetypes, id)
			hash := arch.Key().Hash()
			ids := s.byKey[hash]
			for i, storedID := range ids {
				if storedID == id {
					s.byKey[hash] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}
}

func (s *ArchetypeStore) Count() int {
	return len(s.archetypes)
}
