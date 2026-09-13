package ecs

import (
	"fmt"
	"reflect"

	internalecs "github.com/leonard-atorough/castrum/internal/ecs"
)

// World represents the central manager for all ECS state, handling entities, components, and hierarchical relationships.
type World struct {
	entities      map[EntityID]*Entity
	nextID        uint64
	destroyed     []*Entity
	hierarchy     *Hierarchy
	storage       *internalecs.Storage
	systemManager *SystemManager
	resources     map[reflect.Type]any
}

// NewWorld creates and initializes a new World instance with default values.
func NewWorld() *World {
	return &World{
		entities:      make(map[EntityID]*Entity),
		hierarchy:     NewHierarchy(),
		nextID:        0,
		destroyed:     make([]*Entity, 0),
		storage:       internalecs.NewStorage(),
		systemManager: NewSystemManager(),
		resources:     make(map[reflect.Type]any),
	}
}

// Reset clears the world entirely, removing all entities, components, and hierarchy relationships.
func (w *World) Reset() {
	w.entities = make(map[EntityID]*Entity)
	w.storage = internalecs.NewStorage()
	w.hierarchy = NewHierarchy()
	w.nextID = 0
	w.destroyed = make([]*Entity, 0)
}

// Cleanup processes all entities that have been marked for destruction, removing them from the world and updating the hierarchy accordingly.
func (w *World) Cleanup() {
	for _, entity := range w.destroyed {
		if entity == nil {
			continue
		}
		w.removeStoredEntity(entity)
		delete(w.entities, entity.ID)

		if parentID, hasParent := w.hierarchy.Parent(entity.ID); hasParent {
			w.hierarchy.Remove(parentID, entity.ID)
		}
		entity.template = ""
	}
	w.destroyed = make([]*Entity, 0)
}

func (w *World) removeStoredEntity(entity *Entity) bool {
	if !entity.stored {
		return false
	}
	location := internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}
	removed, ok := w.storage.DeleteEntity(location)
	if !ok {
		return false
	}
	entity.stored = false
	if removed.Moved {
		if movedEntity, exists := w.entities[EntityID(removed.MovedID)]; exists {
			movedEntity.archetypeIdx = location.Index
		}
	}
	return true
}

// Count returns the number of active entities in the world.
func (w *World) Count() int {
	return len(w.entities)
}

// Create spawns a bare entity (no components) tagged with blueprintName.
// Use CreateWithComponents to spawn an entity with initial component data.
func (w *World) Create(blueprintName string) *Entity {
	w.nextID++
	id := w.nextID
	loc, err := w.storage.CreateEntity(id, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create entity: %v", err))
	}
	entityID := EntityID(id)
	entity := NewEntity(entityID, blueprintName)
	entity.archetypeID = uint64(loc.ArchetypeID)
	entity.archetypeIdx = loc.Index
	w.entities[entityID] = entity
	return entity
}

// CreateMany spawns count bare entities with the given blueprint name.
func (w *World) CreateMany(blueprintName string, count int) []*Entity {
	if count <= 0 {
		return []*Entity{}
	}
	entities := make([]*Entity, 0, count)
	for range count {
		entities = append(entities, w.Create(blueprintName))
	}
	return entities
}

func (w *World) CreateWithComponents(blueprintName string, components ...Component) (*Entity, error) {
	componentTypes := make([]reflect.Type, len(components))
	componentValues := make([]any, len(components))
	for i, comp := range components {
		if comp == nil {
			return nil, fmt.Errorf("component at index %d is nil", i)
		}
		componentTypes[i] = reflect.TypeOf(comp)
		componentValues[i] = comp
	}
	w.nextID++
	id := w.nextID
	loc, err := w.storage.CreateEntity(id, componentTypes, componentValues)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %v", err)
	}
	entityID := EntityID(id)
	entity := NewEntity(entityID, blueprintName)
	entity.archetypeID = uint64(loc.ArchetypeID)
	entity.archetypeIdx = loc.Index
	w.entities[entityID] = entity
	return entity, nil
}

