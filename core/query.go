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
// Retain [Entry.Entity] instead; the handle is safe to keep after the query pass.
type Entry struct {
	entityID EntityID
	index    int
	columns  map[reflect.Type][]any
	pass     *queryPass
}

// queryPass carries the per-pass iteration context shared by all entries of
// one Execute call.
type queryPass struct {
	arch *ecs.Archetype
	tick uint64
}

// ID returns the identifier of the entity associated with this entry.
func (e Entry) ID() EntityID {
	return e.entityID
}

// Entity returns the handle for the visited entity.
func (e Entry) Entity() Entity {
	return NewEntity(e.entityID)
}

// SetComponent overwrites the visited entity's component of type T. Only
// types listed in [Query.With] are available; other types cause a panic.
// The write bypasses [Validatable] so iteration stays error-free; systems
// whose components can be written by other systems mid-loop should
// re-validate before consuming them. If T implements [TrackChanges], the
// write stamps the row's change tick at the current [World.Tick].
func (e Entry) SetComponent[T any](value T) {
	typ := reflect.TypeFor[T]()
	column, ok := e.columns[typ]
	if !ok {
		panic(fmt.Sprintf("castrum: Entry.Set: type %v was not listed in With", typ))
	}
	column[e.index] = value
	e.pass.arch.StampChanged(typ, e.index, e.pass.tick)
}

// Update applies fn to the visited entity's component of type T and writes
// the result back. Only types listed in [Query.With] are available; other
// types cause a panic. Like SetComponent, the write bypasses [Validatable]
// and stamps the row's change tick when T implements [TrackChanges].
func (e Entry) Update[T any](fn func(*T)) {
	typ := reflect.TypeFor[T]()
	column, ok := e.columns[typ]
	var value T

	if !ok || e.index >= len(column) {
		panic(fmt.Sprintf("castrum: Entry.Update: type %v was not listed in With", typ))
	}

	value = column[e.index].(T)
	fn(&value)
	column[e.index] = value
	e.pass.arch.StampChanged(typ, e.index, e.pass.tick)
}

// ChangedOn returns the world change ticks at which the visited entity's
// component of type T was added and most recently written. Only types listed
// in [Query.With] are available; other types cause a panic, as does a type
// that does not implement [TrackChanges]. The ticks are comparable with
// [World.Tick] and [Context.Tick].
func (e Entry) ChangedOn[T any]() (added, changed uint64) {
	typ := reflect.TypeFor[T]()
	if _, ok := e.columns[typ]; !ok {
		panic(fmt.Sprintf("castrum: Entry.ChangedOn: type %v was not listed in With", typ))
	}
	added, changed, ok := e.pass.arch.Ticks(typ, e.index)
	if !ok {
		panic(fmt.Sprintf("castrum: Entry.ChangedOn: type %v does not implement TrackChanges", typ))
	}
	return added, changed
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

// Query selects entities by component presence and optional predicates.
// Configure it with [Query.With], [Query.Without], [Query.Where],
// [Query.Added], and [Query.Changed], then iterate its results with
// [Query.Execute]. Filter changes between passes take effect on the next
// execution.
//
// The Added and Changed filters compare per-row change ticks against a
// watermark recorded by this query's most recent Execute, so they mean
// "since I last looked": two queries over the same component types hold
// independent views. A query that has never executed matches every row on
// its first pass, which gives the desired initial-recompute behavior.
// Keep change-detection queries persistent across ticks and execute them
// once per tick; the watermark advances on every Execute, including passes
// stopped early by breaking the range.
type Query struct {
	world   *World
	with    []reflect.Type
	without []reflect.Type
	where   func(e Entry) bool
	added   bool
	changed bool

	matches    []*ecs.Archetype
	columns    map[reflect.Type][]any
	pass       queryPass
	generation uint64
	matchValid bool
	iterating  bool
	lastTick   uint64
	hasRun     bool
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
		q.matchValid = false
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
		q.matchValid = false
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

// Added filters matches to entities whose component of one of the [Query.With]
// types was added since this query's last [Query.Execute] pass. At least one
// With type must implement [TrackChanges]; other With types are ignored by
// the filter, and a With set with no tracked type causes a panic. A query
// that has never executed matches every row on its first pass.
func (q *Query) Added() *Query {
	q.requireTrackedWith("Added")
	q.added = true
	return q
}

// Changed filters matches to entities whose component of one of the
// [Query.With] types was written since this query's last [Query.Execute]
// pass; additions count as changes. At least one With type must implement
// [TrackChanges]; other With types are ignored by the filter, and a With set
// with no tracked type causes a panic. A query that has never executed
// matches every row on its first pass.
//
// Writes stamp at [World.Tick]. Within one tick a write is visible only to
// queries whose last pass ran at an earlier tick, so run a change-detection
// system after the systems that write the observed components and execute
// each such query once per tick; then every change is seen exactly once.
func (q *Query) Changed() *Query {
	q.requireTrackedWith("Changed")
	q.changed = true
	return q
}

func (q *Query) requireTrackedWith(filter string) {
	for _, t := range q.with {
		if tracksChanges(t) {
			return
		}
	}
	panic(fmt.Sprintf("castrum: %s: no type in With implements TrackChanges", filter))
}

// matchesChange reports whether the row matches the Added/Changed filter at
// the query's current watermark.
func (q *Query) matchesChange(a *ecs.Archetype, index int) bool {
	if !q.hasRun {
		return true
	}
	for _, t := range q.with {
		added, changed, ok := a.Ticks(t, index)
		if !ok {
			continue
		}
		if q.added && added > q.lastTick {
			return true
		}
		if q.changed && changed > q.lastTick {
			return true
		}
	}
	return false
}

// Execute yields entries for matching entities in stable order while the
// world's structure is unchanged. Breaking the range stops the pass, and the
// query can be executed again immediately. Each pass records the current
// [World.Tick] as the query's Added/Changed watermark, even when the pass is
// stopped early.
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

		tick := q.world.archetypes.Tick()
		filterChanges := q.added || q.changed

		gen := q.world.archetypes.Generation()
		if !q.matchValid || gen != q.generation {
			q.matches = q.world.archetypes.MatchInto(q.matches, q.with, q.without)
			slices.SortFunc(q.matches, byArchetypeID)
			q.generation = gen
			q.matchValid = true
		}
		stopped := false
	rows:
		for _, archetype := range q.matches {
			clear(q.columns)
			for _, t := range q.with {
				q.columns[t] = archetype.Column(t)
			}
			q.pass.arch, q.pass.tick = archetype, tick
			entry := Entry{columns: q.columns, pass: &q.pass}
			archetype.EachEntity(func(index int, id EntityID) bool {
				entry.entityID, entry.index = id, index
				if filterChanges && !q.matchesChange(archetype, index) {
					return true
				}
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
				break rows
			}
		}
		q.lastTick, q.hasRun = tick, true
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
