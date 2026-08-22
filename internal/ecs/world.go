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
		w.index.AddTag(id, "Province")
	case "City":
		w.index.AddTag(id, "City")
	default:
		w.index.AddTag(id, "Generic")
	}

	return id
}

// Query methods

// Store access methods

// Hierarchy access methods

// Lifecycle methods

// General utility methods
