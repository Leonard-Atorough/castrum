package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
)

const noKey input.Key = -1

func TestSceneControlsAndEntitySystems(t *testing.T) {
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	director := g.World().MustResource[*scene.Director]()
	hangar := requireScene(t, director, "hangar")
	if got := ownedEntities(g, hangar.ID); got != 2 {
		t.Fatalf("hangar entities = %d, want 2", got)
	}
	ship := ownedMovingEntity(t, g, hangar.ID, spin{})
	advance(t, g, input.KeyS)
	if _, ok := director.ByName("scanner"); ok {
		t.Fatal("scanner loaded in the hangar")
	}

	advance(t, g, input.KeySpace)
	field := requireScene(t, director, "field")
	if got := ownedEntities(g, field.ID); got != 0 {
		t.Fatalf("pending field entities = %d, want 0", got)
	}
	if rotation(t, g, ship) == 0 {
		t.Fatal("hangar ship did not rotate before the switch was applied")
	}

	advance(t, g, noKey)
	if _, ok := director.Get(hangar.ID); ok {
		t.Fatal("hangar survived switch")
	}
	if got := ownedEntities(g, hangar.ID); got != 0 {
		t.Fatalf("old hangar entities = %d, want 0", got)
	}
	if got := ownedEntities(g, field.ID); got != 7 {
		t.Fatalf("field entities = %d, want 7", got)
	}
	debris := ownedMovingEntity(t, g, field.ID, drift{})
	debrisPosition := position(t, g, debris)

	advance(t, g, input.KeyS)
	if position(t, g, debris) <= debrisPosition {
		t.Fatal("field debris did not move")
	}
	scanner := requireScene(t, director, "scanner")
	if got := ownedEntities(g, scanner.ID); got != 0 {
		t.Fatalf("pending scanner entities = %d, want 0", got)
	}
	advance(t, g, noKey)
	if got := ownedEntities(g, scanner.ID); got != 1 {
		t.Fatalf("scanner entities = %d, want 1", got)
	}
	if got := ownedEntities(g, field.ID); got != 7 {
		t.Fatalf("field entities with scanner = %d, want 7", got)
	}

	advance(t, g, input.KeyS)
	advance(t, g, noKey)
	if _, ok := director.Get(scanner.ID); ok {
		t.Fatal("scanner survived removal")
	}
	if got := ownedEntities(g, scanner.ID); got != 0 {
		t.Fatalf("removed scanner entities = %d, want 0", got)
	}
	if got := ownedEntities(g, field.ID); got != 7 {
		t.Fatalf("field entities after scanner removal = %d, want 7", got)
	}
	advance(t, g, input.KeyS)
	scanner = requireScene(t, director, "scanner")
	advance(t, g, noKey)
	advance(t, g, input.KeySpace)
	advance(t, g, noKey)
	if _, ok := director.Get(field.ID); ok {
		t.Fatal("field survived return to hangar")
	}
	if _, ok := director.Get(scanner.ID); ok {
		t.Fatal("scanner survived switch")
	}
	if got := ownedEntities(g, field.ID) + ownedEntities(g, scanner.ID); got != 0 {
		t.Fatalf("old field and scanner entities = %d, want 0", got)
	}
	newHangar := requireScene(t, director, "hangar")
	if got := ownedEntities(g, newHangar.ID); got != 2 {
		t.Fatalf("returned hangar entities = %d, want 2", got)
	}
	newShip := ownedMovingEntity(t, g, newHangar.ID, spin{})
	hangarRotation := rotation(t, g, newShip)
	advance(t, g, noKey)
	if rotation(t, g, newShip) == hangarRotation {
		t.Fatal("new hangar ship did not rotate")
	}
}

func advance(t *testing.T, g *castrum.Game, key input.Key) {
	t.Helper()
	snapshot := &input.Snapshot{}
	if key >= 0 {
		snapshot.Keys[key] = input.State{Pressed: true, Held: true}
	}
	g.Context().Input = snapshot
	if err := g.Advance(20 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func requireScene(t *testing.T, director *scene.Director, name string) scene.Instance {
	t.Helper()
	inst, ok := director.ByName(name)
	if !ok {
		t.Fatalf("scene %q not found", name)
	}
	return inst
}

func ownedEntities(g *castrum.Game, id scene.InstanceID) int {
	count := 0
	for entry := range core.NewQuery(g.World()).With(scene.OwnershipRecord{}).Execute() {
		owner, _ := entry.Component[scene.OwnershipRecord]()
		if owner.Instance == id {
			count++
		}
	}
	return count
}

func ownedMovingEntity(t *testing.T, g *castrum.Game, id scene.InstanceID, marker any) *core.Entity {
	t.Helper()
	for entry := range core.NewQuery(g.World()).With(scene.OwnershipRecord{}, marker).Execute() {
		owner, _ := entry.Component[scene.OwnershipRecord]()
		if owner.Instance == id {
			return core.NewEntity(entry.ID())
		}
	}
	t.Fatalf("scene %d has no entity with %T", id, marker)
	return nil
}

func rotation(t *testing.T, g *castrum.Game, entity *core.Entity) float64 {
	t.Helper()
	transform, ok := entity.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("spin probe has no transform")
	}
	return transform.Rotation
}

func position(t *testing.T, g *castrum.Game, entity *core.Entity) float64 {
	t.Helper()
	transform, ok := entity.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("drift probe has no transform")
	}
	return transform.Position.X
}
