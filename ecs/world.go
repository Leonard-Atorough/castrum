package ecs

import (
	"fmt"
	"reflect"
	"slices"
	"sync/atomic"
)

// World represents the central manager for all ECS state, handling entities, components, and hierarchical relationships.
type World struct {
	entities         map[EntityID]*Entity
	nextID           atomic.Uint64
	destroyed        []*Entity
	hierarchy        *Hierarchy
	archetypeManager *ArchetypeManager
	resources        map[reflect.Type]any
}

func NewWorld() *World {
	return &World{
		entities:         make(map[EntityID]*Entity),
		hierarchy:        NewHierarchy(),
		nextID:           atomic.Uint64{},
		destroyed:        make([]*Entity, 0),
		archetypeManager: NewArchetypeManager(),
		resources:        make(map[reflect.Type]any),
	}
}

// Reset clears the world entirely, removing all entities, components, and hierarchy relationships.
func (w *World) Reset() {
	w.entities = make(map[EntityID]*Entity)
	w.archetypeManager = NewArchetypeManager()
	w.hierarchy = NewHierarchy()
	w.nextID.Store(0)
	w.destroyed = make([]*Entity, 0)
}

// This method should be called after calling Destroy() to perform the actual removal.
func (w *World) Cleanup() {
	for _, entity := range w.destroyed {
		if entity == nil {
			continue
		}

		delete(w.entities, entity.ID)

		if parentID, hasParent := w.hierarchy.Parent(entity.ID); hasParent {
			w.hierarchy.Remove(parentID, entity.ID)
		}

		entity.template = ""
	}
	w.destroyed = make([]*Entity, 0)
}

// Count returns the number of active entities in the world.
func (w *World) Count() int {
	return len(w.entities)
}

// Create spawns a bare entity (no components) tagged with blueprintName.
// Use CreateWithComponents to spawn an entity with initial component data.
func (w *World) Create(blueprintName string) *Entity {
	archetype := w.archetypeManager.GetOrCreateArchetype()
	return w.createEntity(archetype, blueprintName)
}

func (w *World) CreateMany(blueprintName string, count int) []*Entity {
	entities := make([]*Entity, count)
	for i := range count {
		entities[i] = w.Create(blueprintName)
	}
	return entities
}

func (w *World) CreateWithComponents(blueprintName string, components ...Component) (*Entity, error) {
	componentTypes := make([]reflect.Type, len(components))
	for i, comp := range components {
		componentTypes[i] = reflect.TypeOf(comp)
	}
	archetype := w.archetypeManager.GetOrCreateArchetype(componentTypes...)
	entity := w.createEntity(archetype, blueprintName)

	for _, comp := range components {
		if err := w.updateComponentInArchetype(entity, comp, reflect.TypeOf(comp), archetype); err != nil {
			return nil, err
		}
	}
	return entity, nil
}

func (w *World) createEntity(archetype *Archetype, blueprintName string) *Entity {
	id := EntityID(w.nextID.Add(1))

	entity := NewEntity(id, blueprintName)
	w.entities[id] = entity

	entity.archetypeID = archetype.ID
	entity.archetypeIdx = len(archetype.entities)
	archetype.entities = append(archetype.entities, id)

	return entity
}

// DestroyEntity marks an entity for destruction. If cascade is true, all descendants of the entity will also be destroyed.
func (w *World) DestroyEntity(entityID EntityID, cascade bool) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "DestroyEntity",
			Err:      ErrEntityNotFound,
		}
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "DestroyEntity",
			Err:      ErrArchetypeNotFound,
		}
	}
	if movedID, moved := archetype.removeEntity(entity.archetypeIdx); moved {
		w.entities[movedID].archetypeIdx = entity.archetypeIdx
	}

	if cascade {
		// destroy all descendants of the entity
		descendants := w.hierarchy.Descendants(entityID)
		for _, descID := range descendants {
			descEntity, exists := w.entities[descID]
			if !exists {
				continue
			}
			descEntity.Destroy()
			w.destroyed = append(w.destroyed, descEntity)
		}
	} else {
		// just detach direct children of the entity from the hierarchy
		children := w.hierarchy.Children(entityID)
		for _, childID := range children {
			w.hierarchy.Remove(entityID, childID)
		}
	}

	entity.Destroy()
	w.destroyed = append(w.destroyed, entity)

	return nil
}

