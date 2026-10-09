package core

import (
	"cmp"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// Entry is a query result for one entity. Its component data is backed by
// iteration storage and may refer to another row after iteration advances.
// Retain [Entry.ID] instead; the ID is safe to keep after the query pass.
type Entry struct {
	entityID EntityID
	index    int
	columns  map[reflect.Type][]any
}

// ID returns the identifier of the entity associated with this entry.
func (e Entry) ID() EntityID {
	return e.entityID
}

// SetComponent overwrites the visited entity's component of type T. Only
// types listed in [Query.With] are available; other types cause a panic.
func (e Entry) SetComponent[T any](value T) {
	typ := reflect.TypeFor[T]()
	column, ok := e.columns[typ]
	if !ok {
		panic(fmt.Sprintf("castrum: Entry.Set: type %v was not listed in With", typ))
	}
	column[e.index] = value
}

// Update applies fn to the visited entity's component of type T and writes
// the result back. Only types listed in [Query.With] are available; other
// types cause a panic.
func (e Entry) Update[T any](fn func(*T)) {
	value, ok := e.Component[T]()
	if !ok {
		panic(fmt.Sprintf("castrum: Entry.Update: type %v was not listed in With", reflect.TypeFor[T]()))
	}
	fn(&value)
	e.SetComponent(value)
}

// Component returns the visited entity's component of type T. It reports
// false for types not listed in [Query.With], even if the entity has them.
func (e Entry) Component[T any]() (T, bool) {
	v, ok := e.columns[reflect.TypeFor[T]()]
	if !ok || e.index >= len(v) {
		var zero T
		return zero, false
	}
	val, ok := v[e.index].(T)
	return val, ok
}

// Query selects entities by component presence and an optional predicate.
// Configure it with [Query.With], [Query.Without], and [Query.Where], then
// iterate its results with [Query.Execute].
type Query struct {
	world   *World
	with    []reflect.Type
	without []reflect.Type
	where   func(e Entry) bool

	matches    []*ecs.Archetype
	columns    map[reflect.Type][]any
	generation uint64
	iterating  bool
}

// NewQuery creates a query over world.
func NewQuery(world *World) *Query {
	return &Query{
		world:   world,
		columns: make(map[reflect.Type][]any),
	}
}

// With adds component types that an entity must have to match. Pass zero
// values as type markers; nil and pointer values cause a panic. These are the
// only types available through [Entry.Component], [Entry.SetComponent], and
// [Entry.Update] during iteration.
func (q *Query) With(components ...any) *Query {
	for _, c := range components {
		typ := reflect.TypeOf(c)
		if typ == nil {
			panic("castrum: With: cannot provide nil type to query, component types must be non-nil")
		}
		if typ.Kind() == reflect.Pointer {
			panic("castrum: With: pointer types are not allowed, component types must be non-pointer")
		}
		q.with = append(q.with, typ)
	}
	return q
}

// Without adds component types that an entity must not have to match. Pass
// zero values as type markers; nil and pointer values cause a panic.
func (q *Query) Without(components ...any) *Query {
	for _, c := range components {
		typ := reflect.TypeOf(c)
		if typ == nil {
			panic("castrum: Without: cannot provide nil type to query, component types must be non-nil")
		}
		if typ.Kind() == reflect.Pointer {
			panic("castrum: Without: pointer types are not allowed, component types must be non-pointer")
		}
		q.without = append(q.without, typ)
	}
	return q
}

// Where adds a predicate that an entity must satisfy to match. The predicate
// runs for each candidate on every pass and can access only components listed
// in [Query.With], through [Entry.Component]. Use [Query.With] or
// [Query.Without] for component-presence filters.
func (q *Query) Where(predicate func(e Entry) bool) *Query {
	q.where = predicate
	return q
}

// Execute yields entries for matching entities in stable order while the
// world's structure is unchanged. Breaking the range stops the pass, and the
// query can be executed again immediately.
//
// A query cannot be executed recursively; use a separate query for nested
// iteration. Do not spawn or destroy entities, or add or remove components,
// during a pass. Such changes invalidate the iteration. Updating component
// values through an [Entry] is allowed.
func (q *Query) Execute() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		// One active iteration per query: the prefetch scratch is shared.
		if q.iterating {
			panic("castrum: query is already executing; create a separate query for nested iteration")
		}
		q.iterating = true
		defer func() { q.iterating = false }()

		gen := q.world.archetypes.Generation()
		if gen != q.generation {
			q.matches = q.world.archetypes.MatchInto(q.matches, q.with, q.without)
			slices.SortFunc(q.matches, byArchetypeID)
			q.generation = gen
		}
		stopped := false
		for _, archetype := range q.matches {
			clear(q.columns)
			for _, t := range q.with {
				q.columns[t] = archetype.Column(t)
			}
			entry := Entry{columns: q.columns}
			archetype.EachEntity(func(index int, id EntityID) bool {
				entry.entityID, entry.index = id, index
				if q.where != nil && !q.where(entry) {
					return true
				}
				if !yield(entry) {
					stopped = true
					return false
				}
				return true
			})
			if stopped {
				return
			}
		}
	}
}

// First returns the first entry yielded by the query and whether a match exists.
func (q *Query) First() (Entry, bool) {
	var result Entry
	found := false
	for e := range q.Execute() {
		result = e
		found = true
		break
	}
	return result, found
}

func byArchetypeID(a, b *ecs.Archetype) int {
	return cmp.Compare(a.ID(), b.ID())
}
