package core

import (
	"reflect"
)

// TrackChanges is the opt-in contract for component change detection. A
// component type implementing it gets per-row added/changed ticks in
// archetype storage, which power the [Query.Added] and [Query.Changed]
// filters, [Entry.ChangedOn], and the [World.Removals] log. Types that do not
// implement it are stored exactly as before, at zero cost.
//
// Implement it with a value receiver - pointer components are rejected
// outright, so a pointer-receiver method never opts in:
//
//	type Health struct{ Amount int }
//
//	func (Health) TrackChanges() {}
//
// The engine stamps writes made through [World.NewEntity],
// [World.NewEntities], [Entity.AddComponent], [Entity.SetComponent],
// [Entity.Update], [Entry.SetComponent], and [Entry.Update]. Writes that
// bypass these APIs are not stamped, the same boundary as [Validatable].
type TrackChanges interface {
	// TrackChanges opts the component type into per-row change tracking.
	TrackChanges()
}

var trackChangesType = reflect.TypeOf((*TrackChanges)(nil)).Elem()

// tracksChanges reports whether component type t implements [TrackChanges].
func tracksChanges(t reflect.Type) bool {
	return t != nil && t.Implements(trackChangesType)
}

// Removal records the removal of a tracked component type from an entity.
// Removals are not query filters - the component is gone - so the world
// keeps a short log instead; see [World.Removals].
type Removal struct {
	// Entity is the handle of the entity that lost the component. The handle
	// may no longer refer to a live entity if the entity was destroyed.
	Entity Entity
	// Tick is the world change tick at which the removal was recorded. It is
	// comparable with [World.Tick] and [Context.Tick].
	Tick uint64
}

// AdvanceTick advances the world's change tick by one and retires the oldest
// [World.Removals] window: removals from the tick that just ended stay
// readable until the next advance.
//
// A game built with the engine advances the tick once per fixed update, in
// step with [Context.Tick]. A world driven without the engine runner should
// call AdvanceTick once per update so change detection progresses; stamps
// written while the tick is frozen are invisible to queries that already
// executed at that tick.
func (w *World) AdvanceTick() {
	w.archetypes.AdvanceTick()
}

// Tick returns the world's current change tick. It starts at zero and
// advances with [World.AdvanceTick], which the engine calls once per fixed
// update. Ticks stamped on components through [Entry.ChangedOn] and
// [World.Removals] are comparable with it and with [Context.Tick].
func (w *World) Tick() uint64 {
	return w.archetypes.Tick()
}

// Removals returns the removals of component type T recorded during the
// current and the previous change tick, oldest first. The result is newly
// allocated.
//
// Only tracked types are recorded: if T does not implement [TrackChanges],
// Removals returns nil. Removing a component from an entity, or destroying
// the entity, records one removal per tracked component type the entity had.
func (w *World) Removals[T any]() []Removal {
	typ := reflect.TypeFor[T]()
	if !tracksChanges(typ) {
		return nil
	}
	raw := w.archetypes.Removals(typ)
	if len(raw) == 0 {
		return nil
	}
	removals := make([]Removal, len(raw))
	for i, r := range raw {
		removals[i] = Removal{Entity: NewEntity(r.Entity), Tick: r.Tick}
	}
	return removals
}
