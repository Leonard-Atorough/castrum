package core

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

type queryPos struct{ X, Y float64 }
type queryVel struct{ X, Y float64 }
type queryTag struct{}

func mustQueryEntity(t *testing.T, w *World, components ...any) Entity {
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

func TestQueryEntityHandle(t *testing.T) {
	w := NewWorld()
	first := mustQueryEntity(t, w, queryPos{})
	second := mustQueryEntity(t, w, queryPos{})
	var handles []Entity
	for entry := range NewQuery(w).With(queryPos{}).Execute() {
		handles = append(handles, entry.Entity())
	}
	if len(handles) != 2 || handles[0] != first || handles[1] != second || !w.HasEntity(handles[0]) {
		t.Fatalf("query handles = %v, want live entities %v and %v", handles, first, second)
	}
	if err := w.DestroyEntity(handles[0]); err != nil {
		t.Fatal(err)
	}
	if w.HasEntity(first) || !w.HasEntity(handles[1]) {
		t.Fatal("query handle did not destroy only the intended entity")
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
	for i, entity := range []Entity{first, second} {
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

	t.Run("filter reconfiguration", func(t *testing.T) {
		world := NewWorld()
		posOnly := mustQueryEntity(t, world, queryPos{})
		both := mustQueryEntity(t, world, queryPos{}, queryVel{})
		tagged := mustQueryEntity(t, world, queryPos{}, queryVel{}, queryTag{})
		ids := func(q *Query) []EntityID {
			var result []EntityID
			for entry := range q.Execute() {
				result = append(result, entry.ID())
			}
			return result
		}
		check := func(t *testing.T, q, fresh *Query, want []EntityID) {
			t.Helper()
			got, reference := ids(q), ids(fresh)
			if !slices.Equal(got, want) || !slices.Equal(got, reference) {
				t.Fatalf("reconfigured query = %v, fresh query = %v, want %v", got, reference, want)
			}
		}

		q := NewQuery(world).With(queryPos{})
		check(t, q, NewQuery(world).With(queryPos{}), []EntityID{posOnly.ID(), both.ID(), tagged.ID()})
		generation := world.archetypes.Generation()
		q.With(queryVel{})
		check(t, q, NewQuery(world).With(queryPos{}, queryVel{}), []EntityID{both.ID(), tagged.ID()})
		q.Without(queryTag{})
		check(t, q, NewQuery(world).With(queryPos{}, queryVel{}).Without(queryTag{}), []EntityID{both.ID()})
		if got := world.archetypes.Generation(); got != generation {
			t.Fatalf("world generation changed from %d to %d during reconfiguration", generation, got)
		}

		q = NewQuery(world).With(queryPos{})
		ids(q)
		q.Without(queryVel{})
		check(t, q, NewQuery(world).With(queryPos{}).Without(queryVel{}), []EntityID{posOnly.ID()})

		for _, tc := range []struct {
			name      string
			configure func(*Query)
			fresh     func() *Query
			want      []EntityID
		}{
			{"With", func(q *Query) { q.With(queryVel{}, nil) }, func() *Query {
				return NewQuery(world).With(queryPos{}, queryVel{})
			}, []EntityID{both.ID(), tagged.ID()}},
			{"Without", func(q *Query) { q.Without(queryVel{}, nil) }, func() *Query {
				return NewQuery(world).With(queryPos{}).Without(queryVel{})
			}, []EntityID{posOnly.ID()}},
		} {
			t.Run(tc.name+" partial panic", func(t *testing.T) {
				q := NewQuery(world).With(queryPos{})
				ids(q)
				func() {
					defer func() {
						if recover() == nil {
							t.Error("invalid marker should panic")
						}
					}()
					tc.configure(q)
				}()
				check(t, q, tc.fresh(), tc.want)
			})
		}
	})
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

func TestExecutePanicsOnStructuralChangeDuringPass(t *testing.T) {
	for _, op := range []string{"destroy", "spawn", "add-component", "remove-component"} {
		t.Run(op, func(t *testing.T) {
			w := NewWorld()
			a := mustQueryEntity(t, w, queryPos{X: 1})
			b := mustQueryEntity(t, w, queryPos{X: 2}, queryVel{X: 9})
			q := NewQuery(w).With(queryPos{})

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("structural change during a pass should panic")
				}
				if msg, ok := r.(string); !ok || !strings.Contains(msg, "invalidated") {
					t.Fatalf("panic = %v, want an iteration-invalidated message", r)
				}
			}()
			for e := range q.Execute() {
				switch op {
				case "destroy":
					_ = w.DestroyEntity(b)
				case "spawn":
					_, _ = w.NewEntity(queryPos{X: 3})
				case "add-component":
					_ = a.AddComponent(w, queryVel{}) // not present on a: migrates archetypes
				case "remove-component":
					_ = b.RemoveComponent[queryVel](w)
				}
				_ = e
			}
		})
	}
}

func TestEntryUsableAfterPassUntilStructuralChange(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1})
	q := NewQuery(w).With(queryPos{})

	entry, ok := q.First()
	if !ok {
		t.Fatal("First found no entry")
	}
	// No structural change since the pass: the entry still reads correctly.
	if p, ok := entry.Component[queryPos](); !ok || p.X != 1 {
		t.Fatalf("entry after pass = %+v, %v; want X=1", p, ok)
	}

	// A structural change invalidates retained entries loudly.
	mustQueryEntity(t, w, queryPos{X: 2})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("stale entry use should panic")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "stale") {
			t.Fatalf("panic = %v, want a stale-entry message", r)
		}
	}()
	_, _ = entry.Component[queryPos]()
}

