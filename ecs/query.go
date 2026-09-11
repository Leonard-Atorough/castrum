package ecs

import (
	"iter"
	"reflect"

	"github.com/leonard-atorough/castrum/components"
)

type ResultEntry struct {
	Archetype  *Archetype
	EntityID   EntityID
	Entity     *Entity
	_archetype *Archetype // Private: for lazy component access
	_idx       int        // Private: entity index in archetype
}

// Get retrieves a component from the result entry with error handling.
// Use this instead of world.GetComponent when you already have a query result,
// as the component data is accessed directly from the archetype cache.
func (r ResultEntry) Get[T any]() (T, error) {
	t := reflect.TypeFor[T]()

	// Access directly from archetype
	if r._archetype != nil {
		if raw, ok := r._archetype.componentData[t]; ok {
			// Fast path: try direct type assertion to []T
			if typedSlice, ok := raw.([]T); ok {
				if r._idx >= 0 && r._idx < len(typedSlice) {
					return typedSlice[r._idx], nil
				}
				return *new(T), &EntityError{
					EntityID: r.EntityID,
					Op:       "ResultEntry.Get",
					Err:      ErrEntityNotFound,
				}
			}
			// Fallback: use reflection for generic access
			sliceVal := reflect.ValueOf(raw)
			if sliceVal.Kind() == reflect.Slice && r._idx >= 0 && r._idx < sliceVal.Len() {
				compVal := sliceVal.Index(r._idx)
				if compVal.IsValid() {
					if typed, ok := compVal.Interface().(T); ok {
						return typed, nil
					}
				}
			}
		}
	}

	var zero T
	return zero, &EntityError{
		EntityID: r.EntityID,
		Op:       "ResultEntry.Get",
		Err:      ErrComponentNotFound,
	}
}

type Query struct {
	world    *World
	required []Component
	excluded []Component
	filter   func(ResultEntry) bool
}

func NewQuery(w *World) *Query {
	return &Query{
		world:    w,
		required: []Component{},
		excluded: []Component{},
	}
}

func (q *Query) WithRequiredComponents(comps ...Component) *Query {
	q.required = append(q.required, comps...)
	return q
}

func (q *Query) WithExcludedComponents(comps ...Component) *Query {
	q.excluded = append(q.excluded, comps...)
	return q
}

// WithFilter adds a predicate evaluated per-entity after archetype matching.
// Multiple filters compose: an entity must satisfy all of them.
func (q *Query) WithFilter(fn func(ResultEntry) bool) *Query {
	previous := q.filter
	if previous == nil {
		q.filter = fn
		return q
	}
	q.filter = func(r ResultEntry) bool {
		return previous(r) && fn(r)
	}
	return q
}

// InScene restricts results to entities tagged with the given scene ID via components.SceneTag.
// Requiring SceneTag narrows the archetype search before the per-entity value check runs.
func (q *Query) InScene(sceneID string) *Query {
	q.required = append(q.required, components.SceneTag{})
	return q.WithFilter(func(r ResultEntry) bool {
		tag, err := r.Get[components.SceneTag]()
		return err == nil && tag.SceneID == sceneID
	})
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
				entity, ok := q.world.GetEntity(entityID)
				if !ok {
					continue
				}
				resultEntry := ResultEntry{
					Archetype:  archetype,
					EntityID:   entityID,
					Entity:     entity,
					_archetype: archetype,
					_idx:       i,
				}
				if q.filter != nil && !q.filter(resultEntry) {
					continue
				}
				if !yield(resultEntry) {
					return
				}
			}
		}
	}
}

// All materializes all results into a slice in a single pass.
func (q *Query) All() []ResultEntry {
	results := make([]ResultEntry, 0, 16) // Small initial capacity to avoid zero-alloc edge case
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

// EntityIDs returns just the entity IDs (no component data) in a single pass.
func (q *Query) EntityIDs() []EntityID {
	ids := make([]EntityID, 0, 16) // Small initial capacity to avoid zero-alloc edge case
	for entry := range q.Execute() {
		ids = append(ids, entry.EntityID)
	}
	return ids
}

func types(comps ...Component) []reflect.Type {
	componentTypes := make([]reflect.Type, 0, len(comps))
	for _, comp := range comps {
		componentTypes = append(componentTypes, reflect.TypeOf(comp))
	}
	return componentTypes
}
