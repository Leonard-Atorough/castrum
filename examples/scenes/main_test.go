package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

const noKey input.Key = -1

func TestSceneControlsAndEntitySystems(t *testing.T) {
	d, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	g := d.game
	director := d.director
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}

	if !director.Live(d.root) {
		t.Fatal("hangar did not enter at startup")
	}
	// hangar.json authors three nodes: the backdrop, the ship, and the
	// location title.
	if got := len(director.Entities(d.root)); got != 3 {
		t.Fatalf("hangar entities = %d, want 3", got)
	}
	ship := movingEntity(t, g, spin{})

	advance(t, g, input.KeyS)
	if director.Live(d.scanner) {
		t.Fatal("scanner loaded in the hangar")
	}

	hangar := d.root
	advance(t, g, input.KeySpace) // queues the switch; the hangar keeps running
	if got := len(director.Entities(d.root)); got != 0 {
		t.Fatalf("pending field entities = %d, want 0", got)
	}
	if rotation(t, g, ship) == 0 {
		t.Fatal("hangar ship did not rotate before the switch was applied")
	}

	advance(t, g, noKey) // applies the switch
	if director.Live(hangar) {
		t.Fatal("hangar survived switch")
	}
	if got := len(director.Entities(hangar)); got != 0 {
		t.Fatalf("old hangar entities = %d, want 0", got)
	}
	if got := len(director.Entities(d.root)); got != 7 {
		t.Fatalf("field entities = %d, want 7", got)
	}
	field := d.root
	debris := movingEntity(t, g, drift{})
	debrisPosition := position(t, g, debris)

	advance(t, g, input.KeyS) // queues the scanner load
	if got := len(director.Entities(d.scanner)); got != 0 {
		t.Fatalf("pending scanner entities = %d, want 0", got)
	}
	advance(t, g, noKey)
	if got := len(director.Entities(d.scanner)); got != 1 {
		t.Fatalf("scanner entities = %d, want 1", got)
	}
	if got := len(director.Entities(d.root)); got != 7 {
		t.Fatalf("field entities with scanner = %d, want 7", got)
	}
	if position(t, g, debris) <= debrisPosition {
		t.Fatal("field debris did not move")
	}

	scanner := d.scanner
	advance(t, g, input.KeyS) // queues the removal
	advance(t, g, noKey)
	if director.Live(scanner) {
		t.Fatal("scanner survived removal")
	}
	if got := len(director.Entities(scanner)); got != 0 {
		t.Fatalf("removed scanner entities = %d, want 0", got)
	}

	advance(t, g, input.KeyS) // queues a reload
	advance(t, g, noKey)
	if !director.Live(d.scanner) {
		t.Fatal("scanner did not reload")
	}

	advance(t, g, input.KeySpace) // queues the return to the hangar
	advance(t, g, noKey)
	if director.Live(field) || director.Live(d.scanner) {
		t.Fatal("field or scanner survived the switch")
	}
	if got := len(director.Entities(field)) + len(director.Entities(d.scanner)); got != 0 {
		t.Fatalf("old field and scanner entities = %d, want 0", got)
	}
	if got := len(director.Entities(d.root)); got != 3 {
		t.Fatalf("returned hangar entities = %d, want 3", got)
	}
	newShip := movingEntity(t, g, spin{})
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

func movingEntity(t *testing.T, g *castrum.Game, marker any) core.Entity {
	t.Helper()
	query := core.NewQuery(g.World()).With(core.Transform{}, marker)
	entry, ok := query.First()
	if !ok {
		t.Fatalf("no entity with %T", marker)
	}
	return entry.Entity()
}

func rotation(t *testing.T, g *castrum.Game, entity core.Entity) float64 {
	t.Helper()
	transform, ok := entity.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("spin probe has no transform")
	}
	return transform.Rotation
}

func position(t *testing.T, g *castrum.Game, entity core.Entity) float64 {
	t.Helper()
	transform, ok := entity.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("drift probe has no transform")
	}
	return transform.Position.X
}
