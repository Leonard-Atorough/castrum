package ecs

import "sync/atomic"

type World struct {
	entities  map[EntityID]*entity
	store     *componentStore
	index     entityIndex
	hierarchy *Hierarchy
	nextID    atomic.Uint64
}

func NewWorld() *World {
	return &World{
		entities:  make(map[EntityID]*entity),
		store:     NewComponentStore(),
		index:     NewEntityIndex(),
		hierarchy: NewHierarchy(),
	}
}

//Entity methods

// Spawn takes in a template and
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

func (w *World) Destroy(entityID EntityID) {
	entity, exists := w.entities[entityID]
	if !exists {
		return
	}
	// Remove all components associated with the entity
	w.store.RemoveAll(entityID)
	// Remove the entity from the index
	w.index.RemoveTemplate(entityID, entity.Template())
	for tag := range entity.tags {
		w.index.RemoveTag(entityID, tag)
	}
	// mark the entity as destroyed
	entity.Destroy()
	// Entity is still in the map but marked as destroyed. It will be deleted in the next cleanup cycle.
}

// Query methods

// Store access methods

// Hierarchy access methods

// Lifecycle methods

// General utility methods
