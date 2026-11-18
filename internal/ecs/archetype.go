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

// trackChangesType matches the core.TrackChanges marker structurally: a
// component type opts into per-row change tracking by implementing
// TrackChanges() with a value receiver.
var trackChangesType = reflect.TypeOf((*interface{ TrackChanges() })(nil)).Elem()

func isTracked(t reflect.Type) bool {
	return t != nil && t.Implements(trackChangesType)
}

type changeTicks struct {
	added   []uint64
	changed []uint64
}

type tickPair struct {
	added   uint64
	changed uint64
}

// Archetype groups entities that share the same component types.
//
// Each component column has one value per entity, in the same row order as
// the entity IDs. Tracked component types additionally carry row-aligned
// change ticks.
type Archetype struct {
	archetypeID
	key       key
	entityIDs []EntityID
	columns   map[reflect.Type][]any
	tracks    map[reflect.Type]*changeTicks
}

func newArchetype(id archetypeID, key key) *Archetype {
	columns := make(map[reflect.Type][]any, len(key))
	var tracks map[reflect.Type]*changeTicks
	for _, typ := range key {
		columns[typ] = make([]any, 0, initialColumnCapacity)
		if isTracked(typ) {
			if tracks == nil {
				tracks = make(map[reflect.Type]*changeTicks, len(key))
			}
			tracks[typ] = &changeTicks{
				added:   make([]uint64, 0, initialColumnCapacity),
				changed: make([]uint64, 0, initialColumnCapacity),
			}
		}
	}
	return &Archetype{
		archetypeID: id,
		key:         key,
		entityIDs:   make([]EntityID, 0, initialColumnCapacity),
		columns:     columns,
		tracks:      tracks,
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

// EntityIDs returns a copy of the entity IDs in row order.
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
	for _, tr := range a.tracks {
		tr.added = append(tr.added, 0)
		tr.changed = append(tr.changed, 0)
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
	for _, tr := range a.tracks {
		oldLen := len(tr.added)
		tr.added = slices.Grow(tr.added, len(entityIDs))[:oldLen+len(entityIDs)]
		tr.changed = slices.Grow(tr.changed, len(entityIDs))[:oldLen+len(entityIDs)]
		clear(tr.added[oldLen:])
		clear(tr.changed[oldLen:])
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
		col[lastIndex] = nil
		a.columns[typ] = col[:lastIndex]
	}
	for _, tr := range a.tracks {
		if index != lastIndex {
			tr.added[index] = tr.added[lastIndex]
			tr.changed[index] = tr.changed[lastIndex]
		}
		tr.added[lastIndex] = 0
		tr.changed[lastIndex] = 0
		tr.added = tr.added[:lastIndex]
		tr.changed = tr.changed[:lastIndex]
	}

	return movedID, moved
}

func (a *Archetype) stampAdded(typ reflect.Type, index int, tick uint64) {
	if tr, ok := a.tracks[typ]; ok && index >= 0 && index < len(tr.added) {
		tr.added[index] = tick
		tr.changed[index] = tick
	}
}

func (a *Archetype) stampAddedRange(typ reflect.Type, start, end int, tick uint64) {
	tr, ok := a.tracks[typ]
	if !ok {
		return
	}
	for i := max(start, 0); i < min(end, len(tr.added)); i++ {
		tr.added[i] = tick
		tr.changed[i] = tick
	}
}

func (a *Archetype) StampChanged(typ reflect.Type, index int, tick uint64) {
	if tr, ok := a.tracks[typ]; ok && index >= 0 && index < len(tr.changed) {
		tr.changed[index] = tick
	}
}

// Ticks returns when a tracked component was added and last written.
// It reports false for untracked types and out-of-range rows.
func (a *Archetype) Ticks(typ reflect.Type, index int) (added, changed uint64, ok bool) {
	tr, ok := a.tracks[typ]
	if !ok || index < 0 || index >= len(tr.added) {
		return 0, 0, false
	}
	return tr.added[index], tr.changed[index], true
}

func (a *Archetype) tickPairAt(typ reflect.Type, index int) tickPair {
	added, changed, _ := a.Ticks(typ, index)
	return tickPair{added: added, changed: changed}
}

func (a *Archetype) setTicks(typ reflect.Type, index int, pair tickPair) {
	if tr, ok := a.tracks[typ]; ok && index >= 0 && index < len(tr.added) {
		tr.added[index] = pair.added
		tr.changed[index] = pair.changed
	}
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

// Column returns the backing slice for typ, or nil if the type is absent.
// The slice aliases archetype storage; do not append to it or retain it across
// structural changes, which may move rows or reallocate the column. Writes
// to tracked components must also call [Archetype.StampChanged].
func (a *Archetype) Column(typ reflect.Type) []any {
	return a.columns[typ]
}

// EachEntity calls yield for each entity's row index and ID until yield
// returns false. Do not structurally modify the archetype during iteration.
func (a *Archetype) EachEntity(yield func(index int, entityID EntityID) bool) {
	for i, entityID := range a.entityIDs {
		if !yield(i, entityID) {
			return
		}
	}
}

type store struct {
	archetypes map[archetypeID]*Archetype
	byHash     map[hash]archetypeID
	generation uint64
	nextID     archetypeID
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
