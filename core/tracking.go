package core

import (
	"reflect"
)

// TrackChanges opts a component type into change detection. Tracked types
// support [Query.Added], [Query.Changed], [Entry.ChangedOn], and
// [World.Removals].
//
// Implement it on the component value type; pointer components are rejected:
//
//	type Health struct{ Amount int }
//
//	func (Health) TrackChanges() {}
//
// The world stamps writes made through [World.NewEntity],
// [World.NewEntities], [Entity.AddComponent], [Entity.SetComponent],
// [Entity.Update], [Entry.SetComponent], and [Entry.Update]. Writes that
// bypass these APIs are not stamped.
type TrackChanges interface {
	// TrackChanges opts the component type into per-row change tracking.
	TrackChanges()
}

var trackChangesType = reflect.TypeFor[TrackChanges]()

func tracksChanges(t reflect.Type) bool {
	return t != nil && t.Implements(trackChangesType)
}

// Removal records the removal of a tracked component from an entity.
type Removal struct {
	// Entity is the handle of the entity that lost the component. The handle
	// may no longer refer to a live entity if the entity was destroyed.
	Entity Entity
	// Tick is the world change tick at which the removal was recorded.
	Tick uint64
}

// AdvanceTick advances the world's change tick and retires removals from the
// preceding tick.
//
// The engine calls AdvanceTick once per fixed update. A manually driven world
// should call it once per update: writes made after a query has executed at
// the same tick are not visible to that query's change filters.
func (w *World) AdvanceTick() {
	w.archetypes.AdvanceTick()
}

// Tick returns the world's current change tick, starting at zero. It is
// comparable with the ticks returned by [Entry.ChangedOn] and [World.Removals].
// When advanced by the engine, it is in step with [Context.Tick].
func (w *World) Tick() uint64 {
	return w.archetypes.Tick()
}

// Removals returns the removals of component type T recorded during the
// current and previous change ticks, oldest first. A nonempty result is newly
// allocated; if there are no records, it returns nil.
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
