package core

import (
	"testing"
)

type trackHealth struct{ Amount int }

func (trackHealth) TrackChanges() {}

type trackMana struct{ Amount int }

func (trackMana) TrackChanges() {}

func queryIDs(q *Query) []EntityID {
	var ids []EntityID
	for e := range q.Execute() {
		ids = append(ids, e.ID())
	}
	return ids
}

func TestQueryAddedFirstPassSeesEveryMatch(t *testing.T) {
	w := NewWorld()
	a := mustQueryEntity(t, w, trackHealth{Amount: 1})
	b := mustQueryEntity(t, w, trackHealth{Amount: 2})

	q := NewQuery(w).With(trackHealth{}).Added()
	got := queryIDs(q)
	if len(got) != 2 || got[0] != a.ID() || got[1] != b.ID() {
		t.Fatalf("first Added pass = %v, want [%d %d]", got, a.ID(), b.ID())
	}
	if got := queryIDs(q); len(got) != 0 {
		t.Fatalf("second Added pass = %v, want none", got)
	}
}

func TestQueryAddedDetectsNewEntitiesPerTick(t *testing.T) {
	w := NewWorld()
	old := mustQueryEntity(t, w, trackHealth{})
	q := NewQuery(w).With(trackHealth{}).Added()
	if got := queryIDs(q); len(got) != 1 || got[0] != old.ID() {
		t.Fatalf("first pass = %v, want [%d]", got, old.ID())
	}

	w.AdvanceTick()
	fresh := mustQueryEntity(t, w, trackHealth{})
	got := queryIDs(q)
	if len(got) != 1 || got[0] != fresh.ID() {
		t.Fatalf("pass after spawn = %v, want only %d (old %d must not reappear)", got, fresh.ID(), old.ID())
	}

	w.AdvanceTick()
	if got := queryIDs(q); len(got) != 0 {
		t.Fatalf("pass with no new entities = %v, want none", got)
	}
}

func TestQueryChangedDetectsWrites(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{Amount: 1})
	q := NewQuery(w).With(trackHealth{}).Changed()
	if got := queryIDs(q); len(got) != 1 {
		t.Fatalf("first Changed pass = %v, want one (adds count as changes)", got)
	}

	w.AdvanceTick()
	if got := queryIDs(q); len(got) != 0 {
		t.Fatalf("pass with no writes = %v, want none", got)
	}
	// Engine order: the tick advances, then the write stamps it, then the
	// watcher's pass sees it.
	w.AdvanceTick()
	if err := e.SetComponent(w, trackHealth{Amount: 2}); err != nil {
		t.Fatal(err)
	}
	if got := queryIDs(q); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("pass after SetComponent = %v, want [%d]", got, e.ID())
	}
	w.AdvanceTick()
	if err := e.Update(w, func(h *trackHealth) { h.Amount = 3 }); err != nil {
		t.Fatal(err)
	}
	if got := queryIDs(q); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("pass after Update = %v, want [%d]", got, e.ID())
	}
	w.AdvanceTick()
	if got := queryIDs(q); len(got) != 0 {
		t.Fatalf("pass after consumption = %v, want none (each change seen once)", got)
	}
}

func TestQueryChangedSeesEntryWrites(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{Amount: 1})
	writer := NewQuery(w).With(trackHealth{})
	watcher := NewQuery(w).With(trackHealth{}).Changed()
	if got := queryIDs(watcher); len(got) != 1 {
		t.Fatalf("first Changed pass = %v, want one", got)
	}

	w.AdvanceTick()
	for entry := range writer.Execute() {
		entry.Update(func(h *trackHealth) { h.Amount++ })
	}
	w.AdvanceTick()
	if got := queryIDs(watcher); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("Changed pass after Entry.Update = %v, want [%d]", got, e.ID())
	}

	w.AdvanceTick()
	for entry := range writer.Execute() {
		entry.SetComponent(trackHealth{Amount: 99})
	}
	w.AdvanceTick()
	if got := queryIDs(watcher); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("Changed pass after Entry.SetComponent = %v, want [%d]", got, e.ID())
	}
	if h, ok := e.Component[trackHealth](w); !ok || h.Amount != 99 {
		t.Fatalf("component = %v ok=%v, want Amount 99", h, ok)
	}
}

func TestQueryWatermarkIsPerQuery(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{})
	first := NewQuery(w).With(trackHealth{}).Changed()
	second := NewQuery(w).With(trackHealth{}).Changed()

	if got := queryIDs(first); len(got) != 1 {
		t.Fatalf("first query first pass = %v, want one", got)
	}
	// The second query has not run yet, so it still sees the spawn even
	// though the first query already advanced its own watermark.
	if got := queryIDs(second); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("second query first pass = %v, want [%d]", got, e.ID())
	}

	w.AdvanceTick()
	if err := e.SetComponent(w, trackHealth{Amount: 2}); err != nil {
		t.Fatal(err)
	}
	w.AdvanceTick()
	if got := queryIDs(first); len(got) != 1 {
		t.Fatalf("first query after write = %v, want one", got)
	}
	if got := queryIDs(second); len(got) != 1 {
		t.Fatalf("second query after write = %v, want one (independent watermark)", got)
	}
}

