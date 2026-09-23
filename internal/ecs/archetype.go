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

type id uint64

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

func (k key) Len() int {
	return len(k)
}

func (k key) String() string {
	strs := make([]string, len(k))
	for i, t := range k {
		strs[i] = t.String()
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

const initialColumnCapacity = 16

type archetype struct {
	id
	key       key
	entityIDs []uint64
	columns   map[reflect.Type][]any
}

func newArchetype(id id, key key) *archetype {
	columns := make(map[reflect.Type][]any, len(key))
	for _, typ := range key {
		columns[typ] = make([]any, 0, initialColumnCapacity)
	}
	return &archetype{
		id:        id,
		key:       key,
		entityIDs: make([]uint64, 0, initialColumnCapacity),
		columns:   columns,
	}
}

func (a *archetype) ID() id {
	return a.id
}

func (a *archetype) Key() key {
	return a.key
}

func (a *archetype) EntityIDs() []uint64 {
	return slices.Clone(a.entityIDs)
}

func (a *archetype) Len() int {
	return len(a.entityIDs)
}

func (a *archetype) insertEntity(entityID uint64) int {
	a.entityIDs = append(a.entityIDs, entityID)
	for typ, col := range a.columns {
		a.columns[typ] = append(col, nil)
	}
	return len(a.entityIDs) - 1
}

func (a *archetype) removeEntity(index int) (movedID uint64, moved bool) {
	lastIndex := len(a.entityIDs) - 1
	if index < 0 || index > lastIndex {
		return 0, false
	}
	if index != lastIndex {
		a.entityIDs[index] = a.entityIDs[lastIndex]
		movedID = a.entityIDs[index]
		moved = true
	}
	a.entityIDs = a.entityIDs[:lastIndex]

	// Columns always match entityIDs in length: swap the moved row down,
	// then shrink every column.
	for typ, col := range a.columns {
		if index != lastIndex {
			col[index] = col[lastIndex]
		}
		a.columns[typ] = col[:lastIndex]
	}

	return movedID, moved
}

func (a *archetype) setComponent(index int, typ reflect.Type, value any) bool {
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

func (a *archetype) component(index int, typ reflect.Type) any {
	if index < 0 {
		return nil
	}
	col, ok := a.columns[typ]
	if !ok || index >= len(col) {
		return nil
	}
	return col[index]
}

type store struct {
	archetypes map[id]*archetype
	byHash     map[hash]id
	nextID     id
}

func newArchetypes() *store {
	return &store{
		archetypes: make(map[id]*archetype),
		byHash:     make(map[hash]id),
		nextID:     1,
	}
}

func (s *store) nextArchetypeID() id {
	id := s.nextID
	s.nextID++
	return id
}

func (s *store) getOrCreate(types ...reflect.Type) *archetype {
	key := newKey(types...)
	hash := key.hash()
	if id, ok := s.byHash[hash]; ok {
		return s.archetypes[id]
	}

	id := s.nextArchetypeID()
	archetype := newArchetype(id, key)
	s.byHash[hash] = id
	s.archetypes[id] = archetype
	return archetype
}

func (s *store) get(id id) (*archetype, bool) {
	archetype, ok := s.archetypes[id]
	return archetype, ok
}

func (s *store) match(required []reflect.Type, excluded []reflect.Type) []*archetype {
	var result []*archetype
	for _, archetype := range s.archetypes {
		if archetype.Key().containsAll(required...) && archetype.Key().containsNone(excluded...) {
			result = append(result, archetype)
		}
	}
	return result
}

func (s *store) cleanupEmpty() {
	for id, archetype := range s.archetypes {
		if archetype.Len() == 0 {
			delete(s.archetypes, id)
			delete(s.byHash, archetype.Key().hash())
		}
	}
}

func (s *store) Len() int {
	return len(s.archetypes)
}

// location represents the position of an entity within an archetype,
// including the archetype ID and the index within that archetype.
type location struct {
	ArchetypeID id
	Index       int
}

// Result represents the outcome of an operation on an entity within the ECS.
type Result struct {
	// Moved indicates if the entity was moved to a different archetype as a result of the operation.
	Moved bool
	// MovedID is the ID of the entity that was moved, if any.
	MovedID uint64
	// Success indicates if the operation was successful.
	Success bool
	// Error contains any error encountered during the operation.
	Error error
}

// Service provides methods to create, remove, and update entities within the ECS,
// managing their locations and archetypes.
type Service struct {
	store     *store
	locations map[uint64]location
}

// NewService initializes and returns a new Service instance with an empty store and location map.
func NewService() *Service {
	return &Service{
		store:     newArchetypes(),
		locations: make(map[uint64]location),
	}
}

// Create adds a new entity with the specified components to the ECS.
// It returns a [Result] indicating the success or failure of the operation.
func (s *Service) Create(entityID uint64, types []reflect.Type, values []any) Result {
	if err := validateComponentInput(types, values); err != nil {
		return Result{
			Success: false,
			Error:   err,
		}
	}

	if _, exists := s.locations[entityID]; exists {
		return Result{
			Success: false,
			Error:   fmt.Errorf("entity with ID %d already exists", entityID),
		}
	}

	arch := s.store.getOrCreate(types...)
	index := arch.insertEntity(entityID)
	for i, t := range types {
		arch.setComponent(index, t, values[i])
	}
	s.locations[entityID] = location{
		ArchetypeID: arch.ID(),
		Index:       index,
	}
	return Result{
		Success: true,
	}

}

// Destroy deletes the entity with the specified ID from the ECS.
// It returns a [Result] indicating the success or failure of the operation,
// and whether the entity was moved within its archetype as a result of the removal.
func (s *Service) Destroy(entityID uint64) Result {
	loc, res := s.location(entityID)
	if !res.Success {
		return res
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Success: false,
			Error:   fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	movedID, moved := arch.removeEntity(loc.Index)
	delete(s.locations, entityID)
	if moved {
		s.locations[movedID] = location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}
	s.store.cleanupEmpty()

	return Result{
		Moved:   moved,
		MovedID: movedID,
		Success: true,
	}
}

// AddComponents attaches components to an existing entity, migrating it to
// the archetype that holds the union of its current and new components.
// Values of existing components are preserved. Adding a component the
// entity already has overwrites its value without migrating.
//
// It returns a [Result] reporting whether the entity migrated; Moved and
// MovedID refer to the target entity, not to swap-removal bookkeeping.
func (s *Service) AddComponents(entityID uint64, types []reflect.Type, values []any) Result {
	if err := validateComponentInput(types, values); err != nil {
		return Result{
			Success: false,
			Error:   err,
		}
	}

	loc, res := s.location(entityID)
	if !res.Success {
		return res
	}

	from, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Success: false,
			Error:   fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	if from.key.containsAll(types...) {
		for i, t := range types {
			from.setComponent(loc.Index, t, values[i])
		}
		return Result{Success: true}
	}

	oldKey := from.key
	oldValues := make([]any, oldKey.Len())
	for i, t := range oldKey {
		oldValues[i] = from.component(loc.Index, t)
	}

	movedID, moved := from.removeEntity(loc.Index)
	if moved {
		s.locations[movedID] = location{
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
	s.locations[entityID] = location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty()

	return Result{
		Success: true,
		Moved:   true,
		MovedID: entityID,
	}
}

// RemoveComponents detaches components from an existing entity, migrating
// it to the archetype that holds its remaining components. Values of the
// remaining components are preserved. Types the entity does not have are
// ignored; removing all components leaves the entity in an empty archetype.
//
// It returns a [Result] reporting whether the entity migrated; Moved and
// MovedID refer to the target entity, not to swap-removal bookkeeping.
func (s *Service) RemoveComponents(entityID uint64, types []reflect.Type) Result {
	if err := validateTypes(types); err != nil {
		return Result{
			Success: false,
			Error:   err,
		}
	}

	loc, res := s.location(entityID)
	if !res.Success {
		return res
	}

	from, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return Result{
			Success: false,
			Error:   fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	// Fast path: nothing to remove, nothing migrates.
	if from.key.containsNone(types...) {
		return Result{Success: true}
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
		s.locations[movedID] = location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}

	to := s.store.getOrCreate(remaining...)
	newIndex := to.insertEntity(entityID)
	for i, t := range remaining {
		to.setComponent(newIndex, t, oldValues[i])
	}
	s.locations[entityID] = location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty()

	return Result{
		Success: true,
		Moved:   true,
		MovedID: entityID,
	}
}

// Component retrieves the value of a specific component for the given entity.
// If the component is not found or the entity does not exist, an error is returned.
func (s *Service) Component(entityID uint64, t reflect.Type) (any, error) {
	val, res := s.resolveComponent(entityID, t)
	if !res.Success {
		return nil, res.Error
	}
	return val, nil
}

// HasComponent reports whether the given entity has a specific component.
// It returns false if the entity does not exist or does not have the
// component.
func (s *Service) HasComponent(entityID uint64, t reflect.Type) bool {
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

// SetComponent sets the value of a specific component for the given entity.
// It returns a boolean indicating success and an error if the operation failed.
func (s *Service) SetComponent(entityID uint64, t reflect.Type, value any) (bool, error) {
	loc, res := s.location(entityID)
	if !res.Success {
		return false, res.Error
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return false, fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID)
	}

	if arch.component(loc.Index, t) == nil {
		return false, fmt.Errorf("component of type %v not found for entity with ID %d", t, entityID)
	}

	success := arch.setComponent(loc.Index, t, value)
	if !success {
		return false, fmt.Errorf("failed to set component of type %v for entity with ID %d", t, entityID)
	}
	return true, nil
}

// Match returns a list of archetypes that match the required and excluded component types.
// The `required` parameter specifies the component types that must be present in the archetype.
// The `excluded` parameter specifies the component types that must not be present in the archetype.
func (s *Service) Match(required, excluded []reflect.Type) []*archetype {
	return s.store.match(required, excluded)
}

func (s *Service) location(entityID uint64) (location, Result) {
	loc, exists := s.locations[entityID]
	if !exists {
		return location{}, Result{
			Success: false,
			Error:   fmt.Errorf("entity with ID %d not found", entityID),
		}
	}
	return loc, Result{
		Success: true,
	}
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

func (s *Service) resolveComponent(entityID uint64, t reflect.Type) (any, Result) {
	loc, res := s.location(entityID)
	if !res.Success {
		return nil, res
	}

	arch, ok := s.store.get(loc.ArchetypeID)
	if !ok {
		return nil, Result{
			Success: false,
			Error:   fmt.Errorf("archetype with ID %d not found", loc.ArchetypeID),
		}
	}

	value := arch.component(loc.Index, t)
	if value == nil {
		return nil, Result{
			Success: false,
			Error:   fmt.Errorf("component of type %v not found for entity with ID %d", t, entityID),
		}
	}

	return value, Result{
		Success: true,
	}
}
