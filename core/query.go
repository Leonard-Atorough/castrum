package core

import (
	"cmp"
	"fmt"
	"iter"
	"reflect"
	"slices"

	"github.com/Leonard-Atorough/castrum/internal/ecs"
)

// Entry represents a single entity and its associated component data within a query result.
// It provides methods to access the entity ID and retrieve component values by type.
//
// By design Entries are not meant to be retained. The component columns a
// Get reads are the query's shared prefetch scratch, reused for every
// archetype the iteration visits: a retained entry keeps a stale row
// index into whichever archetype the iteration is on now, so a later Get
// reads the wrong data. Keep the ID instead — [Entry.ID] is a copy, safe to
// hold and resolve through the world's ID-keyed methods after iteration.
type Entry struct {
	entityID EntityID
	index    int
	columns  map[reflect.Type][]any
}

// ID returns the unique identifier of the entity associated with this entry.
// Unlike the entry itself, the ID is safe to retain.
func (e Entry) ID() EntityID {
	return e.entityID
}

// SetComponent overwrites the visited entity's component of type T in place,
// through the prefetched column. Only the types listed in [Query.With]
// are available — the same contract as [Entry.Component]. Value writes are
// safe during iteration: they mutate the entity's slot without moving
// rows. Structural changes (spawning, destroying, adding or removing
// components) remain invalid during a pass.
func (e Entry) SetComponent[T any](value T) {
	typ := reflect.TypeFor[T]()
	column, ok := e.columns[typ]
	if !ok {
		panic(fmt.Sprintf("castrum: Entry.Set: type %v was not listed in With", typ))
	}
	column[e.index] = value
}

// Component retrieves the component value of type T for the entity associated with this entry.
// It returns the value and a boolean indicating whether the component was found.
// Only the component types listed in [Query.With] are available: the
// prefetch carries those types alone, so any other type reports false even
// if the entity has the component.
func (e Entry) Component[T any]() (T, bool) {
	v, ok := e.columns[reflect.TypeFor[T]()]
	if !ok || e.index >= len(v) {
		var zero T
		return zero, false
	}
	val, ok := v[e.index].(T)
	return val, ok
}

// Query selects entities by component types and, optionally, a per-entity
// predicate. Construct it with NewQuery, configure it with With, Without,
// and Where, then range over Execute. A Query is reusable across passes:
// it caches the matching archetypes and re-matches only when the world's
// structure changes. One Query supports one active iteration at a time.
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

// NewQuery creates a reusable query against world. Configure it with With,
// Without, and Where; each Execute reuses the query's cached archetype
// matches until the world's structure changes.
func NewQuery(world *World) *Query {
	return &Query{
		world:   world,
		columns: make(map[reflect.Type][]any),
	}
}

// With adds component types that must be present for an entity to match
// the query. Pass zero values of the component types — the values are
// markers, only their types are used. A nil or a pointer value panics:
// both are build-time mistakes, and a pointer would otherwise silently
// match no component at all.
//
// The listed types are also the only ones [Entry.Component] can retrieve during
// iteration: they are prefetched per archetype, which is what makes Get a
// bare column lookup.
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

// Without adds component types that must be absent for an entity to match
// the query. Pass zero values of the component types — the values are
// markers, only their types are used. A nil or a pointer value panics.
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

// Where adds a predicate function that must return true for an entity to match the query.
// The predicate is evaluated once per entity during iteration and is never
// cached; it can only Get the types listed in With, like the loop body
// itself. Prefer a component over a predicate when the condition is
// stable: structural filters are archetype-granular and free, while
// predicates pay per entity on every pass.
func (q *Query) Where(predicate func(e Entry) bool) *Query {
	q.where = predicate
	return q
}

// Execute runs the query and yields matching entries to the provided iterator function.
// It reuses the query's cached archetype matches and component columns for efficiency.
// Only one active iteration is allowed per query at a time; nested iterations require separate queries.
//
// Iteration order is deterministic: matches are sorted by archetype ID, and
// entities are visited in storage order within an archetype, so two passes
// over an unchanged world yield the same sequence. Breaking out of the
// range stops iteration cleanly, and the query can be executed again
// immediately.
//
// Structural mutation — spawning, destroying, or adding and removing
// components — during iteration is not supported: it moves rows under the
// iterator and corrupts the pass. Perform structural changes before
// iterating or between passes.
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
