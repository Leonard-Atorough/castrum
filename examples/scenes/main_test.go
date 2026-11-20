package main

import (
	"slices"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestControls(t *testing.T) {
	d := startGame(t)
	panel := d.game.World().MustResource[*debug.Panel]()
	if d.location != hangar.name {
		t.Fatalf("initial location = %q, want %q", d.location, hangar.name)
	}

	advance(t, d.game, input.KeyS)
	advance(t, d.game)
	if !slices.Contains(panel.StatusLines(), "Scanner: active") {
		t.Fatal("scanner control did not activate the scanner")
	}
	advance(t, d.game, input.KeyS)
	advance(t, d.game)
	if slices.Contains(panel.StatusLines(), "Scanner: active") {
		t.Fatal("scanner control did not deactivate the scanner")
	}

	advance(t, d.game, input.KeyS)
	advance(t, d.game)
	scanner := d.scanner
	advance(t, d.game, input.KeySpace, input.KeyS)
	if d.scanner != scanner {
		t.Fatal("scanner control ran when switching had priority")
	}
	advance(t, d.game)
	if d.location != field.name || !slices.Contains(panel.StatusLines(), "Location: field") {
		t.Fatalf("location after travel = %q, status = %v", d.location, panel.StatusLines())
	}
	if slices.Contains(panel.StatusLines(), "Scanner: active") {
		t.Fatal("scanner status survived switching rooms")
	}

	// A scanner ID from the previous room must not prevent a new load.
	advance(t, d.game, input.KeyS)
	advance(t, d.game)
	if !slices.Contains(panel.StatusLines(), "Scanner: active") {
		t.Fatal("scanner did not reactivate after switching rooms")
	}
	advance(t, d.game, input.KeySpace)
	advance(t, d.game)
	if d.location != hangar.name || !slices.Contains(panel.StatusLines(), "Location: hangar") {
		t.Fatalf("location after return = %q, status = %v", d.location, panel.StatusLines())
	}
}

func TestFieldObstacles(t *testing.T) {
	d := startGame(t)
	advance(t, d.game, input.KeySpace)
	advance(t, d.game)

	var obstacles []core.Entity
	for _, entity := range d.director.Entities(d.room) {
		if entity.HasComponent[drift](d.game.World()) {
			obstacles = append(obstacles, entity)
		}
	}
	if len(obstacles) != 6 {
		t.Fatalf("field obstacles = %d, want 6", len(obstacles))
	}
	before, ok := obstacles[0].Component[core.Transform](d.game.World())
	if !ok {
		t.Fatal("obstacle has no transform")
	}
	advance(t, d.game)
	after, ok := obstacles[0].Component[core.Transform](d.game.World())
	if !ok || after.Position.X <= before.Position.X {
		t.Fatal("obstacle did not drift")
	}

	if err := obstacles[0].Update(d.game.World(), func(t *core.Transform) {
		t.Position.X = 1300
	}); err != nil {
		t.Fatal(err)
	}
	advance(t, d.game)
	wrapped, ok := obstacles[0].Component[core.Transform](d.game.World())
	if !ok || wrapped.Position.X < -120 || wrapped.Position.X >= -118 {
		t.Fatalf("obstacle position after wrapping = %v", wrapped.Position.X)
	}

	advance(t, d.game, input.KeySpace)
	advance(t, d.game)
	moving := core.NewQuery(d.game.World()).With(drift{})
	for range moving.Execute() {
		t.Fatal("field obstacle survived returning to the hangar")
	}
}

func startGame(t *testing.T) *demo {
	t.Helper()
	d, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.game.Startup(); err != nil {
		t.Fatal(err)
	}
	return d
}

func advance(t *testing.T, g *castrum.Game, keys ...input.Key) {
	t.Helper()
	snapshot := &input.Snapshot{}
	for _, key := range keys {
		snapshot.Keys[key] = input.State{Pressed: true, Held: true}
	}
	g.Context().Input = snapshot
	if err := g.Advance(20 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
}
