package ecs

import (
	"reflect"
	"testing"
)

// Tracked and untracked test component types for change-tick tests.
type TrackedHealth struct{ Value int }

func (TrackedHealth) TrackChanges() {}

type TrackedMana struct{ Value int }

func (TrackedMana) TrackChanges() {}

func trackedArch(t *testing.T, svc *Service, entityID EntityID) *Archetype {
	t.Helper()
	loc, err := svc.location(entityID)
	if err != nil {
		t.Fatalf("location(%d): %v", entityID, err)
	}
	arch, ok := svc.store.get(loc.ArchetypeID)
	if !ok {
		t.Fatalf("archetype %d not found", loc.ArchetypeID)
	}
	return arch
}

func TestServiceCreateStampsTrackedRows(t *testing.T) {
	svc := NewService()
	svc.AdvanceTick()
	svc.AdvanceTick()

	err := svc.Create(1, []reflect.Type{
		reflect.TypeFor[TrackedHealth](),
		reflect.TypeFor[Position](),
	}, []any{
		TrackedHealth{Value: 10},
		Position{X: 1},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	arch := trackedArch(t, svc, 1)
	added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), 0)
	if !ok {
		t.Fatal("tracked type has no ticks")
	}
	if added != 2 || changed != 2 {
		t.Fatalf("added/changed = %d/%d, want 2/2 (current tick)", added, changed)
	}
	if _, _, ok := arch.Ticks(reflect.TypeFor[Position](), 0); ok {
		t.Fatal("untracked type reported ticks")
	}
}

func TestServiceCreateBatchStampsRange(t *testing.T) {
	svc := NewService()
	svc.AdvanceTick()

	ids := []EntityID{1, 2, 3}
	err := svc.CreateBatch(ids, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{Value: 1}})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	arch := trackedArch(t, svc, 1)
	for i, id := range ids {
		if arch.entityIDs[i] != id {
			t.Fatalf("row %d holds entity %d, want %d", i, arch.entityIDs[i], id)
		}
		added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), i)
		if !ok || added != 1 || changed != 1 {
			t.Fatalf("row %d ticks = %d/%d ok=%v, want 1/1 true", i, added, changed, ok)
		}
	}
}

func TestServiceSetComponentStampsChangedOnly(t *testing.T) {
	svc := NewService()
	if err := svc.Create(1, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{Value: 1}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	svc.AdvanceTick()
	if err := svc.SetComponent(1, reflect.TypeFor[TrackedHealth](), TrackedHealth{Value: 2}); err != nil {
		t.Fatalf("SetComponent: %v", err)
	}

	arch := trackedArch(t, svc, 1)
	added, changed, _ := arch.Ticks(reflect.TypeFor[TrackedHealth](), 0)
	if added != 0 || changed != 1 {
		t.Fatalf("added/changed = %d/%d, want 0/1", added, changed)
	}
}

func TestServiceAddComponentsReplaceStampsChanged(t *testing.T) {
	svc := NewService()
	if err := svc.Create(1, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{Value: 1}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	svc.AdvanceTick()
	res := svc.AddComponents(1, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{Value: 9}})
	if res.Error != nil {
		t.Fatalf("AddComponents: %v", res.Error)
	}

	arch := trackedArch(t, svc, 1)
	added, changed, _ := arch.Ticks(reflect.TypeFor[TrackedHealth](), 0)
	if added != 0 || changed != 1 {
		t.Fatalf("added/changed = %d/%d, want 0/1 (replace is a change, not an add)", added, changed)
	}
}

func TestServiceAddComponentsMigrationPreservesTicks(t *testing.T) {
	svc := NewService()
	if err := svc.Create(1, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{Value: 1}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	svc.AdvanceTick()
	svc.AdvanceTick()
	res := svc.AddComponents(1,
		[]reflect.Type{reflect.TypeFor[TrackedMana]()},
		[]any{TrackedMana{Value: 5}})
	if res.Error != nil {
		t.Fatalf("AddComponents: %v", res.Error)
	}
	if !res.Moved {
		t.Fatal("AddComponents with a new type should migrate")
	}

	arch := trackedArch(t, svc, 1)
	loc := trackedArchLocation(t, svc, 1)
	added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), loc.Index)
	if !ok || added != 0 || changed != 0 {
		t.Fatalf("migrated Health ticks = %d/%d ok=%v, want 0/0 true (preserved, not restamped)", added, changed, ok)
	}
	added, changed, ok = arch.Ticks(reflect.TypeFor[TrackedMana](), loc.Index)
	if !ok || added != 2 || changed != 2 {
		t.Fatalf("new Mana ticks = %d/%d ok=%v, want 2/2 true", added, changed, ok)
	}
}

