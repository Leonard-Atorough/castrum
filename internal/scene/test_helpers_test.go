package scene

import (
	"errors"
	"reflect"

	"github.com/leonard-atorough/castrum/ecs"
)

// mockWorld is a minimal implementation of ecs.World for unit testing
type mockWorld struct {
	entities map[ecs.EntityID]map[string]bool
	nextID   ecs.EntityID
}

func newMockWorld() *mockWorld {
	return &mockWorld{
		entities: make(map[ecs.EntityID]map[string]bool),
		nextID:   1,
	}
}

func (m *mockWorld) CreateEntity(template string) ecs.EntityID {
	id := m.nextID
	m.nextID++
	m.entities[id] = make(map[string]bool)
	return id
}

func (m *mockWorld) DestroyEntity(id ecs.EntityID, cascade bool) error {
	delete(m.entities, id)
	return nil
}

func (m *mockWorld) Query(components ...reflect.Type) []ecs.EntityID {
	return nil
}

func (m *mockWorld) QueryByTag(tag string) []ecs.EntityID {
	var result []ecs.EntityID
	for id, tags := range m.entities {
		if tags[tag] {
			result = append(result, id)
		}
	}
	return result
}

func (m *mockWorld) AddComponent(id ecs.EntityID, component ecs.Component) error {
	return nil
}

func (m *mockWorld) RemoveComponent(id ecs.EntityID, componentType reflect.Type) error {
	return nil
}

func (m *mockWorld) GetComponent(id ecs.EntityID, componentType reflect.Type) (ecs.Component, error) {
	return nil, nil
}

func (m *mockWorld) HasComponent(id ecs.EntityID, componentType reflect.Type) bool {
	return false
}

func (m *mockWorld) Components(id ecs.EntityID) []ecs.Component {
	return nil
}

func (m *mockWorld) AddTag(id ecs.EntityID, tag string) error {
	if _, exists := m.entities[id]; !exists {
		return errors.New("entity not found")
	}
	m.entities[id][tag] = true
	return nil
}

func (m *mockWorld) RemoveTag(id ecs.EntityID, tag string) error {
	if _, exists := m.entities[id]; !exists {
		return errors.New("entity not found")
	}
	delete(m.entities[id], tag)
	return nil
}

func (m *mockWorld) HasTag(id ecs.EntityID, tag string) (bool, error) {
	if _, exists := m.entities[id]; !exists {
		return false, errors.New("entity not found")
	}
	return m.entities[id][tag], nil
}

func (m *mockWorld) SetParent(childID, parentID ecs.EntityID) {}

func (m *mockWorld) ParentOf(id ecs.EntityID) (ecs.EntityID, bool) {
	return 0, false
}

func (m *mockWorld) ChildrenOf(parentID ecs.EntityID) []ecs.EntityID {
	return nil
}
