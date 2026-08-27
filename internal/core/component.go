package core

import (
	"fmt"
	"reflect"
)

type componentStore struct {
}

// The below functions are set to replace the componentStore with a more efficient structure for managing components in archetypes.

// GetComponent retrieves a component of type T for the given entity ID from the specified archetype.
func GetComponent[T any](archetype *Archetype, index int) (T, error) {
	compType := reflect.TypeFor[T]()
	if data, exists := archetype.componentData[compType]; exists {
		if slice, ok := data.([]T); ok {
			if index >= 0 && index < len(slice) {
				return slice[index], nil
			}
			return *new(T), fmt.Errorf("index %d out of bounds for component type %v", index, compType)
		}
	}
	var zero T
	return zero, fmt.Errorf("component of type %v not found in archetype", compType)
}

func SetComponent[T any](archetype *Archetype, index int, comp T) {
	compType := reflect.TypeFor[T]()

	if _, exists := archetype.componentData[compType]; !exists {
		archetype.componentData[compType] = make([]T, 0)
	}

	slice, _ := archetype.componentData[compType].([]T)

	if index >= len(slice) {
		// Extend the slice to accommodate the new index
		newSlice := make([]T, index+1)
		copy(newSlice, slice)
		slice = newSlice
		archetype.componentData[compType] = slice
	}

	slice[index] = comp
}
