package ecs

import "reflect"

type World interface {
	// Lifecycle
	Spawn(template string) EntityID
	Destroy(id EntityID, cascade bool) error

	// Querying
	Query(components ...reflect.Type) []EntityID
	QueryByTag(tag string) []EntityID

	// Components
	AddComponent(id EntityID, component Component) error
	RemoveComponent(id EntityID, componentType reflect.Type) error
	GetComponent(id EntityID, componentType reflect.Type) (Component, error)
	HasComponent(id EntityID, componentType reflect.Type) bool
	Components(id EntityID) []Component

	// Tags
	AddTag(id EntityID, tag string) error
	RemoveTag(id EntityID, tag string) error
	HasTag(id EntityID, tag string) (bool, error)

	// Hierarchy
	SetParent(childID, parentID EntityID)
	ParentOf(id EntityID) (EntityID, bool)
	ChildrenOf(id EntityID) []EntityID
}
