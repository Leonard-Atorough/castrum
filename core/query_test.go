package core

import (
	"testing"
)

type queryPos struct{ X, Y float64 }
type queryVel struct{ X, Y float64 }
type queryTag struct{}

func TestQueryYieldsMatchingEntities(t *testing.T) {
	w := NewWorld()
	_, _ = w.NewEntity(queryPos{X: 1, Y: 2}) // pos only: excluded by With(vel)
	both, _ := w.NewEntity(queryPos{X: 3, Y: 4}, queryVel{X: 5, Y: 6})

	q := NewQuery(w).
		With(queryPos{}, queryVel{})

	var got []EntityID
	for e := range q.Execute() {
		p, ok := e.Get[queryPos]()
		v, okV := e.Get[queryVel]()
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
	posOnly, _ := w.NewEntity(queryPos{X: 1, Y: 2})
	_, _ = w.NewEntity(queryPos{X: 3, Y: 4}, queryVel{X: 5, Y: 6})

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
	_, _ = w.NewEntity(queryPos{X: 5, Y: 0})
	high1, _ := w.NewEntity(queryPos{X: 15, Y: 0})
	high2, _ := w.NewEntity(queryPos{X: 25, Y: 0})

	q := NewQuery(w).
		With(queryPos{}).
		Where(func(e Entry) bool {
			p, ok := e.Get[queryPos]()
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
	a, _ := w.NewEntity(queryPos{X: 1, Y: 1})
	b, _ := w.NewEntity(queryPos{X: 2, Y: 2})
	c, _ := w.NewEntity(queryPos{X: 3, Y: 3})
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
	_, _ = w.NewEntity(queryPos{X: 1, Y: 1}, queryVel{X: 1, Y: 1})

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
	_, _ = w.NewEntity(queryPos{X: 2, Y: 2}, queryVel{X: 2, Y: 2})
	count = 0
	for range q.Execute() {
		count++
	}
	if count != 2 {
		t.Fatalf("same-key Execute yielded %d, want 2", count)
	}

	// A new component combination creates a new archetype and bumps the
	// generation: the query must re-match and include it.
	_, _ = w.NewEntity(queryPos{X: 3, Y: 3}, queryVel{X: 3, Y: 3}, queryTag{})
	count = 0
	for range q.Execute() {
		count++
	}
	if count != 3 {
		t.Fatalf("after new archetype, Execute yielded %d, want 3", count)
	}
}

func TestQueryGetLimitedToWithTypes(t *testing.T) {
	w := NewWorld()
	_, _ = w.NewEntity(queryPos{X: 1, Y: 2}, queryVel{X: 3, Y: 4})

	q := NewQuery(w).With(queryPos{})
	for e := range q.Execute() {
		if _, ok := e.Get[queryVel](); ok {
			t.Fatal("Get returned a component that was not in With: the prefetch must only carry requested types")
		}
		if p, ok := e.Get[queryPos](); !ok || p.X != 1 {
			t.Fatalf("Get[queryPos] = %v, %v, want the stored value", p, ok)
		}
	}
}

func TestQueryOrderDeterministic(t *testing.T) {
	w := NewWorld()
	_, _ = w.NewEntity(queryPos{X: 1, Y: 1}, queryVel{X: 1, Y: 1})
	_, _ = w.NewEntity(queryPos{X: 2, Y: 2}, queryTag{})
	_, _ = w.NewEntity(queryPos{X: 3, Y: 3})
	_, _ = w.NewEntity(queryPos{X: 4, Y: 4}, queryVel{X: 4, Y: 4}, queryTag{})

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
	_, _ = w.NewEntity(queryPos{X: 1, Y: 1})
	_, _ = w.NewEntity(queryPos{X: 2, Y: 2})

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
