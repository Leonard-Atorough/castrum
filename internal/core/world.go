package core

import (
	"reflect"
	"sync/atomic"

	"github.com/leonard-atorough/castrum/ecs"
)

// World is the central manager for all ECS state.
// It manages entity lifecycle, components, tags, templates, and hierarchical relationships.
// All public interactions with the ECS should go through the World API.
type World struct {
	entities  map[ecs.EntityID]*entity
	store     *componentStore
	index     entityIndex
	hierarchy *Hierarchy
	nextID    atomic.Uint64
	destroyed []*entity
}

// NewWorld creates and returns a new empty World.
func NewWorld() *World {
	return &World{
		entities:  make(map[ecs.EntityID]*entity),
		store:     NewComponentStore(),
		index:     NewEntityIndex(),
		hierarchy: NewHierarchy(),
		nextID:    atomic.Uint64{},
		destroyed: make([]*entity, 0),
	}
}

// Create creates a new entity with the specified template and returns its ecs.EntityID.
// The entity is automatically registered with the world and the index.
func (w *World) Create(template string) ecs.EntityID {
	id := ecs.EntityID(w.nextID.Add(1))

	entity := NewEntity(id, template)
	w.entities[id] = entity

	// Defer tag/template index maintenance until the first query or tag mutation.
	// This keeps entity creation cheap while preserving correctness when lookup indexes are needed.
	w.index.lazyTagTemplateIndex = true

	// two temporary templates for testing - Province and City
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

// Destroy marks an entity for destruction. If cascade is true, all descendants of the entity will also be destroyed.
func (w *World) Destroy(entityID ecs.EntityID, cascade bool) error {
	entity, exists := w.entities[entityID]
	if !exists {
		return &EntityError{
			EntityID: entityID,
			Op:       "Destroy",
			Err:      ErrEntityNotFound,
		}
	}
	entity.Destroy()
	w.destroyed = append(w.destroyed, entity)

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
	return nil
}

func (w *World) GetEntity(entityID ecs.EntityID) (*entity, bool) {
	entity, exists := w.entities[entityID]
	return entity, exists
}

// Exists checks if an entity with the given ecs.EntityID exists in the world.
func (w *World) Exists(entityID ecs.EntityID) bool {
	_, exists := w.entities[entityID]
	return exists
}

// Query retrieves all entities that have all the specified component types.
// Returns a slice of matching EntityIDs, or nil if none match.
func (w *World) Query(components ...reflect.Type) []ecs.EntityID {
	return w.index.GetEntitiesWithComponents(components...)
}

// QueryByTag retrieves all entities that have the specified tag.
// Returns a slice of matching EntityIDs, or nil if none match.
func (w *World) QueryByTag(tag string) []ecs.EntityID {
	w.ensureTagAndTemplateIndex()
	return w.index.GetEntitiesWithTag(tag)
}

// QueryByTemplate retrieves all entities that use the specified template.
// Returns a slice of matching EntityIDs, or nil if none match.
func (w *World) QueryByTemplate(template string) []ecs.EntityID {
	w.ensureTagAndTemplateIndex()
	return w.index.GetEntitiesWithTemplate(template)
}

// AddTag adds a tag to an entity. The entity must exist in the world.
func (w *World) AddTag(entityID ecs.EntityID, tag string) error {
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
func (w *World) RemoveTag(entityID ecs.EntityID, tag string) error {
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
func (w *World) HasTag(entityID ecs.EntityID, tag string) (bool, error) {
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

func (w *World) ensureTagAndTemplateIndex() {
	if !w.index.lazyTagTemplateIndex {
		return
	}
	w.index.rebuildTagAndTemplateIndex(w.entities)
}

// AddComponent adds a component to an entity. The entity must exist in the world.
func (w *World) AddComponent(entityID ecs.EntityID, comp ecs.Component) error {
	if _, exists := w.entities[entityID]; !exists {
		return ErrEntityNotFound
	}
	w.store.Set(entityID, comp)
	w.index.AddComponent(entityID, reflect.TypeOf(comp))
	return nil
}

// RemoveComponent removes a component from an entity by type.
func (w *World) RemoveComponent(entityID ecs.EntityID, componentType reflect.Type) error {
	if _, exists := w.entities[entityID]; !exists {
		return ErrEntityNotFound
	}

	// Get all components and find the one matching the type
	component, err := w.store.Get(entityID, componentType)
	if err == nil {
		w.store.Remove(entityID, component)
		w.index.RemoveComponent(entityID, componentType)
	}

	return nil // silently ignore if component type not found
}

// GetComponent returns the first component of the specified type for an entity.
func (w *World) GetComponent(entityID ecs.EntityID, compType reflect.Type) (ecs.Component, error) {
	comp, err := w.store.Get(entityID, compType)
	if err != nil {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      err,
		}
	}
	return comp, nil
}

func (w *World) GetComponentByName(entityID ecs.EntityID, compName string) (ecs.Component, error) {
	comp, err := w.store.GetByString(entityID, compName)
	if err != nil {
		return nil, &EntityError{
			EntityID: entityID,
			Op:       "GetComponentByName",
			Err:      err,
		}
	}
	return comp, nil
}

func (w *World) Components(entityID ecs.EntityID) []ecs.Component {
	return w.store.GetAll(entityID)
}

// HasComponent checks whether an entity has a component of the specified type.
func (w *World) HasComponent(entityID ecs.EntityID, compType reflect.Type) bool {
	_, err := w.store.Get(entityID, compType)
	return err == nil
}

// SetParent sets the parent of childID to parentID in the hierarchy.
func (w *World) SetParent(childID, parentID ecs.EntityID) {
	w.hierarchy.Add(parentID, childID)
}

// GetParent returns the parent ecs.EntityID of the given child entity, if one exists.
func (w *World) ParentOf(childID ecs.EntityID) (ecs.EntityID, bool) {
	return w.hierarchy.Parent(childID)
}

// GetChildren returns all direct children of the given parent entity.
func (w *World) ChildrenOf(parentID ecs.EntityID) []ecs.EntityID {
	return w.hierarchy.Children(parentID)
}

// Detach removes the parent-child relationship for the given entity.
func (w *World) Detach(id ecs.EntityID) {
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
		components := w.store.GetAll(entity.ID())
		for _, comp := range components {
			w.index.RemoveComponent(entity.ID(), reflect.TypeOf(comp))
		}

		w.store.RemoveAll(entity.ID())

		if parentID, hasParent := w.hierarchy.Parent(entity.ID()); hasParent {
			w.hierarchy.Remove(parentID, entity.ID())
		}
	}
	w.destroyed = w.destroyed[:0]
}

// Reset clears the world entirely, removing all entities, components, and hierarchy relationships.
func (w *World) Reset() {
	w.entities = make(map[ecs.EntityID]*entity)
	w.store = NewComponentStore()
	w.index = NewEntityIndex()
	w.hierarchy = NewHierarchy()
	w.nextID.Store(0)
	w.destroyed = nil
}

// Count returns the number of active entities in the world.
func (w *World) Count() int {
	return len(w.entities)
}

// General utility methods