// DestroyEntity marks an entity for destruction. If cascade is true, all descendants of the entity will also be destroyed.
func (w *World) DestroyEntity(entityID EntityID, cascade bool) error {
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return &EntityError{
			EntityID: entityID,
			Op:       "DestroyEntity",
			Err:      ErrEntityNotFound,
		}
	}
	if !w.removeStoredEntity(entity) {
		return &EntityError{
			EntityID: entityID,
			Op:       "DestroyEntity",
			Err:      ErrArchetypeNotFound,
		}
	}
	if cascade {
		// destroy all descendants of the entity
		descendants := w.hierarchy.Descendants(entityID)
		for _, descID := range descendants {
			descEntity, exists := w.entities[descID]
			if !exists {
				continue
			}
			descEntity.Destroy()
			w.destroyed = append(w.destroyed, descEntity)
		}
	} else {
		// just detach direct children of the entity from the hierarchy
		children := w.hierarchy.Children(entityID)
		for _, childID := range children {
			w.hierarchy.Remove(entityID, childID)
		}
	}
	entity.Destroy()
	w.destroyed = append(w.destroyed, entity)
	return nil
}

// GetEntity returns an entity by ID, including entities pending cleanup.
func (w *World) GetEntity(entityID EntityID) (*Entity, bool) {
	entity, exists := w.entities[entityID]
	return entity, exists
}

// HasEntity checks if an entity with the given EntityID exists in the world.
func (w *World) HasEntity(entityID EntityID) bool {
	_, exists := w.entities[entityID]
	return exists
}

// AddComponent adds one or more components to an entity. The entity must exist in the world. Returns an error if the entity is not found.
func (w *World) AddComponent(entityID EntityID, comp ...Component) error {
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return ErrEntityNotFound
	}

	location := internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}
	components := w.storage.Components(location)
	if components == nil {
		return ErrEntityNotFound
	}

	componentValues := make(map[reflect.Type]any, len(components)+len(comp))
	componentOrder := make([]reflect.Type, 0, len(components)+len(comp))
	for _, existing := range components {
		typ := reflect.TypeOf(existing)
		componentValues[typ] = existing
		componentOrder = append(componentOrder, typ)
	}
	for _, c := range comp {
		if c == nil {
			return &EntityError{
				EntityID: entityID,
				Op:       "AddComponent",
				Err:      fmt.Errorf("component must not be nil"),
			}
		}
		typ := reflect.TypeOf(c)
		if _, exists := componentValues[typ]; !exists {
			componentOrder = append(componentOrder, typ)
		}
		componentValues[typ] = c
	}
	if len(comp) == 0 {
		return nil
	}
	oldIndex := entity.archetypeIdx
	componentData := make([]any, len(componentOrder))
	for i, typ := range componentOrder {
		componentData[i] = componentValues[typ]
	}

	newLocation, moveResult, ok := w.storage.MoveEntity(uint64(entityID), internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}, componentOrder, componentData)
	if !ok {
		return &EntityError{
			EntityID: EntityID(entityID),
			Op:       "AddComponent",
			Err:      fmt.Errorf("failed to add component"),
		}
	}
	entity.archetypeID = uint64(newLocation.ArchetypeID)
	entity.archetypeIdx = newLocation.Index

	if moveResult.Moved {
		w.entities[EntityID(moveResult.MovedID)].archetypeIdx = oldIndex
	}
	return nil
}

// RemoveComponent removes a component of the specified type from an entity. Returns an error if the entity is not found.
func (w *World) RemoveComponent[T Component](entityID EntityID) error {
	componentType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return ErrEntityNotFound
	}

	components := w.storage.Components(internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	})
	if components == nil {
		return nil
	}

	var toKeepTypes []reflect.Type
	var toKeepValues []any
	found := false
	for _, comp := range components {
		if reflect.TypeOf(comp) != componentType {
			toKeepTypes = append(toKeepTypes, reflect.TypeOf(comp))
			toKeepValues = append(toKeepValues, comp)
		} else {
			found = true
		}
	}
	if !found {
		return nil
	}

	oldIndex := entity.archetypeIdx
	newLocation, moveResult, ok := w.storage.MoveEntity(uint64(entityID), internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}, toKeepTypes, toKeepValues)
	if !ok {
		return &EntityError{EntityID: entityID, Op: "RemoveComponent", Err: ErrArchetypeNotFound}
	}
	entity.archetypeID = uint64(newLocation.ArchetypeID)
	entity.archetypeIdx = newLocation.Index

	if moveResult.Moved {
		w.entities[EntityID(moveResult.MovedID)].archetypeIdx = oldIndex
	}
	return nil
}

