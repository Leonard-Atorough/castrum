package ecs

import (
	"fmt"
	"reflect"
)

// Storage manages entities and their components, organizing them into archetypes for efficient access.
type Storage struct {
	archetypes *ArchetypeStore
}

// EntityLocation represents the location of an entity within an archetype.
type EntityLocation struct {
	ArchetypeID ArchetypeID
	Index       int
}

// RemovalResult represents the result of removing an entity from an archetype, indicating if another entity was moved to fill the gap.
type RemovalResult struct {
	MovedID uint64
	Moved   bool
}

func NewStorage() *Storage {
	return &Storage{
		archetypes: newArchetypeStore(),
	}
}

// CreateEntity creates a new entity with the specified components and values, returning its location within the archetype.
func (s *Storage) CreateEntity(entityID uint64, types []reflect.Type, values []any) (EntityLocation, error) {
	if err := validateComponentData(types, values); err != nil {
		return EntityLocation{}, err
	}
	arch := s.archetypes.GetOrCreate(types...)
	idx := arch.AddEntity(entityID)
	for i, typ := range types {
		arch.Set(idx, typ, values[i])
	}
	return EntityLocation{
		ArchetypeID: arch.ID(),
		Index:       idx,
	}, nil
}

// DeleteEntity removes an entity from its archetype, returning the result of the removal and a boolean indicating success.
func (s *Storage) DeleteEntity(loc EntityLocation) (RemovalResult, bool) {
	arch, ok := s.archetypes.Get(loc.ArchetypeID)
	if !ok || loc.Index < 0 || loc.Index >= arch.Len() {
		return RemovalResult{}, false
	}
	movedID, moved := arch.RemoveEntity(loc.Index)
	return RemovalResult{MovedID: movedID, Moved: moved}, true
}

// MoveEntity moves an entity from one archetype to another, updating its components as specified, and returns the new location, removal result, and a boolean indicating success.
func (s *Storage) MoveEntity(entityID uint64, from EntityLocation, toTypes []reflect.Type, toValues []any) (EntityLocation, RemovalResult, bool) {
	if err := validateComponentData(toTypes, toValues); err != nil {
		return EntityLocation{}, RemovalResult{}, false
	}
	fromArch, ok := s.archetypes.Get(from.ArchetypeID)
	if !ok {
		return EntityLocation{}, RemovalResult{}, false
	}
	if from.Index < 0 || from.Index >= fromArch.Len() {
		return EntityLocation{}, RemovalResult{}, false
	}
	newArch := s.archetypes.GetOrCreate(toTypes...)
	if newArch == fromArch {
		for i, typ := range toTypes {
			newArch.Set(from.Index, typ, toValues[i])
		}
		return from, RemovalResult{}, true
	}
	newIdx := newArch.AddEntity(entityID)
	for i, typ := range toTypes {
		newArch.Set(newIdx, typ, toValues[i])
	}
	movedID, moved := fromArch.RemoveEntity(from.Index)
	return EntityLocation{
		ArchetypeID: newArch.ID(),
		Index:       newIdx,
	}, RemovalResult{MovedID: movedID, Moved: moved}, true
}

// Get retrieves the value of a component for an entity at the specified location, returning the value and a boolean indicating if it exists.
func (s *Storage) Get(location EntityLocation, typ reflect.Type) (any, bool) {
	arch, ok := s.archetypes.Get(location.ArchetypeID)
	if !ok {
		return nil, false
	}
	return arch.Get(location.Index, typ)
}

// Set updates the value of a component for an entity at the specified location, returning a boolean indicating success.
func (s *Storage) Set(location EntityLocation, typ reflect.Type, value any) bool {
	arch, ok := s.archetypes.Get(location.ArchetypeID)
	if !ok {
		return false
	}
	return arch.Set(location.Index, typ, value)
}

// Has checks if an entity at the specified location has a component of the given type, returning a boolean result.
func (s *Storage) Has(location EntityLocation, typ reflect.Type) bool {
	arch, ok := s.archetypes.Get(location.ArchetypeID)
	if !ok {
		return false
	}
	_, has := arch.Get(location.Index, typ)
	return has
}

func validateComponentData(types []reflect.Type, values []any) error {
	if len(types) != len(values) {
		return fmt.Errorf("component type/value count mismatch: %d types, %d values", len(types), len(values))
	}
	for i, typ := range types {
		if typ == nil {
			return fmt.Errorf("component type at index %d is nil", i)
		}
		value := reflect.ValueOf(values[i])
		if !value.IsValid() || !value.Type().AssignableTo(typ) {
			return fmt.Errorf("component value at index %d has type %T, want %s", i, values[i], typ)
		}
	}
	return nil
}
