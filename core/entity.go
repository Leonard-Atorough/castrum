package core

import (
	"reflect"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// EntityID identifies an entity. IDs are the durable reference to an
// entity: query results carry IDs, every world operation accepts one, and
// an ID remains valid to hold across iterations.
type EntityID = ecs.EntityID

// Entity is a lightweight handle to an entity's ID. It exists for
// spawn-time ergonomics — creation code keeps the handle for a few
// follow-up calls before dropping down to the ID-keyed world methods.
// The handle carries no engine state beyond its liveness flag, and
// liveness here reflects the handle, not the storage: it reports false
// only after [World.DestroyEntity] killed this handle. The ID is the
// source of truth.
type Entity struct {
	id    EntityID
	alive bool
}

// NewEntity creates a handle for the entity with the given ID. Use
// [World.NewEntity] to spawn an entity and receive its handle; for an ID
// already in a world — collected from a query, for example — minting a
// handle with NewEntity is the way to turn the ID back into a working
// Entity whose methods perform component operations.
func NewEntity(id EntityID) *Entity {
	return &Entity{
		id:    id,
		alive: true,
	}
}

// ID returns the entity's durable identifier.
func (e *Entity) ID() EntityID {
	return e.id
}

// IsAlive reports whether this handle's entity has not been destroyed
// through [World.DestroyEntity]. Storage membership is the authoritative
// liveness: queries only yield entities that exist.
func (e *Entity) IsAlive() bool {
	return e.alive
}

// Kill marks the handle's entity as destroyed. It does not remove the
// entity from the world; [World.DestroyEntity] does both.
func (e *Entity) Kill() {
	e.alive = false
}

// Component returns the entity's component of type T. It reports false
// if the entity does not exist in the world or has no component of that
// type. The world is passed explicitly: a handle holds no storage, so
// every component operation goes through the world.
func (e *Entity) Component[T any](w *World) (T, bool) {
	value, ok := w.archetypes.Component(e.id, reflect.TypeFor[T]())
	if !ok {
		var zero T
		return zero, false
	}
	return value.(T), true
}

// HasComponent reports whether the entity has a component of type T. A
// nonexistent entity reports false.
func (e *Entity) HasComponent[T any](w *World) bool {
	return w.archetypes.HasComponent(e.id, reflect.TypeFor[T]())
}

// SetComponent overwrites the entity's component of type T. The entity
// must already have the component; use AddComponent to attach one. It
// returns an error if the entity does not exist or has no component of
// that type.
func (e *Entity) SetComponent[T any](w *World, value T) error {
	return w.archetypes.SetComponent(e.id, reflect.TypeFor[T](), value)
}

// AddComponent attaches a component of type T to the entity, migrating it
// to the archetype for its new component set. Existing components are
// preserved.
func (e *Entity) AddComponent[T any](w *World, value T) error {
	res := w.archetypes.AddComponents(e.id, []reflect.Type{reflect.TypeFor[T]()}, []any{value})
	return res.Error
}

// RemoveComponent detaches the component of type T from the entity,
// migrating it to the archetype for its remaining components. Removing a
// component the entity does not have succeeds without effect.
func (e *Entity) RemoveComponent[T any](w *World) error {
	res := w.archetypes.RemoveComponents(e.id, []reflect.Type{reflect.TypeFor[T]()})
	return res.Error
}