func trackedArchLocation(t *testing.T, svc *Service, entityID EntityID) Location {
	t.Helper()
	loc, err := svc.location(entityID)
	if err != nil {
		t.Fatalf("location(%d): %v", entityID, err)
	}
	return loc
}

func TestServiceRemoveComponentsLogsAndPreserves(t *testing.T) {
	svc := NewService()
	if err := svc.Create(1, []reflect.Type{
		reflect.TypeFor[TrackedHealth](),
		reflect.TypeFor[TrackedMana](),
	}, []any{
		TrackedHealth{Value: 1},
		TrackedMana{Value: 2},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	svc.AdvanceTick()
	res := svc.RemoveComponents(1, []reflect.Type{reflect.TypeFor[TrackedMana]()})
	if res.Error != nil {
		t.Fatalf("RemoveComponents: %v", res.Error)
	}

	removals := svc.Removals(reflect.TypeFor[TrackedMana]())
	if len(removals) != 1 || removals[0].Entity != 1 || removals[0].Tick != 1 {
		t.Fatalf("removals = %+v, want one removal of entity 1 at tick 1", removals)
	}
	if r := svc.Removals(reflect.TypeFor[TrackedHealth]()); len(r) != 0 {
		t.Fatalf("Health removals = %+v, want none", r)
	}
	if r := svc.Removals(reflect.TypeFor[Position]()); len(r) != 0 {
		t.Fatalf("untracked Position removals = %+v, want none", r)
	}

	loc := trackedArchLocation(t, svc, 1)
	arch := trackedArch(t, svc, 1)
	added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), loc.Index)
	if !ok || added != 0 || changed != 0 {
		t.Fatalf("surviving Health ticks = %d/%d ok=%v, want 0/0 true", added, changed, ok)
	}
}

