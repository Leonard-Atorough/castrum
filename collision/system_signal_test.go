package collision

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
)

func TestSystem_EnteredSignals(t *testing.T) {
	world := newCollisionWorld(t)
	sys := NewSystem()
	player, wall := playerAndWall(t, world)
	hub := world.MustResource[*signal.Hub]()
	entered := hub.NewReader[Entered]()
	exited := hub.NewReader[Exited]()

	if err := sys.Update(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatalf("collision tick: %v", err)
	}

	var got []Entered
	for event, ok := entered.Next(); ok; event, ok = entered.Next() {
		got = append(got, event)
	}
	if len(got) != 2 {
		t.Fatalf("entered signals = %v, want two (one per perspective)", got)
	}
	wantPlayer := Entered{Entity: player.ID(), Other: wall.ID(),
		Point: geom.Vector2{X: 1, Y: 0}, Normal: geom.Vector2{X: 1, Y: 0},
		Penetration: 1, Tick: 1}
	wantWall := wantPlayer
	wantWall.Entity, wantWall.Other = wall.ID(), player.ID()
	wantWall.Normal = geom.Vector2{X: -1, Y: 0}
	if got[0] != wantPlayer && got[1] != wantPlayer {
		t.Fatalf("player's entered signal = %v, want %v", got, wantPlayer)
	}
	if got[0] != wantWall && got[1] != wantWall {
		t.Fatalf("wall's entered signal = %v, want %v", got, wantWall)
	}

	if err := sys.Update(&core.Context{World: world, Tick: 2}); err != nil {
		t.Fatalf("collision tick: %v", err)
	}
	got = nil
	for event, ok := entered.Next(); ok; event, ok = entered.Next() {
		got = append(got, event)
	}
	if len(got) != 0 {
		t.Errorf("entered signals during stay = %v, want none", got)
	}
	var exits []Exited
	for event, ok := exited.Next(); ok; event, ok = exited.Next() {
		exits = append(exits, event)
	}
	if len(exits) != 0 {
		t.Errorf("exited signals while overlapping = %v, want none", exits)
	}
}

func TestSystem_ExitedSignals(t *testing.T) {
	world := newCollisionWorld(t)
	sys := NewSystem()
	player, wall := playerAndWall(t, world)
	hub := world.MustResource[*signal.Hub]()
	entered := hub.NewReader[Entered]()
	exited := hub.NewReader[Exited]()

	if err := sys.Update(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatalf("collision tick: %v", err)
	}
	if err := player.Update(world, func(tr *core.Transform) { tr.Position = geom.Vector2{X: -50} }); err != nil {
		t.Fatalf("move player: %v", err)
	}
	if err := sys.Update(&core.Context{World: world, Tick: 2}); err != nil {
		t.Fatalf("collision tick: %v", err)
	}

	var got []Exited
	for event, ok := exited.Next(); ok; event, ok = exited.Next() {
		got = append(got, event)
	}
	if len(got) != 2 {
		t.Fatalf("exited signals = %v, want two (one per perspective)", got)
	}
	wantPlayer := Exited{Entity: player.ID(), Other: wall.ID(), Tick: 2}
	wantWall := Exited{Entity: wall.ID(), Other: player.ID(), Tick: 2}
	if got[0] != wantPlayer && got[1] != wantPlayer {
		t.Fatalf("player's exited signal = %v, want %v", got, wantPlayer)
	}
	if got[0] != wantWall && got[1] != wantWall {
		t.Fatalf("wall's exited signal = %v, want %v", got, wantWall)
	}
	var entries []Entered
	for event, ok := entered.Next(); ok; event, ok = entered.Next() {
		entries = append(entries, event)
	}
	if len(entries) != 2 || entries[0].Tick != 1 || entries[1].Tick != 1 {
		t.Errorf("entered signals = %v, want only the two signals from tick 1", entries)
	}
}
