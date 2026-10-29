package core

import (
	"testing"
)

type queryPos struct{ X, Y float64 }
type queryVel struct{ X, Y float64 }
type queryTag struct{}

func mustQueryEntity(t *testing.T, w *World, components ...any) *Entity {
	t.Helper()
	entity, err := w.NewEntity(components...)
	if err != nil {
		t.Fatalf("create query entity: %v", err)
	}
	return entity
}

func TestQueryFirst(t *testing.T) {
	w := NewWorld()
	q := NewQuery(w).With(queryPos{}).Where(func(e Entry) bool {
		p, ok := e.Component[queryPos]()
		return ok && p.X > 10
	})
	if _, ok := q.First(); ok {
		t.Fatal("First on an empty world found an entry")
	}
	mustQueryEntity(t, w, queryPos{X: 5})
	if _, ok := q.First(); ok {
		t.Fatal("First found an entry excluded by the predicate")
	}
	first := mustQueryEntity(t, w, queryPos{X: 15})
	second := mustQueryEntity(t, w, queryPos{X: 25})

	entry, ok := q.First()
	if !ok || entry.ID() != first.ID() {
		t.Fatalf("First = %v, %t; want entity %d", entry.ID(), ok, first.ID())
	}
	if p, ok := entry.Component[queryPos](); !ok || p.X != 15 {
		t.Fatalf("First component = %v, %t; want X=15", p, ok)
	}
	var ids []EntityID
	for e := range q.Execute() {
		ids = append(ids, e.ID())
	}
	if len(ids) != 2 || ids[0] != first.ID() || ids[1] != second.ID() {
		t.Fatalf("Execute after First = %v, want [%d %d]", ids, first.ID(), second.ID())
	}
}

func TestQueryYieldsMatchingEntities(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1, Y: 2})
	both := mustQueryEntity(t, w, queryPos{X: 3, Y: 4}, queryVel{X: 5, Y: 6})

	q := NewQuery(w).
		With(queryPos{}, queryVel{})

	var got []EntityID
	for e := range q.Execute() {
		p, ok := e.Component[queryPos]()
		v, okV := e.Component[queryVel]()
		if !ok || !okV {
			t.Fatalf("entry %d: pos ok=%v vel ok=%v, want both present", e.ID(), ok, okV)
		}
		if p.X != 3 || v.X != 5 {
			t.Fatalf("entry %d: pos=%v vel=%v, want the stored values", e.ID(), p, v)
		}
		got = append(got, e.ID())
	}
	if len(got) != 1 || got[0] != both.ID() {
		t.Fatalf("query yielded %v, want only %d", got, both.ID())
	}
}

func TestQueryWithoutExcludes(t *testing.T) {
	w := NewWorld()
	posOnly := mustQueryEntity(t, w, queryPos{X: 1, Y: 2})
	mustQueryEntity(t, w, queryPos{X: 3, Y: 4}, queryVel{X: 5, Y: 6})

	q := NewQuery(w).
		With(queryPos{}).
		Without(queryVel{})

	var got []EntityID
	for e := range q.Execute() {
		got = append(got, e.ID())
	}
	if len(got) != 1 || got[0] != posOnly.ID() {
		t.Fatalf("query yielded %v, want only %d", got, posOnly.ID())
	}
}

func TestQueryWhereFilters(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 5, Y: 0})
	high1 := mustQueryEntity(t, w, queryPos{X: 15, Y: 0})
	high2 := mustQueryEntity(t, w, queryPos{X: 25, Y: 0})

	q := NewQuery(w).
		With(queryPos{}).
		Where(func(e Entry) bool {
			p, ok := e.Component[queryPos]()
			return ok && p.X > 10
		})

	want := map[EntityID]bool{high1.ID(): true, high2.ID(): true}
	count := 0
	for e := range q.Execute() {
		if !want[e.ID()] {
			t.Errorf("query yielded %d, which fails the predicate", e.ID())
		}
		count++
	}
	if count != 2 {
		t.Fatalf("query yielded %d entities, want 2", count)
	}
}

