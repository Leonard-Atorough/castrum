// Package ecs stores entities in archetypes grouped by component type sets.
// [Service] manages entity locations and component values across those archetypes.
package ecs

import (
	"reflect"
	"slices"
	"sort"
	"strings"
)

const (
	fnvOffsetBasis64 = 14695981039346656037
	fnvPrime64       = 1099511628211
)

type archetypeID uint64

// EntityID uniquely identifies an entity within a [Service].
type EntityID uint64

type key []reflect.Type

type hash uint64

func newKey(types ...reflect.Type) key {
	sortedTypes := make([]reflect.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].String() < sortedTypes[j].String()
	})
	uniqueTypes := make([]reflect.Type, 0, len(sortedTypes))
	for i, t := range sortedTypes {
		if i == 0 || t != sortedTypes[i-1] {
			uniqueTypes = append(uniqueTypes, t)
		}
	}
	return uniqueTypes
}

func (k key) hash() hash {
	var h uint64
	h = fnvOffsetBasis64
	for _, t := range k {
		s := t.String()
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= fnvPrime64
		}
	}
	return hash(h)
}

func (k key) equals(other key) bool {
	if len(k) != len(other) {
		return false
	}
	for i := range k {
		if k[i] != other[i] {
			return false
		}
	}
	return true
}

func (k key) contains(t reflect.Type) bool {
	return slices.Contains(k, t)
}

func (k key) containsAll(types ...reflect.Type) bool {
	for _, t := range types {
		if !k.contains(t) {
			return false
		}
	}
	return true
}

func (k key) containsNone(types ...reflect.Type) bool {
	return !slices.ContainsFunc(types, k.contains)
}

// Len returns the number of component types in the key.
func (k key) Len() int {
	return len(k)
}

