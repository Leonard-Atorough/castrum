// Package ecs stores entities in archetypes grouped by component type sets.
// [Service] manages entity locations and component values across those archetypes.
package ecs

import (
	"fmt"
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

func (s *store) cleanupEmpty() {
	for id, archetype := range s.archetypes {
		if archetype.Len() == 0 {
			delete(s.archetypes, id)

			if s.byHash[archetype.Key().hash()] == id {
				delete(s.byHash, archetype.Key().hash())
			}
		}
	}
	s.generation++
}

// Len returns the number of archetypes in the store.
func (s *store) Len() int {
	return len(s.archetypes)
}

// Location identifies an entity's row in an archetype.
type Location struct {
	// ArchetypeID identifies the archetype containing the entity.
	ArchetypeID archetypeID
	// Index is the entity's row within that archetype.
	Index int
}

// Result describes the outcome of an entity operation.
type Result struct {
	// Moved reports whether an entity moved within or between archetypes.
	Moved bool
	// MovedID identifies the moved entity when Moved is true.
	MovedID EntityID
	// Error is the failure encountered by the operation, if any.
	Error error
}

// Service manages entities, their component values, and their archetype locations.
type Service struct {
	// store owns the entity archetypes.
	store *store
	// locations maps each live entity to its archetype row.
	locations map[EntityID]Location
}

// NewService returns an empty entity service.
func NewService() *Service {
	return &Service{
		store:     newArchetypes(),
		locations: make(map[EntityID]Location),
	}
}

// Create adds an entity with the given component types and values.
// types and values must have the same length, and each value must be assignable
// to the type at the corresponding index. An entity ID already in use is an
// error.
func (s *Service) Create(entityID EntityID, types []reflect.Type, values []any) error {
	if err := validateComponentInput(types, values); err != nil {
		return err
	}
	for _, value := range values {
		if err := validateComponent(value); err != nil {
			return err
		}
	}

	if _, exists := s.locations[entityID]; exists {
		return fmt.Errorf("entity with ID %d already exists", entityID)
	}

	arch := s.store.getOrCreate(types...)
	index := arch.insertEntity(entityID)
	for i, t := range types {
		arch.setComponent(index, t, values[i])
	}
	s.locations[entityID] = Location{
		ArchetypeID: arch.ID(),
		Index:       index,
	}
	return nil

}

// Destroy removes an entity and its components. If removing its row moves
// another entity within the archetype, the result identifies that entity in
// MovedID. Destroy reports an error in Result.Error if entityID does not exist.
func (s *Service) Destroy(entityID EntityID) Result {
	loc, err := s.location(entityID)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Error: fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	movedID, moved := arch.removeEntity(loc.Index)
	delete(s.locations, entityID)
	if moved {
		s.locations[movedID] = Location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}
	s.store.cleanupEmpty()

	return Result{
		Moved:   moved,
		MovedID: movedID,
	}
}

// AddComponents attaches or replaces components on an existing entity.
// Existing component values are preserved when the entity migrates to the
// archetype for the resulting type set. If all specified types are already
// present, their values are replaced without migration. Component values
// must be assignable to their corresponding types and pass validation; an
// invalid input leaves the entity unchanged.
//
// The result reports errors in Error. Moved and MovedID identify the target
// entity when it migrates; they do not report any other entity moved by
// swap-removal.
func (s *Service) AddComponents(entityID EntityID, types []reflect.Type, values []any) Result {
	if err := validateComponentInput(types, values); err != nil {
		return Result{
			Error: err,
		}
	}
	// Validate before any mutation so a failure leaves the entity in its
	// current archetype, untouched.
	for _, value := range values {
		if err := validateComponent(value); err != nil {
			return Result{
				Error: err,
			}
		}
	}

	loc, err := s.location(entityID)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	from, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Error: fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	if from.key.containsAll(types...) {
		for i, t := range types {
			from.setComponent(loc.Index, t, values[i])
		}
		return Result{}
	}

	oldKey := from.key
	oldValues := make([]any, oldKey.Len())
	for i, t := range oldKey {
		oldValues[i] = from.component(loc.Index, t)
	}

	movedID, moved := from.removeEntity(loc.Index)
	if moved {
		s.locations[movedID] = Location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}

	to := s.store.getOrCreate(append(slices.Clone(oldKey), types...)...)
	newIndex := to.insertEntity(entityID)
	for i, t := range oldKey {
		to.setComponent(newIndex, t, oldValues[i])
	}
	for i, t := range types {
		to.setComponent(newIndex, t, values[i])
	}
	s.locations[entityID] = Location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty()

	return Result{
		Moved:   true,
		MovedID: entityID,
	}
}

// RemoveComponents detaches the specified component types from an existing
// entity. Types the entity does not have are ignored; remaining values are
// preserved when it migrates to the archetype for its remaining types.
// Removing all components leaves the entity in an empty archetype.
//
// The result reports errors in Error. Moved and MovedID identify the target
// entity when it migrates; they do not report any other entity moved by
// swap-removal.
func (s *Service) RemoveComponents(entityID EntityID, types []reflect.Type) Result {
	if err := validateTypes(types); err != nil {
		return Result{
			Error: err,
		}
	}

	loc, err := s.location(entityID)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	from, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Error: fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	// Fast path: nothing to remove, nothing migrates.
	if from.key.containsNone(types...) {
		return Result{}
	}

	remaining := make(key, 0, from.key.Len())
	for _, t := range from.key {
		if !slices.Contains(types, t) {
			remaining = append(remaining, t)
		}
	}

	// Snapshot current values before the swap-remove reuses the row.
	oldValues := make([]any, from.key.Len())
	for i, t := range from.key {
		oldValues[i] = from.component(loc.Index, t)
	}

	movedID, moved := from.removeEntity(loc.Index)
	if moved {
		s.locations[movedID] = Location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}

	to := s.store.getOrCreate(remaining...)
	newIndex := to.insertEntity(entityID)
	for i, t := range remaining {
		to.setComponent(newIndex, t, oldValues[i])
	}
	s.locations[entityID] = Location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty()

	return Result{
		Moved:   true,
		MovedID: entityID,
	}
}

// Component returns the value of type t on entityID. The second result is
// false if the entity or component does not exist.
func (s *Service) Component(entityID EntityID, t reflect.Type) (any, bool) {
	val, err := s.resolveComponent(entityID, t)
	if err != nil {
		return nil, false
	}

	return val, true
}

// HasComponent reports whether entityID exists and has a component of type t.
func (s *Service) HasComponent(entityID EntityID, t reflect.Type) bool {
	loc, exists := s.locations[entityID]
	if !exists {
		return false
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return false
	}

	return arch.component(loc.Index, t) != nil
}

// SetComponent replaces the value of an existing component on entityID.
// value must be assignable to t and pass validation. It returns an error if
// the entity or component does not exist, or if the value is invalid.
func (s *Service) SetComponent(entityID EntityID, t reflect.Type, value any) error {
	loc, err := s.location(entityID)
	if err != nil {
		return err
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID)
	}

	if err := validateComponent(value); err != nil {
		return err
	}
	if arch.component(loc.Index, t) == nil {
		return fmt.Errorf("component of type %v not found for entity with ID %d", t, entityID)
	}

	success := arch.setComponent(loc.Index, t, value)
	if !success {
		return fmt.Errorf("failed to set component of type %v for entity with ID %d", t, entityID)
	}
	return nil
}