func TestQueryEarlyStop(t *testing.T) {
	w := NewWorld()
	a := mustQueryEntity(t, w, queryPos{X: 1, Y: 1})
	b := mustQueryEntity(t, w, queryPos{X: 2, Y: 2})
	c := mustQueryEntity(t, w, queryPos{X: 3, Y: 3})
	all := []EntityID{a.ID(), b.ID(), c.ID()}

	q := NewQuery(w).With(queryPos{})

	count := 0
	var first EntityID
	for e := range q.Execute() {
		if count == 0 {
			first = e.ID()
		}
		count++
		if count >= 1 {
			break
		}
	}
	if count != 1 {
		t.Fatalf("early stop yielded %d entities, want 1", count)
	}
	if first != all[0] {
		t.Errorf("first yielded entity = %d, want %d", first, all[0])
	}

	// A stopped iteration must not leave the query in a broken state.
	count = 0
	for range q.Execute() {
		count++
	}
	if count != 3 {
		t.Fatalf("second Execute after early stop yielded %d entities, want 3", count)
	}
}

func TestQueryGenerationInvalidation(t *testing.T) {
	w := NewWorld()
	first := mustQueryEntity(t, w, queryPos{X: 1, Y: 1}, queryVel{X: 1, Y: 1})

	q := NewQuery(w).
		With(queryPos{}, queryVel{})

	count := 0
	for range q.Execute() {
		count++
	}
	if count != 1 {
		t.Fatalf("first Execute yielded %d, want 1", count)
	}

	// Same-key spawn adds no archetype: the cached matches stay valid and
	// the new entity is still seen because iteration reads live storage.
	second := mustQueryEntity(t, w, queryPos{X: 2, Y: 2}, queryVel{X: 2, Y: 2})
	count = 0
	for range q.Execute() {
		count++
	}
	if count != 2 {
		t.Fatalf("same-key Execute yielded %d, want 2", count)
	}

	// A new component combination creates a new archetype and bumps the
	// generation: the query must re-match and include it.
	mustQueryEntity(t, w, queryPos{X: 3, Y: 3}, queryVel{X: 3, Y: 3}, queryTag{})
	count = 0
	for range q.Execute() {
		count++
	}
	if count != 3 {
		t.Fatalf("after new archetype, Execute yielded %d, want 3", count)
	}

	generation := w.archetypes.Generation()
	for i, entity := range []*Entity{first, second} {
		if err := w.DestroyEntity(entity); err != nil {
			t.Fatalf("DestroyEntity: %v", err)
		}
		if got := w.archetypes.Generation(); got != generation+uint64(i) {
			t.Errorf("generation after destruction %d = %d, want %d", i+1, got, generation+uint64(i))
		}
		count = 0
		for entry := range q.Execute() {
			if entry.ID() == first.ID() || (i == 1 && entry.ID() == second.ID()) {
				t.Errorf("query yielded destroyed entity %d", entry.ID())
			}
			count++
		}
		if count != 2-i {
			t.Fatalf("after destruction %d, Execute yielded %d, want %d", i+1, count, 2-i)
		}
	}

	recreated, err := w.NewEntity(queryPos{X: 4, Y: 4}, queryVel{X: 4, Y: 4})
	if err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	count = 0
	found := false
	for entry := range q.Execute() {
		count++
		if entry.ID() == recreated.ID() {
			found = true
			if pos, ok := entry.Component[queryPos](); !ok || pos.X != 4 {
				t.Errorf("recreated archetype component = %v, %v, want X=4", pos, ok)
			}
		}
	}
	if count != 2 || !found {
		t.Fatalf("after archetype recreation, Execute yielded %d, found new entity=%v", count, found)
	}

	if _, err := w.NewEntity(queryPos{X: 5, Y: 5}, queryVel{X: 5, Y: 5}); err != nil {
		t.Fatalf("NewEntity: %v", err)
	}
	generation = w.archetypes.Generation()
	for _, migrate := range []func() error{
		func() error { return recreated.AddComponent(w, queryTag{}) },
		func() error { return recreated.RemoveComponent[queryTag](w) },
	} {
		if err := migrate(); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if got := w.archetypes.Generation(); got != generation {
			t.Errorf("migration between populated archetypes changed generation: got %d, want %d", got, generation)
		}
		count = 0
		var sum float64
		for entry := range q.Execute() {
			pos, ok := entry.Component[queryPos]()
			if !ok {
				t.Fatal("migrated query entry has no position")
			}
			sum += pos.X
			count++
		}
		if count != 3 || sum != 12 {
			t.Fatalf("after migration, Execute yielded %d entities with position sum %v, want 3 and 12", count, sum)
		}
	}
}

