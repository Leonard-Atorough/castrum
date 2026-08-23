package ecs

import (
	"fmt"
	"reflect"

	"github.com/leonard-atorough/castrum/pkg/component"
)

type componentStore struct {
	// data maps entity IDs to a slice of components associated with that entity.
	data map[EntityID]map[reflect.Type]component.Component
}

func NewComponentStore() *componentStore {
	return &componentStore{
		data: make(map[EntityID]map[reflect.Type]component.Component),
	}
}

// Set adds a component for a specific entity ID.
func (cs *componentStore) Set(entityID EntityID, comp component.Component) {
	if cs.data[entityID] == nil {
		cs.data[entityID] = make(map[reflect.Type]component.Component)
	}
	cs.data[entityID][reflect.TypeOf(comp)] = comp
}

// GetAll returns all components attached to an entity.
func (cs *componentStore) GetAll(entityID EntityID) []component.Component {
	componentsMap := cs.data[entityID]
	components := make([]component.Component, 0, len(componentsMap))
	for _, comp := range componentsMap {
		components = append(components, comp)
	}
	return components
}

// Get returns the first component of the specified type for the entity.
func (cs *componentStore) Get(entityID EntityID, compType reflect.Type) (component.Component, error) {
	componentsMap, exists := cs.data[entityID]
	if !exists || len(componentsMap) == 0 {
		return nil, fmt.Errorf("no components found for entity ID %d", entityID)
	}

	comp, exists := componentsMap[compType]
	if !exists {
		return nil, fmt.Errorf("no component of type %v found for entity ID %d", compType, entityID)
	}
	return comp, nil
}

func (cs *componentStore) GetByString(entityID EntityID, compName string) (component.Component, error) {
	componentsMap, exists := cs.data[entityID]
	if !exists || len(componentsMap) == 0 {
		return nil, fmt.Errorf("no components found for entity ID %d", entityID)
	}

	for _, comp := range componentsMap {
		if comp.Name() == compName {
			return comp, nil
		}
	}
	return nil, fmt.Errorf("no component with name %s found for entity ID %d", compName, entityID)
}

func (cs *componentStore) RemoveAll(entityID EntityID) {
	delete(cs.data, entityID)
}

func (cs *componentStore) Remove(entityID EntityID, comp component.Component) {
	components, exists := cs.data[entityID]
	if !exists {
		return
	}

	compType := reflect.TypeOf(comp)
	delete(components, compType)
}