// GetComponent retrieves the component of the specified type for an entity. Returns an error if the entity or component is not found.
func (w *World) GetComponent[T Component](entityID EntityID) (T, error) {
	var zero T
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}
	comp, ok := w.storage.Get(internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}, compType)
	if !ok {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      ErrEntityNotFound,
		}
	}
	castedComp, ok := comp.(T)
	if !ok {
		return zero, &EntityError{
			EntityID: entityID,
			Op:       "GetComponent",
			Err:      fmt.Errorf("component type mismatch: expected %T", zero),
		}
	}
	return castedComp, nil
}

// HasComponent checks if an entity has a component of the specified type. Returns true if the component exists, false otherwise.
func (w *World) HasComponent[T Component](entityID EntityID) bool {
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return false
	}

	return w.storage.Has(internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}, compType)
}

// SetComponent sets the component of the specified type for an entity. Returns an error if the entity is not found or the component cannot be set.
func (w *World) SetComponent[T Component](entityID EntityID, newComp T) error {
	compType := reflect.TypeFor[T]()
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}

	if ok := w.storage.Set(internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	}, compType, newComp); !ok {
		return &EntityError{
			EntityID: entityID,
			Op:       "SetComponent",
			Err:      ErrEntityNotFound,
		}
	}
	return nil
}

// Components returns all components associated with an entity.
func (w *World) Components(entityID EntityID) []Component {
	components := make([]Component, 0)
	entity, exists := w.entities[entityID]
	if !exists || !entity.alive {
		return nil
	}

	comps := w.storage.Components(internalecs.EntityLocation{
		ArchetypeID: internalecs.ArchetypeID(entity.archetypeID),
		Index:       entity.archetypeIdx,
	})
	if comps == nil {
		return nil
	}
	for _, comp := range comps {
		components = append(components, comp.(Component))
	}
	return components
}

// SetParent sets the parent of childID to parentID in the hierarchy.
func (w *World) SetParent(childID, parentID EntityID) {
	w.hierarchy.Add(parentID, childID)
}

// GetParent returns the parent EntityID of the given child entity, if one exists.
func (w *World) ParentOf(childID EntityID) (EntityID, bool) {
	return w.hierarchy.Parent(childID)
}

// GetChildren returns all direct children of the given parent entity.
func (w *World) ChildrenOf(parentID EntityID) []EntityID {
	return w.hierarchy.Children(parentID)
}

// Detach removes the parent-child relationship for the given entity.
func (w *World) Detach(id EntityID) {
	if parentID, hasParent := w.hierarchy.Parent(id); hasParent {
		w.hierarchy.Remove(parentID, id)
	}
}

// RegisterSystem adds a system to the world's priority-ordered system manager.
func (w *World) RegisterSystem(name string, priority SystemPriority, system System) error {
	return w.systemManager.Register(name, priority, system, w)
}

// SystemManager returns the world's system manager.
func (w *World) SystemManager() *SystemManager {
	return w.systemManager
}

// NewQuery starts a new lazy query builder. Use WithRequiredComponents, WithExcludedComponents,
// then Execute() to iterate, or All()/EntityIDs() to materialize results.
func (w *World) NewQuery() *Query {
	return NewQuery(w)
}

// GetResource retrieves a typed resource from the world.
func (w *World) GetResource[T any]() (T, bool) {
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

// SetResource stores a typed resource in the world.
func (w *World) SetResource[T any](resource T) {
	w.resources[reflect.TypeFor[T]()] = resource
}

// RemoveResource removes a typed resource from the world.
func (w *World) RemoveResource[T any]() {
	delete(w.resources, reflect.TypeFor[T]())
}
