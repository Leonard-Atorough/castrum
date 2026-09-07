package core

import (
	"iter"
	"reflect"
)

type ResultEntry struct {
	Archetype  *Archetype
	EntityID   EntityID
	Entity     *Entity
	Components map[reflect.Type]Component
}

// Get retrieves a component from the result entry with error handling.
// Use this instead of world.GetComponent when you already have a query result,
// as the component data is pre-fetched and avoids redundant lookups.
func (r ResultEntry) Get[T any]() (T, error) {
	t := reflect.TypeFor[T]()
	comp, ok := r.Components[t]
	if !ok {
		var zero T
		return zero, &EntityError{
			EntityID: r.EntityID,
			Op:       "ResultEntry.Get",
			Err:      ErrComponentNotFound,
		}
	}
	typed, ok := comp.(T)
	if !ok {
		var zero T
		return zero, &EntityError{
			EntityID: r.EntityID,
			Op:       "ResultEntry.Get",
			Err:      ErrComponentNotFound,
		}
	}
	return typed, nil
}

type Query struct {
	world    *World
	required []Component
	excluded []Component
}

func NewQuery(w *World) *Query {
	return &Query{
		world:    w,
		required: []Component{},
		excluded: []Component{},
	}
}

func (q *Query) WithRequiredComponents(components ...Component) *Query {
	q.required = append(q.required, components...)
	return q
}

func (q *Query) WithExcludedComponents(components ...Component) *Query {
	q.excluded = append(q.excluded, components...)
	return q
}

func (q *Query) Execute() iter.Seq[ResultEntry] {
	return func(yield func(result ResultEntry) bool) {
		requiredTypes := types(q.required...)
		excludedTypes := types(q.excluded...)

		for _, archetype := range q.world.archetypeManager.archetypes {
			if len(q.required) > 0 && !archetype.componentTypes.ContainsAll(NewArchetypeKey(requiredTypes...)) {
				continue
			}
			if len(q.excluded) > 0 && archetype.componentTypes.ContainsAny(NewArchetypeKey(excludedTypes...)) {
				continue
			}

			for i, entityID := range archetype.entities {
				var components map[reflect.Type]Component
				if len(requiredTypes) > 0 {
					components = make(map[reflect.Type]Component, len(requiredTypes))
					for _, compType := range requiredTypes {
						if raw, ok := archetype.componentData[compType]; ok {
							if comps, ok := raw.([]Component); ok && len(comps) > i {
								components[compType] = comps[i]
							}
						}
					}
				} else {
					components = make(map[reflect.Type]Component, len(archetype.componentData))
					for compType, raw := range archetype.componentData {
						if comps, ok := raw.([]Component); ok && len(comps) > i {
							components[compType] = comps[i]
						}
					}
				}
				entity, ok := q.world.GetEntity(entityID)
				if !ok {
					continue
				}
				resultEntry := ResultEntry{
					Archetype:  archetype,
					EntityID:   entityID,
					Entity:     entity,
					Components: components,
				}
				if !yield(resultEntry) {
					return
				}
			}
		}
	}
}

// All materializes all results into a slice.
func (q *Query) All() []ResultEntry {
	count := q.Count()
	results := make([]ResultEntry, 0, count)
	for entry := range q.Execute() {
		results = append(results, entry)
	}
	return results
}

func (q *Query) First() (ResultEntry, bool) {
	for entry := range q.Execute() {
		return entry, true
	}
	return ResultEntry{}, false
}

func (q *Query) Any() bool {
	for range q.Execute() {
		return true
	}
	return false
}

func (q *Query) Count() int {
	count := 0
	for range q.Execute() {
		count++
	}
	return count
}

// EntityIDs returns just the entity IDs (no component data).
func (q *Query) EntityIDs() []EntityID {
	count := q.Count()
	ids := make([]EntityID, 0, count)
	for entry := range q.Execute() {
		ids = append(ids, entry.EntityID)
	}
	return ids
}

func types(comps ...Component) []reflect.Type {
	var componentTypes []reflect.Type
	for _, comp := range comps {
		componentTypes = append(componentTypes, reflect.TypeOf(comp))
	}
	return componentTypes
}
