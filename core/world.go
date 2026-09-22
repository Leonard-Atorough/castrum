// Package core provides the foundational structures and mechanisms for managing the game world,
// including entities and resources, and their lifecycle within the world.
package core

import (
	"fmt"
	"reflect"
)

// resourceState represents the current state of a resource within the world.
type resourceState int

const (
	// stateRegistered indicates that the resource has been registered but not yet resolved.
	stateRegistered resourceState = iota
	// stateResolving indicates that the resource is currently being resolved.
	stateResolving
	// stateResolved indicates that the resource has been successfully resolved.
	stateResolved
)

// resourceEntry represents an individual resource within the world, including its current state,
// constructor function, and the instantiated resource once resolved.
type resourceEntry[T any] struct {
	state    resourceState
	ctor     func(*World) (T, error)
	instance T
}

// Resources manages the collection of all resources within the world. It keeps track of their registration,
// resolution state, and the order in which eager resources should be resolved.
type Resources struct {
	entries map[reflect.Type]*resourceEntry[any]
	eager   []reflect.Type
}

const defaultEagerCapacity = 8

// World represents the game world, containing all entities and resources.
type World struct {
	*Resources
}

// NewWorld creates and initializes a new game world with empty resources.
// It returns a pointer to the newly created World instance.
func NewWorld() *World {
	return &World{
		Resources: &Resources{
			entries: make(map[reflect.Type]*resourceEntry[any]),
			eager:   make([]reflect.Type, 0, defaultEagerCapacity),
		},
	}
}

// Provide registers a new resource with the world using the given constructor function.
// The resource will be lazily resolved when first requested.
func (w *World) Provide[T any](ctor func(*World) (T, error)) error {
	return w.provide(ctor, false)
}

// ProvideEager registers a new resource with the world using the given constructor function.
// The resource will be resolved immediately when ResolveEager is called.
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

// Resource retrieves the resource of the specified type from the world.
// If the resource has not been resolved yet, it will be resolved at this time.
// Returns an error if the resource is not registered or if resolution fails.
func (w *World) Resource[T any]() (T, error) {
	typ := reflect.TypeFor[T]()

	if err := w.resolve(typ); err != nil {
		var zero T
		return zero, err
	}
	entry := w.entries[typ]

	return entry.instance.(T), nil
}

// ResolveEager resolves all eager resources registered with the world.
// It returns an error if any resource fails to resolve.
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
