package core

import (
	"reflect"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// EntityID represents a unique identifier for an entity.
// It is an alias for the EntityID type defined in the ECS package.
type EntityID = ecs.EntityID

// Entity represents a game entity with identity, liveness, and hierarchy.
type Entity struct {
	id    EntityID
	alive bool
}

// NewEntity creates a new entity with the given ID.
// Returns a pointer since entities are identity objects that will be
// referenced from multiple places (ECS, systems, etc.).
func NewEntity(id EntityID) *Entity {
	return &Entity{
		id:    id,
		alive: true,
	}
}

// ID returns the entity's unique identifier.
func (e *Entity) ID() EntityID {
	return e.id
}

// IsAlive returns whether the entity is currently active.
func (e *Entity) IsAlive() bool {
	return e.alive
}

// Kill marks the entity as no longer active.
func (e *Entity) Kill() {
	e.alive = false
}

func (e *Entity) Component[T any](w *World) (T, bool) {
	t := reflect.TypeFor[T]()
	value, ok := w.archetypes.Component(e.id, t)
	if !ok {
		var zero T
		return zero, false
	}
	return value.(T), ok
}

func (e *Entity) HasComponent[T any](w *World) bool {
	t := reflect.TypeFor[T]()
	return w.archetypes.HasComponent(e.id, t)
}

func (e *Entity) SetComponent[T any](w *World, value T) error {
	t := reflect.TypeFor[T]()
	return w.archetypes.SetComponent(e.id, t, value)
}

func (e *Entity) AddComponent[T any](w *World, value T) error {
	t := reflect.TypeFor[T]()
	res := w.archetypes.AddComponents(e.id, []reflect.Type{t}, []any{value})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (e *Entity) RemoveComponent[T any](w *World) error {
	t := reflect.TypeFor[T]()
	res := w.archetypes.RemoveComponents(e.id, []reflect.Type{t})
	if res.Error != nil {
		return res.Error
	}
	return nil
}
