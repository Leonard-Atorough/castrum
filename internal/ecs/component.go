package ecs

import (
	"fmt"

	"github.com/leonard-atorough/castrum/pkg/component"
)

type componentStore struct {
	// data maps entity IDs to a slice of components associated with that entity.
	data map[uint64][]component.Component
}

func NewComponentStore() *componentStore {
	return &componentStore{
		data: make(map[uint64][]component.Component),
	}
}

// Set adds a component for a specific entity ID.
func (cs *componentStore) Set(entityID uint64, comp component.Component) {
	cs.data[entityID] = append(cs.data[entityID], comp)
}

// GetAll returns all components attached to an entity.
func (cs *componentStore) GetAll(entityID uint64) []component.Component {
	return cs.data[entityID]
}

// Get returns the first component of concrete type T for the entity.
func Get[T component.Component](cs *componentStore, entityID uint64) (*T, error) {
	components, exists := cs.data[entityID]
	if !exists || len(components) == 0 {
		return nil, fmt.Errorf("no components found for entity ID %d", entityID)
	}

	for _, c := range components {
		if typed, ok := c.(T); ok {
			return &typed, nil
		}
	}

	var zero T
	return nil, fmt.Errorf("no component of type %T found for entity ID %d", zero, entityID)
}

func (cs *componentStore) RemoveAll(entityID uint64) {
	delete(cs.data, entityID)
}

func (cs *componentStore) Remove(entityID uint64, comp component.Component) {
	components, exists := cs.data[entityID]
	if !exists {
		return
	}

	for i, c := range components {
		if c == comp {
			cs.data[entityID] = append(components[:i], components[i+1:]...)
			break
		}
	}
}
