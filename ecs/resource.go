package ecs

import "reflect"

// SetResource stores a singleton value of type T on the world, keyed by its type.
// Only one value per type T can be stored; a later call overwrites the previous value.
func SetResource[T any](w *World, val T) {
	w.resources[reflect.TypeFor[T]()] = val
}

// GetResource retrieves the singleton value of type T previously stored with SetResource.
func GetResource[T any](w *World) (T, bool) {
	var zero T
	v, ok := w.resources[reflect.TypeFor[T]()]
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// RemoveResource deletes the stored value of type T, if any.
func RemoveResource[T any](w *World) {
	delete(w.resources, reflect.TypeFor[T]())
}
