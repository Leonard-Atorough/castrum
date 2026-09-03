package core

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
}

func NewWorld() *World {
	return &World{
		entities:         make(map[EntityID]*Entity),
		hierarchy:        NewHierarchy(),
		nextID:           atomic.Uint64{},
		destroyed:        make([]*Entity, 0),
		archetypeManager: NewArchetypeManager(),
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
	for i := 0; i < count; i++ {
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
	return w.migrateEntityToNewArchetype(entity, toAdd, targetTypes...)
}

// RemoveComponent removes a component from an entity by type.
func (w *World) RemoveComponent(entityID EntityID, componentType reflect.Type) error {
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

	found := false
	for _, t := range archetype.componentTypes {
		if t == componentType {
			found = true
			break
		}
	}

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
	_ = w.migrateEntityToNewArchetype(entity, nil, newComponentTypes...)
	return nil
}

// GetComponent returns the first component of the specified type for an entity.
func (w *World) GetComponent(entityID EntityID, compType reflect.Type) (Component, error) {
	entity, exists := w.entities[entityID]
	if !exists {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrArchetypeNotFound,
		}
	}

	// Check if the component type exists in the archetype
	found := slices.Contains(archetype.componentTypes, compType)

	if !found {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      fmt.Errorf(ErrComponentNotFound.Error(), compType.String()),
		}
	}

	slice, exists := archetype.componentData[compType]
	if !exists || entity.archetypeIdx >= len(slice.([]Component)) {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	return slice.([]Component)[entity.archetypeIdx], nil
}

func (w *World) HasComponent(entityID EntityID, compType reflect.Type) bool {
	entity, exists := w.entities[entityID]
	if !exists {
		return false
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return false
	}

	for _, t := range archetype.componentTypes {
		if t == compType {
			return true
		}
	}
	return false
}

func (w *World) SetComponent(entityID EntityID, compType reflect.Type, newComp Component) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "UpdateComponent",
			Err:      ErrEntityNotFound,
		}
	}

	archetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "UpdateComponent",
			Err:      ErrArchetypeNotFound,
		}
	}

	// Check if the component type exists in the archetype
	found := slices.Contains(archetype.componentTypes, compType)
	if !found {
		return &EntityError{
			EntityID: entityID,
			Op:       "UpdateComponent",
			Err:      fmt.Errorf(ErrComponentNotFound.Error(), compType.String()),
		}
	}

	slice, exists := archetype.componentData[compType]
	if !exists || entity.archetypeIdx >= len(slice.([]Component)) {
		return &EntityError{
			EntityID: entityID,
			Op:       "UpdateComponent",
			Err:      ErrEntityNotFound,
		}
	}

	slice.([]Component)[entity.archetypeIdx] = newComp
	return nil
}

// Query retrieves all entities that have all the specified component types.
// Returns a slice of matching EntityIDs, or nil if none match.
// This uses superset matching - entities with AT LEAST the specified components.
//NOTE: Big query improvements coming to support query iterator generation and on demand fetch, rather than bulk returns
// NOTE: Query package with query builder pattern to support more complex queries.
func (w *World) Query(components ...reflect.Type) []EntityID {
	if len(components) == 0 {
		return nil
	}

	// Use superset matching to find all entities that have at least these components
	queryKey := NewArchetypeKey(components...)
	var result []EntityID

	for _, archetype := range w.archetypeManager.archetypes {
		if archetype.componentTypes.ContainsAll(queryKey) {
			result = append(result, archetype.entities...)
		}
	}

	return result
}

func (w *World) QueryAny(components ...reflect.Type) []EntityID {
	if len(components) == 0 {
		return nil
	}

	var results []EntityID
	seen := make(map[EntityID]bool)

	for _, compType := range components {
		for _, archetype := range w.archetypeManager.archetypes {
			if slices.Contains(archetype.componentTypes, compType) {
					for _, entityID := range archetype.entities {
						if !seen[entityID] {
							results = append(results, entityID)
							seen[entityID] = true
						}
					}
				}
		}
	}

	return results
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
	for _, compType := range archetype.componentTypes {
		if slice, exists := archetype.componentData[compType]; exists {
			compSlice := slice.([]Component)
			if entity.archetypeIdx < len(compSlice) {
				components = append(components, compSlice[entity.archetypeIdx])
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

func (w *World) migrateEntityToNewArchetype(entity *Entity, newComps []Component, newComponentTypes ...reflect.Type) error {
	newArchetype := w.archetypeManager.GetOrCreateArchetype(newComponentTypes...)

	// Copy existing components from current archetype before removing entity
	currentArchetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)
	if exists {
		// Copy all existing components to new archetype
		for _, compType := range currentArchetype.componentTypes {
			if slice, sliceExists := currentArchetype.componentData[compType]; sliceExists {
				compSlice := slice.([]Component)
				if entity.archetypeIdx < len(compSlice) {
					// Ensure new archetype has storage for this component type
					if _, newSliceExists := newArchetype.componentData[compType]; !newSliceExists {
						newArchetype.componentData[compType] = make([]Component, 0)
					}
					// Add component to new archetype
					newSlice := newArchetype.componentData[compType].([]Component)
					if len(newSlice) <= entity.archetypeIdx {
						newSlice = append(newSlice, make([]Component, entity.archetypeIdx-len(newSlice)+1)...)
						newArchetype.componentData[compType] = newSlice
					}
					newSlice[entity.archetypeIdx] = compSlice[entity.archetypeIdx]
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

	return nil
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
	if _, exists := archetype.componentData[compType]; !exists {
		archetype.componentData[compType] = make([]Component, len(archetype.entities))
	}
	compSlice := archetype.componentData[compType].([]Component)

	if index >= len(compSlice) {
		newSlice := make([]Component, len(archetype.entities))
		copy(newSlice, compSlice)
		compSlice = newSlice
		archetype.componentData[compType] = compSlice
	}
	compSlice[index] = comp
}

func Types(comps ...Component) []reflect.Type {
	var componentTypes []reflect.Type
	for _, comp := range comps {
		componentTypes = append(componentTypes, reflect.TypeOf(comp))
	}
	return componentTypes
}