func TestServiceDestroyLogsAllTrackedTypes(t *testing.T) {
	svc := NewService()
	if err := svc.Create(1, []reflect.Type{
		reflect.TypeFor[TrackedHealth](),
		reflect.TypeFor[TrackedMana](),
		reflect.TypeFor[Position](),
	}, []any{
		TrackedHealth{Value: 1},
		TrackedMana{Value: 2},
		Position{},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	svc.AdvanceTick()
	if res := svc.Destroy(1); res.Error != nil {
		t.Fatalf("Destroy: %v", res.Error)
	}

	for _, typ := range []reflect.Type{reflect.TypeFor[TrackedHealth](), reflect.TypeFor[TrackedMana]()} {
		removals := svc.Removals(typ)
		if len(removals) != 1 || removals[0].Entity != 1 || removals[0].Tick != 1 {
			t.Fatalf("removals for %v = %+v, want one removal of entity 1 at tick 1", typ, removals)
		}
	}
	if r := svc.Removals(reflect.TypeFor[Position]()); len(r) != 0 {
		t.Fatalf("untracked Position removals = %+v, want none", r)
	}
}

func TestServiceAdvanceTickRotatesRemovalWindow(t *testing.T) {
	svc := NewService()
	create := func(id EntityID) {
		if err := svc.Create(id, []reflect.Type{reflect.TypeFor[TrackedHealth]()}, []any{TrackedHealth{}}); err != nil {
			t.Fatalf("Create(%d): %v", id, err)
		}
	}
	remove := func(id EntityID) {
		if res := svc.RemoveComponents(id, []reflect.Type{reflect.TypeFor[TrackedHealth]()}); res.Error != nil {
			t.Fatalf("RemoveComponents(%d): %v", id, res.Error)
		}
	}
	health := reflect.TypeFor[TrackedHealth]()
	create(1)
	remove(1)

	// A removal is readable during its tick and the next, oldest first.
	if got := svc.Removals(health); len(got) != 1 || got[0].Entity != 1 || got[0].Tick != 0 {
		t.Fatalf("tick 0 removals = %+v, want [{Entity:1 Tick:0}]", got)
	}
	svc.AdvanceTick()
	create(2)
	remove(2)
	if got := svc.Removals(health); len(got) != 2 || got[0].Entity != 1 || got[0].Tick != 0 || got[1].Entity != 2 || got[1].Tick != 1 {
		t.Fatalf("tick 1 removals = %+v, want [{1 0} {2 1}] oldest first", got)
	}
	svc.AdvanceTick()
	if got := svc.Removals(health); len(got) != 1 || got[0].Entity != 2 {
		t.Fatalf("tick 2 removals = %+v, want only [{Entity:2 Tick:1}] (tick 0 window expired)", got)
	}
	svc.AdvanceTick()
	if got := svc.Removals(health); got != nil {
		t.Fatalf("tick 3 removals = %+v, want nil (window expired)", got)
	}
	if svc.Tick() != 3 {
		t.Fatalf("Tick = %d, want 3", svc.Tick())
	}
}

func TestArchetypeRemoveEntityMovesTicksWithRow(t *testing.T) {
	arch := newArchetype(1, newKey(reflect.TypeFor[TrackedHealth](), reflect.TypeFor[Position]()))
	arch.insertEntity(1)
	arch.insertEntity(2)
	arch.insertEntity(3)
	arch.columns[reflect.TypeFor[TrackedHealth]()][0] = TrackedHealth{Value: 10}
	arch.columns[reflect.TypeFor[TrackedHealth]()][2] = TrackedHealth{Value: 30}
	arch.stampAdded(reflect.TypeFor[TrackedHealth](), 0, 5)
	arch.StampChanged(reflect.TypeFor[TrackedHealth](), 0, 7)
	arch.stampAdded(reflect.TypeFor[TrackedHealth](), 2, 9)

	movedID, moved := arch.removeEntity(0)
	if !moved || movedID != 3 {
		t.Fatalf("removeEntity moved %d/%v, want 3/true", movedID, moved)
	}
	if len(arch.entityIDs) != 2 || arch.entityIDs[0] != 3 || arch.entityIDs[1] != 2 {
		t.Fatalf("entityIDs = %v, want [3 2]", arch.entityIDs)
	}
	added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), 0)
	if !ok || added != 9 || changed != 9 {
		t.Fatalf("moved row ticks = %d/%d ok=%v, want 9/9 true", added, changed, ok)
	}
	if added, _, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), 1); !ok || added != 0 {
		t.Fatalf("untouched row ticks = %d ok=%v, want 0 true", added, ok)
	}
}

func TestArchetypeInsertEntitiesResetsGrownTicks(t *testing.T) {
	arch := newArchetype(1, newKey(reflect.TypeFor[TrackedHealth]()))
	arch.insertEntity(1)
	arch.stampAdded(reflect.TypeFor[TrackedHealth](), 0, 42)
	arch.removeEntity(0)

	start := arch.insertEntities([]EntityID{2, 3})
	arch.stampAddedRange(reflect.TypeFor[TrackedHealth](), start, start+2, 7)
	for i := start; i < start+2; i++ {
		added, changed, ok := arch.Ticks(reflect.TypeFor[TrackedHealth](), i)
		if !ok || added != 7 || changed != 7 {
			t.Fatalf("row %d ticks = %d/%d ok=%v, want 7/7 true", i, added, changed, ok)
		}
	}
}