// String returns the component types in the key as a bracketed list.
func (k key) String() string {
	strs := make([]string, len(k))
	for i, t := range k {
		strs[i] = t.String()
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

const initialColumnCapacity = 16

// Archetype groups entities that share the same component types.
//
// Each component column has one value per entity, in the same row order as
// the entity IDs.
type Archetype struct {
	// archetypeID uniquely identifies this archetype within its store.
	archetypeID
	// key is the component type set shared by every entity.
	key key
	// entityIDs lists the entities in row order.
	entityIDs []EntityID
	// columns stores each component type's values in the same row order.
	columns map[reflect.Type][]any
}

func newArchetype(id archetypeID, key key) *Archetype {
	columns := make(map[reflect.Type][]any, len(key))
	for _, typ := range key {
		columns[typ] = make([]any, 0, initialColumnCapacity)
	}
	return &Archetype{
		archetypeID: id,
		key:         key,
		entityIDs:   make([]EntityID, 0, initialColumnCapacity),
		columns:     columns,
	}
}

// ID returns the archetype's identifier within its store.
func (a *Archetype) ID() archetypeID {
	return a.archetypeID
}

// Key returns the component types shared by every entity in the archetype.
// The returned slice aliases the archetype's key and must not be modified.
func (a *Archetype) Key() key {
	return a.key
}

// EntityIDs returns a copy of the archetype's entity ID list. Hot paths
// should prefer EachEntity, which iterates the list in place.
func (a *Archetype) EntityIDs() []EntityID {
	return slices.Clone(a.entityIDs)
}

// Len returns the number of entities in the archetype.
func (a *Archetype) Len() int {
	return len(a.entityIDs)
}

func (a *Archetype) insertEntity(entityID EntityID) int {
	a.entityIDs = append(a.entityIDs, entityID)
	for typ, col := range a.columns {
		a.columns[typ] = append(col, nil)
	}
	return len(a.entityIDs) - 1
}

func (a *Archetype) insertEntities(entityIDs []EntityID) int {
	start := len(a.entityIDs)
	a.entityIDs = append(a.entityIDs, entityIDs...)
	for typ, col := range a.columns {
		oldLen := len(col)
		col = slices.Grow(col, len(entityIDs))
		col = col[:oldLen+len(entityIDs)]
		a.columns[typ] = col
	}
	return start
}

func (a *Archetype) removeEntity(index int) (movedID EntityID, moved bool) {
	lastIndex := len(a.entityIDs) - 1
	if index < 0 || index > lastIndex {
		return 0, false // 0 is the zero value for entityID
	}
	if index != lastIndex {
		a.entityIDs[index] = a.entityIDs[lastIndex]
		movedID = a.entityIDs[index]
		moved = true
	}
	a.entityIDs = a.entityIDs[:lastIndex]

	for typ, col := range a.columns {
		if index != lastIndex {
			col[index] = col[lastIndex]
		}
		a.columns[typ] = col[:lastIndex]
	}

	return movedID, moved
}

func (a *Archetype) setComponent(index int, typ reflect.Type, value any) bool {
	if index < 0 {
		return false
	}
	col, ok := a.columns[typ]
	if !ok || index >= len(col) {
		return false
	}
	valueType := reflect.TypeOf(value)
	if valueType == nil || !valueType.AssignableTo(typ) {
		return false
	}
	col[index] = value
	return true
}

func (a *Archetype) component(index int, typ reflect.Type) any {
	if index < 0 {
		return nil
	}
	col, ok := a.columns[typ]
	if !ok || index >= len(col) {
		return nil
	}
	return col[index]
}

// Column returns the backing slice for typ, or nil if the archetype has no
// such component type. The slice aliases archetype storage: do not append to
// it or retain it across structural changes, which may reallocate the column.
func (a *Archetype) Column(typ reflect.Type) []any {
	return a.columns[typ]
}

// EachEntity calls yield for each entity's row index and ID. Iteration stops
// when yield returns false.
func (a *Archetype) EachEntity(yield func(index int, entityID EntityID) bool) {
	for i, entityID := range a.entityIDs {
		if !yield(i, entityID) {
			return
		}
	}
}

type store struct {
	// archetypes holds the live archetypes, keyed by ID.
	archetypes map[archetypeID]*Archetype
	// byHash maps component-set hashes to archetype IDs.
	byHash map[hash]archetypeID
	// generation tracks changes to the set of archetypes.
	generation uint64
	// nextID is the next archetype ID to assign.
	nextID archetypeID
}

func newArchetypes() *store {
	return &store{
		archetypes: make(map[archetypeID]*Archetype),
		byHash:     make(map[hash]archetypeID),
		nextID:     1,
	}
}

func (s *store) nextArchetypeID() archetypeID {
	id := s.nextID
	s.nextID++
	return id
}

func (s *store) getOrCreate(types ...reflect.Type) *Archetype {
	key := newKey(types...)
	hash := key.hash()
	if id, ok := s.byHash[hash]; ok {
		if archetype := s.archetypes[id]; archetype.key.equals(key) {
			return archetype
		}

		for _, archetype := range s.archetypes {
			if archetype.key.equals(key) {
				return archetype
			}
		}
	}

	id := s.nextArchetypeID()
	archetype := newArchetype(id, key)
	s.byHash[hash] = id
	s.archetypes[id] = archetype
	s.generation++

	return archetype
}

func (s *store) get(id archetypeID) (*Archetype, bool) {
	archetype, ok := s.archetypes[id]
	return archetype, ok
}

func (s *store) matchInto(buf []*Archetype, required, excluded []reflect.Type) []*Archetype {
	buf = buf[:0]
	for _, archetype := range s.archetypes {
		if archetype.Key().containsAll(required...) && archetype.Key().containsNone(excluded...) {
			buf = append(buf, archetype)
		}
	}
	return buf
}

func (s *store) cleanupEmpty(archetype *Archetype) {
	if archetype.Len() != 0 {
		return
	}
	id := archetype.ID()
	delete(s.archetypes, id)
	if hash := archetype.Key().hash(); s.byHash[hash] == id {
		delete(s.byHash, hash)
	}
	s.generation++
}

// Len returns the number of archetypes in the store.
func (s *store) Len() int {
	return len(s.archetypes)
}