func (w *World) GetEntity(entityID EntityID) (*Entity, bool) {
	entity, exists := w.entities[entityID]
	return entity, exists
}

// HasEntity checks if an entity with the given EntityID exists in the world.
func (w *World) HasEntity(entityID EntityID) bool {
	_, exists := w.entities[entityID]
	return exists
}

// AddComponent adds a component to an entity. The entity must exist in the world.
func (w *World) AddComponent(entityID EntityID, comp ...Component) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return ErrEntityNotFound
	}
	currentArchetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)

	var toAdd []Component
	var newTypes []reflect.Type
	for _, c := range comp {
		compType := reflect.TypeOf(c)
		exists := false
		for _, t := range currentArchetype.componentTypes {
			if t == compType {
				// update the existing component in the current archetype
				if err := w.updateComponentInArchetype(entity, c, compType, currentArchetype); err != nil {
					return err
				}
				exists = true
				break
			}
		}
		if !exists {
			toAdd = append(toAdd, c)
			newTypes = append(newTypes, compType)
		}
	}

	if len(newTypes) == 0 {
		// No new component types to add; nothing to do.
		return nil
	}

	targetTypes := append(currentArchetype.componentTypes, newTypes...)
	// migrate entity to a new archetype that includes the new component type
	w.migrateEntityToNewArchetype(entity, toAdd, targetTypes)
	return nil
}

// RemoveComponent removes a component from an entity by type.
func (w *World) RemoveComponent[T Component](entityID EntityID) error {
	componentType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists {
		return ErrEntityNotFound
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "RemoveComponent",
			Err:      ErrArchetypeNotFound,
		}
	}

	found := slices.Contains(archetype.componentTypes, componentType)

	if !found {
		return nil // Component type not found in the entity's archetype; nothing to remove.
	}

	newComponentTypes := make([]reflect.Type, 0, len(archetype.componentTypes)-1)
	for _, t := range archetype.componentTypes {
		if t != componentType {
			newComponentTypes = append(newComponentTypes, t)
		}
	}

	// Migrate the entity to a new archetype without the removed component type.
	w.migrateEntityToNewArchetype(entity, nil, newComponentTypes)
	return nil
}