func TestQueryGetLimitedToWithTypes(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1, Y: 2}, queryVel{X: 3, Y: 4})

	q := NewQuery(w).With(queryPos{})
	for e := range q.Execute() {
		if _, ok := e.Component[queryVel](); ok {
			t.Fatal("Get returned a component that was not in With: the prefetch must only carry requested types")
		}
		if p, ok := e.Component[queryPos](); !ok || p.X != 1 {
			t.Fatalf("Get[queryPos] = %v, %v, want the stored value", p, ok)
		}
	}
}

func TestQueryOrderDeterministic(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1, Y: 1}, queryVel{X: 1, Y: 1})
	mustQueryEntity(t, w, queryPos{X: 2, Y: 2}, queryTag{})
	mustQueryEntity(t, w, queryPos{X: 3, Y: 3})
	mustQueryEntity(t, w, queryPos{X: 4, Y: 4}, queryVel{X: 4, Y: 4}, queryTag{})

	q := NewQuery(w).With(queryPos{})

	var first, second []EntityID
	for e := range q.Execute() {
		first = append(first, e.ID())
	}
	for e := range q.Execute() {
		second = append(second, e.ID())
	}
	if len(first) != 4 {
		t.Fatalf("query yielded %d entities, want 4", len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("iteration order changed between passes: %v vs %v", first, second)
		}
	}
}

func TestQueryReentrantPanics(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1, Y: 1})
	mustQueryEntity(t, w, queryPos{X: 2, Y: 2})

	q := NewQuery(w).With(queryPos{})

	defer func() {
		if recover() == nil {
			t.Fatal("nested iteration over the same query should panic")
		}
	}()
	for range q.Execute() {
		for range q.Execute() {
		}
	}
}

func TestWithMarkerGuards(t *testing.T) {
	w := NewWorld()

	defer func() {
		if recover() == nil {
			t.Error("nil marker should panic")
		}
	}()
	NewQuery(w).With(nil)
}

func TestWithPointerMarkerPanics(t *testing.T) {
	w := NewWorld()

	defer func() {
		if recover() == nil {
			t.Error("pointer marker should panic")
		}
	}()
	NewQuery(w).With(&queryPos{})
}

func TestWithoutPointerMarkerPanics(t *testing.T) {
	w := NewWorld()

	defer func() {
		if recover() == nil {
			t.Error("pointer marker in Without should panic")
		}
	}()
	NewQuery(w).Without(&queryPos{})
}

func TestEntryUpdateMutates(t *testing.T) {
	w := NewWorld()
	e, err := w.NewEntity(queryPos{X: 1, Y: 2})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	visits := 0
	for entry := range NewQuery(w).With(queryPos{}).Execute() {
		entry.Update(func(p *queryPos) { p.X += 10 })
		visits++
	}
	if visits != 1 {
		t.Fatalf("visits = %d, want 1", visits)
	}
	got, ok := e.Component[queryPos](w)
	if !ok || got.X != 11 {
		t.Fatalf("position = %v, ok %v, want X 11", got, ok)
	}
}

func TestEntryUpdateUnprefetchedPanics(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(queryPos{}, queryVel{}); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Update on a type not listed in With should panic")
		}
	}()
	for entry := range NewQuery(w).With(queryPos{}).Execute() {
		entry.Update(func(v *queryVel) { v.X = 1 })
	}
}