func TestQueryWatermarkAdvancesOnEarlyBreak(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{})
	q := NewQuery(w).With(trackHealth{}).Changed()
	for range q.Execute() {
		break
	}
	w.AdvanceTick()
	if err := e.SetComponent(w, trackHealth{Amount: 2}); err != nil {
		t.Fatal(err)
	}
	w.AdvanceTick()
	// The first pass advanced the watermark even though it broke early, so
	// the pre-break spawn is not re-delivered; only the write is new.
	if got := queryIDs(q); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("pass after early break + write = %v, want [%d]", got, e.ID())
	}
}

func TestQueryAddedChangedPanicsWithoutTrackedWith(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, queryPos{})

	defer func() {
		if recover() == nil {
			t.Fatal("Added with no tracked With type did not panic")
		}
	}()
	NewQuery(w).With(queryPos{}).Added()
}

func TestQueryAddedAllowsMixedTrackedAndUntrackedWith(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{}, queryPos{})
	q := NewQuery(w).With(trackHealth{}, queryPos{}).Added()
	if got := queryIDs(q); len(got) != 1 || got[0] != e.ID() {
		t.Fatalf("Added pass with mixed With = %v, want [%d]", got, e.ID())
	}
}

func TestEntryChangedOn(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{Amount: 1}, queryPos{})
	w.AdvanceTick()
	if err := e.SetComponent(w, trackHealth{Amount: 2}); err != nil {
		t.Fatal(err)
	}
	w.AdvanceTick()

	q := NewQuery(w).With(trackHealth{}, queryPos{})
	for entry := range q.Execute() {
		if entry.ID() != e.ID() {
			continue
		}
		added, changed := entry.ChangedOn[trackHealth]()
		if added != 0 || changed != 1 {
			t.Fatalf("ChangedOn = %d/%d, want 0/1", added, changed)
		}
	}
}

func TestEntryChangedOnPanics(t *testing.T) {
	w := NewWorld()
	mustQueryEntity(t, w, trackHealth{}, queryPos{})
	q := NewQuery(w).With(trackHealth{})
	entry, ok := q.First()
	if !ok {
		t.Fatal("First found no entry")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("ChangedOn for a type not in With did not panic")
			}
		}()
		entry.ChangedOn[queryPos]()
	}()

	untracked := NewQuery(w).With(queryPos{})
	uentry, ok := untracked.First()
	if !ok {
		t.Fatal("untracked query found no entry")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("ChangedOn for an untracked type did not panic")
			}
		}()
		uentry.ChangedOn[queryPos]()
	}()
}

func TestWorldRemovals(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{}, trackMana{}, queryPos{})
	w.AdvanceTick()

	if err := e.RemoveComponent[trackMana](w); err != nil {
		t.Fatal(err)
	}
	removals := w.Removals[trackMana]()
	if len(removals) != 1 || removals[0].Entity != e || removals[0].Tick != 1 {
		t.Fatalf("Removals[trackMana] = %+v, want one removal of %v at tick 1", removals, e)
	}
	if got := w.Removals[trackHealth](); len(got) != 0 {
		t.Fatalf("Removals[trackHealth] = %v, want none", got)
	}
	if got := w.Removals[queryPos](); got != nil {
		t.Fatalf("Removals for untracked type = %v, want nil", got)
	}

	// The removal stays readable for one more tick, then expires.
	w.AdvanceTick()
	if got := w.Removals[trackMana](); len(got) != 1 {
		t.Fatalf("Removals next tick = %v, want one", got)
	}
	w.AdvanceTick()
	if got := w.Removals[trackMana](); got != nil {
		t.Fatalf("Removals after window = %v, want nil", got)
	}
}

func TestWorldRemovalsOnDestroy(t *testing.T) {
	w := NewWorld()
	e := mustQueryEntity(t, w, trackHealth{}, queryPos{})
	w.AdvanceTick()
	if err := w.DestroyEntity(e); err != nil {
		t.Fatal(err)
	}
	removals := w.Removals[trackHealth]()
	if len(removals) != 1 || removals[0].Entity != e || removals[0].Tick != 1 {
		t.Fatalf("Removals after destroy = %+v, want one removal of %v at tick 1", removals, e)
	}
	if w.HasEntity(removals[0].Entity) {
		t.Fatal("destroyed entity still reported live")
	}
}

func TestWorldAdvanceTickAndTick(t *testing.T) {
	w := NewWorld()
	if got := w.Tick(); got != 0 {
		t.Fatalf("fresh world Tick = %d, want 0", got)
	}
	w.AdvanceTick()
	w.AdvanceTick()
	if got := w.Tick(); got != 2 {
		t.Fatalf("Tick after two advances = %d, want 2", got)
	}
}
