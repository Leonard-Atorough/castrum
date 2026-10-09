// Package core provides [World]-owned entities and resources, [Query]
// iteration over components, and the [System] and [Context] contracts for
// updating a game.
package core

import (
	"fmt"
	"reflect"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

type resourceState int

const (
	stateRegistered resourceState = iota
	stateResolving
	stateResolved
)

type resourceEntry[T any] struct {
	state    resourceState
	ctor     func(*World) (T, error)
	instance T
}

const defaultEagerCapacity = 8

// World owns a game's entities, components, and registered resources.
type World struct {
	// entries holds every registered resource by its type, from
	// registration through resolution.
	entries map[reflect.Type]*resourceEntry[any]
	// eager lists the resource types registered eagerly, in
	// registration order, for ResolveEager.
	eager []reflect.Type

	nextEntityID EntityID
	archetypes   *ecs.Service
}

// NewWorld creates an empty world.
func NewWorld() *World {
	return &World{
		entries:    make(map[reflect.Type]*resourceEntry[any]),
		eager:      make([]reflect.Type, 0, defaultEagerCapacity),
		archetypes: ecs.NewService(),
	}
}

// Provide registers a resource constructor. The resource is created on its
// first [World.Resource] request.
func (w *World) Provide[T any](ctor func(*World) (T, error)) error {
	return w.provide(ctor, false)
}

// ProvideEager registers a resource constructor for resolution by
// [World.ResolveEager].
func (w *World) ProvideEager[T any](ctor func(*World) (T, error)) error {
	return w.provide(ctor, true)
}

func (w *World) provide[T any](ctor func(*World) (T, error), eager bool) error {
	typ := reflect.TypeFor[T]()
	if _, exists := w.entries[typ]; exists {
		return fmt.Errorf("resource of type %s is already registered", typ)
	}

	w.entries[typ] = &resourceEntry[any]{
		state: stateRegistered,
		ctor:  func(world *World) (any, error) { return ctor(world) },
	}

	if eager {
		w.eager = append(w.eager, typ)
	}

	return nil
}

// Resource returns the registered resource of type T, resolving it on first
// access. It returns an error if T is not registered or resolution fails.
func (w *World) Resource[T any]() (T, error) {
	typ := reflect.TypeFor[T]()

	if err := w.resolve(typ); err != nil {
		var zero T
		return zero, err
	}
	entry := w.entries[typ]

	return entry.instance.(T), nil
}

// MustResource returns the resource of type T, panicking if it is not
// registered or cannot be resolved. Use it only for resources guaranteed by
// engine-owned wiring; use [World.Resource] when the resource may be absent.
func (w *World) MustResource[T any]() T {
	value, err := w.Resource[T]()
	if err != nil {
		panic(fmt.Sprintf("castrum: MustResource[%s]: %v", reflect.TypeFor[T](), err))
	}
	return value
}

// ResolveEager resolves resources registered with [World.ProvideEager], in
// registration order. It stops at and returns the first resolution error.
func (w *World) ResolveEager() error {
	for _, typ := range w.eager {
		if err := w.resolve(typ); err != nil {
			return err
		}
	}
	return nil
}

func (w *World) resolve(key reflect.Type) error {
	entry, exists := w.entries[key]
	if !exists {
		return fmt.Errorf("resource of type %s is not registered", key)
	}

	switch entry.state {
	case stateRegistered:
		entry.state = stateResolving
		instance, err := entry.ctor(w)
		if err != nil {
			entry.state = stateRegistered
			return err
		}
		entry.instance = instance
		entry.state = stateResolved
	case stateResolving:
		return fmt.Errorf("circular dependency detected for resource of type %s", key)
	case stateResolved:
		// already resolved, do nothing
	}

	return nil
}

// NewEntity creates an entity with the given components and returns its
// handle. It returns an error if a component is nil or fails validation.
// Adding a [Transform] also adds its initial [PrevTransform] unless one is
// provided.
func (w *World) NewEntity(components ...any) (*Entity, error) {
	components = createPreviousTransformComponents(components)
	for i, c := range components {
		if c == nil {
			return nil, fmt.Errorf("castrum: NewEntity: component at index %d is nil", i)
		}
		// Validate before the ID is allocated so a failed spawn burns
		// nothing. The storage service re-checks at entry - every
		// component path is enforced, this one just protects ordering.
		if v, ok := c.(Validatable); ok {
			if err := v.Validate(); err != nil {
				return nil, fmt.Errorf("castrum: NewEntity: component at index %d (%T): %w", i, c, err)
			}
		}
	}
	id := w.getNextID()
	types := make([]reflect.Type, len(components))
	for i, c := range components {
		types[i] = reflect.TypeOf(c)
	}

	values := make([]any, len(components))
	copy(values, components)

	entity := NewEntity(id)
	// Cannot fail here - input validated above, IDs unique by
	// construction - but the check stays honest against the service
	// growing new failure modes.
	if err := w.archetypes.Create(entity.id, types, values); err != nil {
		return nil, err
	}
	return entity, nil
}

// NewEntities creates count entities with the same components and returns
// their handles. Count must be non-negative. If creation fails, it returns
// the error and no handles; entities created before the failure remain in
// the world.
func (w *World) NewEntities(count int, components ...any) ([]*Entity, error) {
	entities := make([]*Entity, count)
	for i := range entities {
		var err error
		components = createPreviousTransformComponents(components)
		entities[i], err = w.NewEntity(components...)
		if err != nil {
			return nil, err
		}
	}
	return entities, nil
}

// DestroyEntity removes the entity from the world and kills entity's handle.
func (w *World) DestroyEntity(entity *Entity) error {
	res := w.archetypes.Destroy(entity.id)
	if res.Error != nil {
		return res.Error
	}

	entity.Kill()
	return nil
}

// DestroyEntities removes the given entities and kills their handles,
// stopping at the first error. Entities removed before an error remain
// destroyed.
func (w *World) DestroyEntities(entities []*Entity) error {
	for _, entity := range entities {
		if err := w.DestroyEntity(entity); err != nil {
			return err
		}
	}
	return nil
}

func (w *World) getNextID() EntityID {
	id := w.nextEntityID
	w.nextEntityID++
	return id
}

func createPreviousTransformComponents(components []any) []any {
	var transform Transform
	hasTransform, hasPrev := false, false
	for _, c := range components {
		if t, ok := c.(Transform); ok {
			transform = t
			hasTransform = true
		}
		if _, ok := c.(PrevTransform); ok {
			hasPrev = true
		}
	}
	if hasTransform && !hasPrev {
		components = append(components, transform.snapshot())
	}
	return components
}