func TestEntryUpdateOutOfRangeMessage(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{X: 1})
	stale := Entry{
		index:   99,
		columns: map[reflect.Type][]any{reflect.TypeFor[queryPos](): {queryPos{X: 1}}},
		pass:    &queryPass{world: w, structVersion: w.archetypes.StructVersion()},
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("out-of-range row should panic")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "out of range") {
			t.Fatalf("panic = %v, want an out-of-range message (not 'not listed in With')", r)
		}
	}()
	stale.Update(func(p *queryPos) { p.X = 2 })
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

	func() {
		defer func() {
			if r := recover(); r != "update failed" {
				t.Fatalf("panic = %v, want update failed", r)
			}
		}()
		for entry := range NewQuery(w).With(queryPos{}).Execute() {
			entry.Update(func(p *queryPos) {
				p.X = 99
				panic("update failed")
			})
		}
	}()
	got, ok = e.Component[queryPos](w)
	if !ok || got.X != 11 {
		t.Fatalf("position after panic = %v, ok %v, want X 11", got, ok)
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

// TestWritePathValidationMatrix pins the validation contract: spawn, add,
// and entity replacement validate through [Validatable] and leave the stored
// value unchanged when they reject, while query entry writes bypass
// validation by design so iteration stays error-free.
func TestWritePathValidationMatrix(t *testing.T) {
	w := NewWorld()
	if _, err := w.NewEntity(rating{n: 99}); err == nil {
		t.Fatal("spawn with an invalid component should error")
	}
	e, err := w.NewEntity(rating{n: 5}, queryPos{})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := e.AddComponent(w, rating{n: 99}); err == nil {
		t.Fatal("AddComponent with an invalid value should error")
	}

	// Entity replacement validates and does not commit the invalid value.
	if err := e.SetComponent(w, rating{n: 99}); err == nil {
		t.Fatal("SetComponent with an invalid value should error")
	}
	if r, ok := e.Component[rating](w); !ok || r.n != 5 {
		t.Fatalf("stored rating after rejected SetComponent = %v, %v; want 5, true", r, ok)
	}
	if err := e.Update(w, func(r *rating) { r.n = 99 }); err == nil {
		t.Fatal("Update writing an invalid value should error")
	}
	if r, _ := e.Component[rating](w); r.n != 5 {
		t.Fatalf("stored rating after rejected Update = %d; want 5 unchanged", r.n)
	}

	// Query entry writes bypass validation, per the contract on
	// [Entry.SetComponent], [Entry.Update], and [Validatable].
	q := NewQuery(w).With(rating{})
	for entry := range q.Execute() {
		entry.SetComponent(rating{n: 99})
	}
	if r, _ := e.Component[rating](w); r.n != 99 {
		t.Fatalf("stored rating after Entry.SetComponent = %d; want 99 (query writes bypass validation)", r.n)
	}
	for entry := range q.Execute() {
		entry.Update(func(r *rating) { r.n = -1 })
	}
	if r, _ := e.Component[rating](w); r.n != -1 {
		t.Fatalf("stored rating after Entry.Update = %d; want -1 (query writes bypass validation)", r.n)
	}
}
