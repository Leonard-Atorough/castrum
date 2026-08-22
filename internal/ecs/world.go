package ecs

import "sync/atomic"

type World struct {
	entities  map[uint64]*entity // map EntityID type later when introduced in public API
	store     *componentStore
	index     entityIndex
	hierarchy *Hierarchy
	nextID    atomic.Uint64
}

func NewWorld() *World {
	return &World{
		entities:  make(map[uint64]*entity),
		store:     NewComponentStore(),
		index:     NewEntityIndex(),
		hierarchy: NewHierarchy(),
	}
}

//General methods

// Store access methods

// Hierarchy access methods

// Lifecycle methods
