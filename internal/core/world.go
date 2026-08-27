package core

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

// World represents the central manager for all ECS state, handling entities, components, tags, templates, and hierarchical relationships.
type World struct {
	entities         map[EntityID]*entity
	nextID           atomic.Uint64
	destroyed        []*entity
	hierarchy        *Hierarchy
	archetypeManager *ArchetypeManager
	index            entityIndex
}

func NewWorld() *World {
	return &World{
		entities:         make(map[EntityID]*entity),
		index:            NewEntityIndex(),
		hierarchy:        NewHierarchy(),
		nextID:           atomic.Uint64{},
		destroyed:        make([]*entity, 0),
		archetypeManager: NewArchetypeManager(),
	}
}

// CreateEntity creates a new entity with the specified template and returns its EntityID.
// The entity is automatically registered with the world and the index.
func (w *World) CreateEntity(template string) EntityID {
	id := EntityID(w.nextID.Add(1))

	entity := NewEntity(id, template)
	w.entities[id] = entity

	emptyArchtype := w.archetypeManager.GetOrCreateArchetype()
	entity.archetypeID = emptyArchtype.ID
	entity.archetypeIdx = len(emptyArchtype.entities)
	emptyArchtype.entities = append(emptyArchtype.entities, id)

	// Defer tag/template index maintenance until the first query or tag mutation.
	// This keeps entity creation cheap while preserving correctness when lookup indexes are needed.
	w.index.lazyTagTemplateIndex = true

	// NOTE: two temporary templates for testing - Province and City
	switch template {
	case "Province":
		entity.AddTag("Province")
	case "City":
		entity.AddTag("City")
	default:
		entity.AddTag("Generic")
	}

	return id
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
	archetype.entities = append(archetype.entities[:entity.archetypeIdx], archetype.entities[entity.archetypeIdx+1:]...)
	// Update the indices of entities in the archetype that were after the removed entity
	for i := entity.archetypeIdx; i < len(archetype.entities); i++ {
		w.entities[archetype.entities[i]].archetypeIdx = i
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

func (w *World) GetEntity(entityID EntityID) (*entity, bool) {
	entity, exists := w.entities[entityID]
	return entity, exists
}

// HasEntity checks if an entity with the given EntityID exists in the world.
func (w *World) HasEntity(entityID EntityID) bool {
	_, exists := w.entities[entityID]
	return exists
}

// Query retrieves all entities that have all the specified component types.
// Returns a slice of matching EntityIDs, or nil if none match.
// This uses superset matching - entities with AT LEAST the specified components.
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

	// This reads as:
	// For each component type, check all archetypes to see if they contain that component type.
	// If they do, add all entities from that archetype to the results, ensuring no duplicates.
	// This is a brute-force approach and can be optimized with a better indexing strategy in the future.
	// We'll need to consider how to efficiently handle queries for "any" component type, especially as the number of archetypes grows.
	for _, compType := range components {
		for _, archetype := range w.archetypeManager.archetypes {
			for _, t := range archetype.componentTypes {
				if t == compType {
					for _, entityID := range archetype.entities {
						if !seen[entityID] {
							results = append(results, entityID)
							seen[entityID] = true
						}
					}
					break
				}
			}
		}
	}

	return results
}

func (w *World) QuerySuperset(components ...reflect.Type) []EntityID {
	if len(components) == 0 {
		return nil
	}

	querykey := NewArchetypeKey(components...)
	var results []EntityID

	for _, archetype := range w.archetypeManager.archetypes {
		if archetype.componentTypes.ContainsAll(querykey) {
			results = append(results, archetype.entities...)
		}
	}

	return results
}

// QueryByTag retrieves all entities that have the specified tag.
// Returns a slice of matching EntityIDs, or nil if none match.
func (w *World) QueryByTag(tag string) []EntityID {
	w.ensureTagAndTemplateIndex()
	entityIDs := w.index.GetEntitiesWithTag(tag)
	return entityIDs
}

// QueryByTemplate retrieves all entities that use the specified template.
// Returns a slice of matching EntityIDs, or nil if none match.
func (w *World) QueryByTemplate(template string) []EntityID {
	w.ensureTagAndTemplateIndex()
	entityIDs := w.index.GetEntitiesWithTemplate(template)
	return entityIDs
}

// AddTag adds a tag to an entity. The entity must exist in the world.
func (w *World) AddTag(entityID EntityID, tag string) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "AddTag",
			Err:      ErrEntityNotFound,
		}
	}
	entity.AddTag(tag)
	if w.index.lazyTagTemplateIndex {
		w.index.rebuildTagAndTemplateIndex(w.entities)
	} else {
		w.index.AddTag(entityID, tag)
	}
	return nil
}

