package ecs

import (
	"fmt"
	"reflect"
	"slices"
)

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

// Removal records the removal of a tracked component type from an entity.
type Removal struct {
	// Entity identifies the entity that lost the component.
	Entity EntityID
	// Tick is the change tick at which the removal was recorded.
	Tick uint64
}

// Service manages entities, their component values, and their archetype locations.
type Service struct {
	// store owns the entity archetypes.
	store *store
	// locations maps each live entity to its archetype row.
	locations map[EntityID]Location
	// tick is the current change tick, advanced by AdvanceTick. Writes stamp
	// tracked rows with it.
	tick uint64
	// removals holds per-type removal records for the current tick window,
	// created lazily on the first tracked removal. prevRemovals holds the
	// previous tick window's records; both rotate on AdvanceTick.
	removals     map[reflect.Type][]Removal
	prevRemovals map[reflect.Type][]Removal
}

// NewService returns an empty entity service.
func NewService() *Service {
	return &Service{
		store:     newArchetypes(),
		locations: make(map[EntityID]Location),
	}
}

// Tick returns the current change tick.
func (s *Service) Tick() uint64 {
	return s.tick
}

// AdvanceTick advances the change tick by one and rotates the removal
// window: records from the tick that just ended stay readable until the next
// advance, then are dropped.
func (s *Service) AdvanceTick() {
	s.tick++
	// Swap wholesale so an empty current window still retires the previous
	// one; clear drops the two-tick-old records without aliasing them.
	// Nil maps (no tracked removals yet) swap and clear as no-ops.
	s.prevRemovals, s.removals = s.removals, s.prevRemovals
	clear(s.removals)
}

// Removals returns the removal records for typ from the current and the
// previous tick window, oldest first. The result is newly allocated, or nil
// when the type is untracked or nothing was removed in either window.
func (s *Service) Removals(typ reflect.Type) []Removal {
	prev := s.prevRemovals[typ]
	cur := s.removals[typ]
	if len(prev) == 0 && len(cur) == 0 {
		return nil
	}
	out := make([]Removal, 0, len(prev)+len(cur))
	out = append(out, prev...)
	out = append(out, cur...)
	return out
}

// logRemoval records a removal of typ on entityID at the current tick.
// Untracked types are not recorded.
func (s *Service) logRemoval(typ reflect.Type, entityID EntityID) {
	if !isTracked(typ) {
		return
	}
	if s.removals == nil {
		s.removals = make(map[reflect.Type][]Removal)
	}
	s.removals[typ] = append(s.removals[typ], Removal{Entity: entityID, Tick: s.tick})
}

// HasEntity reports whether entityID exists in the service.
func (s *Service) HasEntity(entityID EntityID) bool {
	_, ok := s.locations[entityID]
	return ok
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
		arch.columns[t][index] = values[i]
		arch.stampAdded(t, index, s.tick)
	}
	s.locations[entityID] = Location{
		ArchetypeID: arch.ID(),
		Index:       index,
	}
	return nil

}

// CreateBatch adds multiple entities with the same component types and values.
// The same validation rules as [Service.Create] apply. IDs must be distinct.
// Invalid input leaves the service unchanged.
func (s *Service) CreateBatch(ids []EntityID, types []reflect.Type, values []any) error {
	if len(ids) <= 0 {
		return fmt.Errorf("count must be positive")
	}
	if err := validateComponentInput(types, values); err != nil {
		return err
	}
	for _, value := range values {
		if err := validateComponent(value); err != nil {
			return err
		}
	}

	seen := make(map[EntityID]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := s.locations[id]; exists {
			return fmt.Errorf("entity with ID %d already exists", id)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate entity ID %d in batch", id)
		}
		seen[id] = struct{}{}
	}

	arch := s.store.getOrCreate(types...)
	start := arch.insertEntities(ids)
	end := start + len(ids)
	for j, t := range types {
		col := arch.columns[t]
		for i := start; i < end; i++ {
			col[i] = values[j]
		}
		arch.stampAddedRange(t, start, end, s.tick)
	}
	for i, id := range ids {
		s.locations[id] = Location{
			ArchetypeID: arch.ID(),
			Index:       start + i,
		}
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

	for _, t := range arch.key {
		s.logRemoval(t, entityID)
	}
	movedID, moved := arch.removeEntity(loc.Index)
	delete(s.locations, entityID)
	if moved {
		s.locations[movedID] = Location{
			ArchetypeID: loc.ArchetypeID,
			Index:       loc.Index,
		}
	}
	s.store.cleanupEmpty(arch)

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
			from.StampChanged(t, loc.Index, s.tick)
		}
		return Result{}
	}

	oldKey := from.key
	oldValues := make([]any, oldKey.Len())
	// Ticks are snapshotted only when the archetype tracks anything, so
	// untracked migrations allocate nothing extra.
	var oldTicks []tickPair
	if len(from.tracks) > 0 {
		oldTicks = make([]tickPair, oldKey.Len())
	}
	for i, t := range oldKey {
		oldValues[i] = from.component(loc.Index, t)
		if oldTicks != nil {
			oldTicks[i] = from.tickPairAt(t, loc.Index)
		}
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
		if oldTicks != nil {
			to.setTicks(t, newIndex, oldTicks[i])
		}
	}
	for i, t := range types {
		to.setComponent(newIndex, t, values[i])
		if oldKey.contains(t) {
			to.StampChanged(t, newIndex, s.tick)
		} else {
			to.stampAdded(t, newIndex, s.tick)
		}
	}
	s.locations[entityID] = Location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty(from)

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

	for _, t := range types {
		if from.key.contains(t) {
			s.logRemoval(t, entityID)
		}
	}

	remaining := make(key, 0, from.key.Len())
	for _, t := range from.key {
		if !slices.Contains(types, t) {
			remaining = append(remaining, t)
		}
	}

	// Snapshot surviving values and change ticks in type order before
	// swap-removal reuses the row. Ticks are snapshotted only when the
	// archetype tracks anything.
	oldValues := make([]any, remaining.Len())
	var oldTicks []tickPair
	if len(from.tracks) > 0 {
		oldTicks = make([]tickPair, remaining.Len())
	}
	for i, t := range remaining {
		oldValues[i] = from.component(loc.Index, t)
		if oldTicks != nil {
			oldTicks[i] = from.tickPairAt(t, loc.Index)
		}
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
		if oldTicks != nil {
			to.setTicks(t, newIndex, oldTicks[i])
		}
	}
	s.locations[entityID] = Location{
		ArchetypeID: to.ID(),
		Index:       newIndex,
	}

	s.store.cleanupEmpty(from)

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
	arch.StampChanged(t, loc.Index, s.tick)
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