// Match returns archetypes containing every required type and none of the
// excluded types. The returned slice is newly allocated; use [Service.MatchInto]
// to reuse a buffer.
func (s *Service) Match(required, excluded []reflect.Type) []*Archetype {
	return s.store.matchInto(nil, required, excluded)
}

// MatchInto returns archetypes containing every required type and none of the
// excluded types, reusing buf's capacity when possible. Pass the returned
// slice to the next call to retain that capacity.
func (s *Service) MatchInto(buf []*Archetype, required, excluded []reflect.Type) []*Archetype {
	return s.store.matchInto(buf, required, excluded)
}

// Generation returns the store generation, which changes when archetypes are
// created or empty archetypes are removed.
func (s *Service) Generation() uint64 {
	return s.store.generation
}

func (s *Service) location(entityID EntityID) (Location, error) {
	loc, exists := s.locations[entityID]
	if !exists {
		return Location{}, fmt.Errorf("entity with ID %d not found", entityID)
	}
	return Location{
		ArchetypeID: loc.ArchetypeID,
		Index:       loc.Index,
	}, nil
}

func validateTypes(types []reflect.Type) error {
	for i, t := range types {
		if t == nil {
			return fmt.Errorf("type at index %d is nil", i)
		}
	}
	return nil
}

func validateComponentInput(types []reflect.Type, values []any) error {
	if len(types) != len(values) {
		return fmt.Errorf("mismatched number of types and values")
	}
	if err := validateTypes(types); err != nil {
		return err
	}
	for i, t := range types {
		v := reflect.ValueOf(values[i])
		if !v.IsValid() || !v.Type().AssignableTo(t) {
			return fmt.Errorf("value at index %d is not assignable to type %v", i, t)
		}
	}
	return nil
}

func (s *Service) resolveComponent(entityID EntityID, t reflect.Type) (any, error) {
	loc, err := s.location(entityID)
	if err != nil {
		return nil, err
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return nil, fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID)
	}

	value := arch.component(loc.Index, t)
	if value == nil {
		return nil, fmt.Errorf("component of type %v not found for entity with ID %d", t, entityID)
	}

	return value, nil
}

func validateComponent(value any) error {
	if v, ok := value.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("component %T: %w", value, err)
		}
	}
	return nil
}
