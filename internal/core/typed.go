package core

import (
	"fmt"
	"reflect"
)

// This file provides generic, compile-time-typed wrappers around World's
// reflect.Type-based component API. Prefer these in system/game code; the
// reflect-based World methods remain the low-level API for callers that only
// know a component's type dynamically (e.g. the blueprint loader).

// GetComponent returns the typed component T for entityID.
func GetComponent[T Component](w *World, entityID EntityID) (T, error) {
	var zero T
	c, err := w.GetComponent(entityID, reflect.TypeFor[T]())
	if err != nil {
		return zero, err
	}
	typed, ok := c.(T)
	if !ok {
		return zero, fmt.Errorf("entity %d: component is %T, not %T", entityID, c, zero)
	}
	return typed, nil
}

// SetComponent sets the typed component T for entityID.
func SetComponent[T Component](w *World, entityID EntityID, comp T) error {
	return w.SetComponent(entityID, reflect.TypeFor[T](), comp)
}

// HasComponent reports whether entityID currently has a component of type T.
func HasComponent[T Component](w *World, entityID EntityID) bool {
	return w.HasComponent(entityID, reflect.TypeFor[T]())
}

// QueryFor returns all entities that have at least a component of type T.
func QueryFor[T Component](w *World) []EntityID {
	return w.Query(reflect.TypeFor[T]())
}
