package core

import (
	"fmt"
	"reflect"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// EntityID identifies an entity within a [World]. Zero is reserved and IDs
// are not recycled during a world's lifetime.
type EntityID = ecs.EntityID

// Entity is a comparable handle for addressing an entity in its [World].
// The zero value does not identify an entity. Use [World.HasEntity] to check
// whether a handle still refers to an entity in the world.
//
// Handles do not encode world identity. Use a handle only with the world that
// created its entity; another world may have an entity with the same ID.
type Entity struct {
	id EntityID
}

// NewEntity returns a handle for id without checking whether it exists.
// Use [World.NewEntity] to create an entity.
func NewEntity(id EntityID) Entity {
	return Entity{id: id}
}

// ID returns the entity's identifier.
func (e Entity) ID() EntityID {
	return e.id
}

// Component returns the entity's component of type T. It reports false if
// the entity does not exist or has no component of that type.
func (e Entity) Component[T any](w *World) (T, bool) {
	value, ok := w.archetypes.Component(e.id, reflect.TypeFor[T]())
	if !ok {
		var zero T
		return zero, false
	}
	return value.(T), true
}

// HasComponent reports whether the entity has a component of type T. A
// nonexistent entity reports false.
func (e Entity) HasComponent[T any](w *World) bool {
	return w.archetypes.HasComponent(e.id, reflect.TypeFor[T]())
}

// SetComponent overwrites the entity's component of type T. The entity
// must already have the component; use AddComponent to attach one. It
// returns an error if value is a pointer, the entity does not exist, or it has
// no component of that type.
func (e Entity) SetComponent[T any](w *World, value T) error {
	if err := validateComponentPointer(value); err != nil {
		return fmt.Errorf("castrum: SetComponent: %w", err)
	}
	return w.archetypes.SetComponent(e.id, reflect.TypeFor[T](), value)
}

// Update reads the entity's component of type T, applies fn to the
// copy, and writes it back. It returns an error if the entity does not
// exist or has no component of that type; the write validates like
// SetComponent.
func (e Entity) Update[T any](w *World, fn func(*T)) error {
	value, ok := e.Component[T](w)
	if !ok {
		return fmt.Errorf("castrum: entity %d has no component %v", e.id, reflect.TypeFor[T]())
	}
	fn(&value)
	return e.SetComponent(w, value)
}

// AddComponent attaches a component of type T to the entity. Existing
// components are preserved. It returns an error if value is a pointer. Adding
// a [Transform] also adds an initial [PrevTransform] if the entity does not
// already have one.
func (e Entity) AddComponent[T any](w *World, value T) error {
	if err := validateComponentPointer(value); err != nil {
		return fmt.Errorf("castrum: AddComponent: %w", err)
	}
	if reflect.TypeFor[T]() == reflect.TypeFor[Transform]() && !e.HasComponent[PrevTransform](w) {
		transform := any(value).(Transform)
		res := w.archetypes.AddComponents(e.id,
			[]reflect.Type{reflect.TypeFor[Transform](), reflect.TypeFor[PrevTransform]()},
			[]any{value, transform.snapshot()},
		)
		return res.Error
	}
	res := w.archetypes.AddComponents(e.id, []reflect.Type{reflect.TypeFor[T]()}, []any{value})
	return res.Error
}

// RemoveComponent detaches the component of type T from the entity,
// migrating it to the archetype for its remaining components. Removing a
// component the entity does not have succeeds without effect.
func (e Entity) RemoveComponent[T any](w *World) error {
	res := w.archetypes.RemoveComponents(e.id, []reflect.Type{reflect.TypeFor[T]()})
	return res.Error
}