// RemoveTag removes a tag from an entity. The entity must exist in the world.
func (w *World) RemoveTag(entityID EntityID, tag string) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "RemoveTag",
			Err:      ErrEntityNotFound,
		}
	}

	entity.RemoveTag(tag)
	if w.index.lazyTagTemplateIndex {
		w.index.rebuildTagAndTemplateIndex(w.entities)
	} else {
		w.index.RemoveTag(entityID, tag)
	}
	return nil
}

// HasTag checks if an entity has a specific tag. The entity must exist in the world.
func (w *World) HasTag(entityID EntityID, tag string) (bool, error) {
	entity, exists := w.entities[entityID]
	if !exists {
		return false, &EntityError{
			EntityID: entityID,
			Op:       "HasTag",
			Err:      ErrEntityNotFound,
		}
	}
	return entity.HasTag(tag), nil
}

// AddComponent adds a component to an entity. The entity must exist in the world.
func (w *World) AddComponent(entityID EntityID, comp Component) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return ErrEntityNotFound
	}

	compType := reflect.TypeOf(comp)

	currentArchetype, exists := w.archetypeManager.GetArchetypeByID(entity.archetypeID)

	if exists {
		for _, t := range currentArchetype.componentTypes {
			if t == compType {
				// Component type already exists in the current archetype; no need to change archetype.
				return w.updateComponentInArchetype(entity, comp, compType, currentArchetype)
			}
		}
	}

	// migrate entity to a new archetype that includes the new component type
	return w.migrateEntityToNewArchetype(entity, comp, append(currentArchetype.componentTypes, compType)...)

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

	return nil // silently ignore if component type not found
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
	found := false
	for _, t := range archetype.componentTypes {
		if t == compType {
			found = true
			break
		}
	}

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

// Cleanup removes all destroyed entities and clears their associated data from the store, index, and hierarchy.
// This method should be called after calling Destroy() to perform the actual removal.
func (w *World) Cleanup() {
	for _, entity := range w.destroyed {
		if entity == nil {
			continue
		}

		delete(w.entities, entity.ID())
		w.index.RemoveTemplate(entity.ID(), entity.Template())
		for tag := range entity.tags {
			w.index.RemoveTag(entity.ID(), tag)
		}

		if parentID, hasParent := w.hierarchy.Parent(entity.ID()); hasParent {
			w.hierarchy.Remove(parentID, entity.ID())
		}

		entity.tags = nil
		entity.template = ""
	}
	w.destroyed = w.destroyed[:0]
}

// Reset clears the world entirely, removing all entities, components, and hierarchy relationships.
func (w *World) Reset() {
	w.entities = make(map[EntityID]*entity)
	w.archetypeManager = NewArchetypeManager()
	w.index = NewEntityIndex()
	w.hierarchy = NewHierarchy()
	w.nextID.Store(0)
	w.destroyed = make([]*entity, 0)
}

// Count returns the number of active entities in the world.
func (w *World) Count() int {
	return len(w.entities)
}

func (w *World) migrateEntityToNewArchetype(entity *entity, newComp Component, newComponentTypes ...reflect.Type) error {
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
			// The above logic can be read as: for each component type in the current archetype, if the entity has that component, copy it to the new archetype at the same index. This ensures that when we migrate the entity, it retains all its existing components in the new archetype.
			// This does result in an array where the entity's index is preserved, but it may leave gaps in the component slices for other entities. This is acceptable as long as we maintain the correct index for each entity in its respective archetype.
			// We could optimize this further by compacting the component slices after migration, but for now, this approach ensures correctness and simplicity.
		}

		// Remove entity from current archetype
		currentArchetype.entities = append(currentArchetype.entities[:entity.archetypeIdx], currentArchetype.entities[entity.archetypeIdx+1:]...)
		// Update the indices of entities in the current archetype that were after the removed entity
		for i := entity.archetypeIdx; i < len(currentArchetype.entities); i++ {
			w.entities[currentArchetype.entities[i]].archetypeIdx = i
		}
	}

	// Add entity to new archetype
	entity.archetypeID = newArchetype.ID
	entity.archetypeIdx = len(newArchetype.entities)
	newArchetype.entities = append(newArchetype.entities, entity.id)

	// Set the new Component if provided
	if newComp != nil {
		w.setComponentInArchetype(newArchetype, entity.archetypeIdx, reflect.TypeOf(newComp), newComp)
	}

	return nil
}

func (w *World) updateComponentInArchetype(entity *entity, comp Component, compType reflect.Type, archetype *Archetype) error {
	// if no archetype passed in, try to get the current archetype of the entity
	if archetype == nil {
		var exists bool
		archetype, exists = w.archetypeManager.GetArchetypeByID(entity.archetypeID)
		if !exists {
			return &EntityError{
				EntityID: entity.ID(),
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

func (w *World) ensureTagAndTemplateIndex() {
	if !w.index.lazyTagTemplateIndex {
		return
	}
	w.index.rebuildTagAndTemplateIndex(w.entities)
}
