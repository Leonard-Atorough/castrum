package ecs

import (
	"reflect"
	"sync/atomic"

	"github.com/leonard-atorough/castrum/pkg/component"
)

type World struct {
	entities  map[EntityID]*entity
	store     *componentStore
	index     entityIndex
	hierarchy *Hierarchy
	nextID    atomic.Uint64

	destroyed []*entity
}

func NewWorld() *World {
	return &World{
		entities:  make(map[EntityID]*entity),
		store:     NewComponentStore(),
		index:     NewEntityIndex(),
		hierarchy: NewHierarchy(),
	}
}

// Spawn creates a new entity with the specified template and returns its unique EntityID.
func (w *World) Spawn(template string) EntityID {
	id := EntityID(w.nextID.Add(1))

	entity := NewEntity(id, template)
	w.entities[id] = entity

	w.index.AddTemplate(id, template)

	// two temporary templates for testing - Province and City
	switch template {
	case "Province":
		entity.AddTag("Province")
		w.index.AddTag(id, "Province")
	case "City":
		entity.AddTag("City")
		w.index.AddTag(id, "City")
	default:
		entity.AddTag("Generic")
		w.index.AddTag(id, "Generic")
	}

	return id
}

// Destroy marks an entity for destruction. If cascade is true, all descendants of the entity will also be destroyed.
func (w *World) Destroy(entityID EntityID, cascade bool) {
	entity, exists := w.entities[entityID]
	if !exists {
		return
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
}

// GetEntity retrieves an entity by its EntityID. Returns the entity and a boolean indicating if it exists.
func (w *World) GetEntity(entityID EntityID) (*entity, bool) {
	entity, exists := w.entities[entityID]
	return entity, exists
}

// Query methods

// Lifecycle methods

// AddComponent adds a component to an entity. The entity must exist in the world.
func (w *World) AddComponent(entityID EntityID, comp component.Component) error {
	if _, exists := w.entities[entityID]; !exists {
		return ErrEntityNotFound
	}
	w.store.Set(entityID, comp)
	w.index.AddComponent(entityID, reflect.TypeOf(comp))
	return nil
}

// RemoveComponent removes a component from an entity by type.
func (w *World) RemoveComponent(entityID EntityID, componentType reflect.Type) error {
	if _, exists := w.entities[entityID]; !exists {
		return ErrEntityNotFound
	}

	// Get all components and find the one matching the type
	components := w.store.GetAll(entityID)
	for _, comp := range components {
		if reflect.TypeOf(comp) == componentType {
			w.store.Remove(entityID, comp)
			w.index.RemoveComponent(entityID, componentType)
			return nil
		}
	}

	return nil // silently ignore if component type not found
}

// SetParent sets the parent of childID to parentID in the hierarchy.
func (w *World) SetParent(childID, parentID EntityID) {
	w.hierarchy.Add(parentID, childID)
}

// GetParent returns the parent of childID, if any.
func (w *World) GetParent(childID EntityID) (EntityID, bool) {
	return w.hierarchy.Parent(childID)
}

// GetChildren returns all direct children of parentID.
func (w *World) GetChildren(parentID EntityID) []EntityID {
	return w.hierarchy.Children(parentID)
}

// Detach removes the parent-child relationship for the given entity ID.
func (w *World) Detach(id EntityID) {
	if parentID, hasParent := w.hierarchy.Parent(id); hasParent {
		w.hierarchy.Remove(parentID, id)
	}
}

// Cleanup removes all entities marked for destruction and their associated components, tags, and hierarchy relationships.
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

// Reset clears the world, removing all entities, components, and hierarchy relationships.
func (w *World) Reset() {
	w.entities = make(map[EntityID]*entity)
	w.store = NewComponentStore()
	w.index = NewEntityIndex()
	w.hierarchy = NewHierarchy()
	w.nextID.Store(0)
	w.destroyed = nil
}

// Returns the number of active entities in the world.
func (w *World) Count() int {
	return len(w.entities)
}

// General utility methods