// GetComponent returns the first component of the specified type for an entity.
func (w *World) GetComponent[T Component](entityID EntityID) (T, error) {
	var zero T
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrArchetypeNotFound,
		}
	}

	// Check if the component type exists in the archetype
	found := slices.Contains(archetype.componentTypes, compType)

	if !found {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      fmt.Errorf(ErrComponentNotFound.Error(), compType.String()),
		}
	}

	rawSlice, exists := archetype.componentData[compType]
	if !exists {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	// Fast path: try direct type assertion to []T
	if typedSlice, ok := rawSlice.([]T); ok {
		if entity.archetypeIdx < len(typedSlice) {
			return typedSlice[entity.archetypeIdx], nil
		}
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	// Fallback: use reflection for generic access (shouldn't happen often)
	sliceVal := reflect.ValueOf(rawSlice)
	if sliceVal.Kind() != reflect.Slice || entity.archetypeIdx >= sliceVal.Len() {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	compVal := sliceVal.Index(entity.archetypeIdx)
	if !compVal.IsValid() {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	comp, ok := compVal.Interface().(T)
	if !ok {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      fmt.Errorf("component type mismatch: expected %T", zero),
		}
	}

	return comp, nil
}

func (w *World) HasComponent[T Component](entityID EntityID) bool {
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists {
		return false
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return false
	}

	return slices.Contains(archetype.componentTypes, compType)
}

func (w *World) SetComponent[T Component](entityID EntityID, newComp T) error {
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrArchetypeNotFound,
		}
	}

	// Check if the component type exists in the archetype
	found := slices.Contains(archetype.componentTypes, compType)
	if !found {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      fmt.Errorf(ErrComponentNotFound.Error(), compType.String()),
		}
	}

	rawSlice, exists := archetype.componentData[compType]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	// Fast path: try direct type assertion to []T
	if typedSlice, ok := rawSlice.([]T); ok {
		if entity.archetypeIdx < len(typedSlice) {
			typedSlice[entity.archetypeIdx] = newComp
			return nil
		}
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	// Fallback: use reflection for generic access (shouldn't happen often)
	sliceVal := reflect.ValueOf(rawSlice)
	if sliceVal.Kind() != reflect.Slice || entity.archetypeIdx >= sliceVal.Len() {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	// Set the new component value
	if sliceVal.Index(entity.archetypeIdx).CanSet() {
		compVal := reflect.ValueOf(newComp)
		if compVal.IsValid() && compVal.Type().AssignableTo(compType) {
			sliceVal.Index(entity.archetypeIdx).Set(compVal)
		} else if compVal.Type().ConvertibleTo(compType) {
			sliceVal.Index(entity.archetypeIdx).Set(compVal.Convert(compType))
		}
	}
	return nil
}

// Query retrieves all entities that have all the specified component types.
// Returns a slice of matching EntityIDs, or nil if none match.
// This uses superset matching - entities with AT LEAST the specified components.
// NewQuery starts a new lazy query builder. Use WithRequiredComponents, WithExcludedComponents,
// then Execute() to iterate, or All()/EntityIDs() to materialize results.
func (w *World) NewQuery() *Query {
	return NewQuery(w)
}

// Components returns all components associated with an entity.
func (w *World) Components(entityID EntityID) []Component {
	entity, exists := w.entities[entityID]
	if !exists {
		return nil
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return nil
	}

	components := make([]Component, 0, len(archetype.componentTypes))
	idx := entity.archetypeIdx
	for _, compType := range archetype.componentTypes {
		if rawSlice, exists := archetype.componentData[compType]; exists {
			// Use reflection to access the typed slice generically
			sliceVal := reflect.ValueOf(rawSlice)
			if sliceVal.Kind() == reflect.Slice && idx < sliceVal.Len() {
				compVal := sliceVal.Index(idx)
				if compVal.IsValid() {
					components = append(components, compVal.Interface())
				}
			}
		}
	}
	return components
}

// SetParent sets the parent of childID to parentID in the hierarchy.
func (w *World) SetParent(childID, parentID EntityID) {
	w.hierarchy.Add(parentID, childID)
}

// GetParent returns the parent EntityID of the given child entity, if one exists.
func (w *World) ParentOf(childID EntityID) (EntityID, bool) {
	return w.hierarchy.Parent(childID)
}

// GetChildren returns all direct children of the given parent entity.
func (w *World) ChildrenOf(parentID EntityID) []EntityID {
	return w.hierarchy.Children(parentID)
}

// Detach removes the parent-child relationship for the given entity.
func (w *World) Detach(id EntityID) {
	if parentID, hasParent := w.hierarchy.Parent(id); hasParent {
		w.hierarchy.Remove(parentID, id)
	}
}

func (w *World) migrateEntityToNewArchetype(entity *Entity, newComps []Component, newComponentTypes []reflect.Type) {
	newArchetype := w.archetypeManager.GetOrCreateArchetype(newComponentTypes...)

	// Copy existing components from current archetype before removing entity
	currentArchetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if exists {
		// Copy all existing components to new archetype
		for _, compType := range currentArchetype.componentTypes {
			if oldRawSlice, sliceExists := currentArchetype.componentData[compType]; sliceExists {
				// Use reflection to access the old typed slice
				oldSliceVal := reflect.ValueOf(oldRawSlice)
				if oldSliceVal.Kind() != reflect.Slice || entity.archetypeIdx >= oldSliceVal.Len() {
					continue
				}
				
				// Ensure new archetype has storage for this component type
				if _, newSliceExists := newArchetype.componentData[compType]; !newSliceExists {
					// Create a new typed slice of the same type
					newSlice := reflect.MakeSlice(oldSliceVal.Type(), 0, 0).Interface()
					newArchetype.componentData[compType] = newSlice
				}
				
				// Get or create the new typed slice
				newRawSlice := newArchetype.componentData[compType]
				newSliceVal := reflect.ValueOf(newRawSlice)
				if newSliceVal.Kind() != reflect.Slice {
					continue
				}
				
				// Ensure the new slice is large enough
				if entity.archetypeIdx >= newSliceVal.Len() {
					newLen := entity.archetypeIdx + 1
					if cap := newSliceVal.Cap(); cap < newLen {
						newCap := cap * 2
						if newCap < newLen {
							newCap = newLen
						}
						newSlice := reflect.MakeSlice(newSliceVal.Type(), newLen, newCap)
						reflect.Copy(newSlice, newSliceVal)
						newSliceVal = newSlice
					} else {
						newSliceVal = newSliceVal.Slice(0, newLen)
					}
					newArchetype.componentData[compType] = newSliceVal.Interface()
				}
				
				// Copy the component value from old slice to new slice
				if oldSliceVal.Index(entity.archetypeIdx).IsValid() && newSliceVal.Index(entity.archetypeIdx).CanSet() {
					newSliceVal.Index(entity.archetypeIdx).Set(oldSliceVal.Index(entity.archetypeIdx))
				}
			}
			// For each component type in the current archetype, copy the entity's value
			// to the same slot in the new archetype so it isn't lost during migration.
		}

		// Remove entity from current archetype, keeping component slices aligned.
		if movedID, moved := currentArchetype.removeEntity(entity.archetypeIdx); moved {
			w.entities[movedID].archetypeIdx = entity.archetypeIdx
		}
	}

	// Add entity to new archetype
	entity.archetypeID = newArchetype.ID
	entity.archetypeIdx = len(newArchetype.entities)
	newArchetype.entities = append(newArchetype.entities, entity.ID)

	// Store each newly-added component under its own type.
	for _, c := range newComps {
		w.setComponentInArchetype(newArchetype, entity.archetypeIdx, reflect.TypeOf(c), c)
	}
}

func (w *World) updateComponentInArchetype(entity *Entity, comp Component, compType reflect.Type, archetype *Archetype) error {
	if archetype == nil {
		var exists bool
		archetype, exists = w.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if !exists {
			return &EntityError{
				EntityID: entity.ID,
				Op:       "UpdateComponentInArchetype",
				Err:      ErrArchetypeNotFound,
			}
		}
	}
	w.setComponentInArchetype(archetype, entity.archetypeIdx, compType, comp)
	return nil
}

func (w *World) setComponentInArchetype(archetype *Archetype, index int, compType reflect.Type, comp Component) {
	// Get or create the typed slice for this component type
	rawSlice, exists := archetype.componentData[compType]
	
	if !exists {
		// Create a new typed slice with the component's type
		// We use reflection to create a slice of the correct type
		if compType == nil {
			return
		}
		sliceType := reflect.SliceOf(compType)
		rawSlice = reflect.MakeSlice(sliceType, 0, 0).Interface()
		archetype.componentData[compType] = rawSlice
	}
	
	// Use reflection to work with the typed slice generically
	sliceVal := reflect.ValueOf(rawSlice)
	if sliceVal.Kind() != reflect.Slice {
		return
	}
	
	// Ensure the slice is large enough
	if index >= sliceVal.Len() {
		// Grow the slice to accommodate the index
		newLen := index + 1
		if cap := sliceVal.Cap(); cap < newLen {
			newCap := cap * 2
			if newCap < newLen {
				newCap = newLen
			}
			newSlice := reflect.MakeSlice(sliceVal.Type(), newLen, newCap)
			reflect.Copy(newSlice, sliceVal)
			sliceVal = newSlice
		} else {
			sliceVal = sliceVal.Slice(0, newLen)
		}
		archetype.componentData[compType] = sliceVal.Interface()
	}
	
	// Set the component at the specified index
	// We need to use a settable reflection value
	if sliceVal.Index(index).CanSet() {
		// Try to set directly if types match
		compVal := reflect.ValueOf(comp)
		if compVal.IsValid() && compVal.Type().AssignableTo(compType) {
			sliceVal.Index(index).Set(compVal)
		} else if compVal.Type().ConvertibleTo(compType) {
			sliceVal.Index(index).Set(compVal.Convert(compType))
		}
	}
}
