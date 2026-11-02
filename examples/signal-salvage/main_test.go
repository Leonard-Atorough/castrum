package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/timer"
)

func TestRoundWinCollectsEveryCoreAndSwitchesToResult(t *testing.T) {
	game, state := startTestRound(t)
	world := game.World()
	player := findEntity(t, world, pilot{}, core.Transform{})

	cores := core.NewQuery(world).With(dataCore{}, core.Transform{})
	var positions []geom.Vector2
	for entry := range cores.Execute() {
		coreState, _ := entry.Component[dataCore]()
		if coreState.collected {
			continue
		}
		transform, _ := entry.Component[core.Transform]()
		positions = append(positions, transform.Position)
	}
	if len(positions) != coreGoal {
		t.Fatalf("available cores = %d, want %d", len(positions), coreGoal)
	}

	for _, position := range positions {
		if err := player.Update(world, func(t *core.Transform) { t.Position = position }); err != nil {
			t.Fatalf("move player to core: %v", err)
		}
		advanceTestFrame(t, game, &input.Snapshot{})
	}

	if state.score != coreGoal || !state.finished || !state.won {
		t.Fatalf("round state = %+v, want won with %d cores", state, coreGoal)
	}
	advanceTestFrame(t, game, &input.Snapshot{})
	if _, ok := game.World().MustResource[*scene.Director]().ByName("result"); !ok {
		t.Fatal("winning round did not switch to the result scene")
	}
}

func TestRoundLosesOnPulseContact(t *testing.T) {
	game, state := startTestRound(t)
	world := game.World()
	player := findEntity(t, world, pilot{}, core.Transform{})
	drone := findEntity(t, world, pulseDrone{}, core.Transform{})
	droneTransform, _ := drone.Component[core.Transform](world)

	if err := player.Update(world, func(t *core.Transform) { t.Position = droneTransform.Position }); err != nil {
		t.Fatalf("move player into pulse: %v", err)
	}
	advanceTestFrame(t, game, &input.Snapshot{})

	if !state.finished || state.won {
		t.Fatalf("round state = %+v, want loss on pulse contact", state)
	}
}

func TestRoundLosesWhenClockCompletes(t *testing.T) {
	game, state := startTestRound(t)
	clock := findEntity(t, game.World(), roundClock{}, timer.Timer{})

	if err := clock.Update(game.World(), func(t *timer.Timer) { t.Duration = time.Millisecond }); err != nil {
		t.Fatalf("shorten round clock: %v", err)
	}
	advanceTestFrame(t, game, &input.Snapshot{})

	if !state.finished || state.won {
		t.Fatalf("round state = %+v, want timeout loss", state)
	}
}

func startTestRound(t *testing.T) (*castrum.Game, *runState) {
	t.Helper()
	game, state, err := newGame()
	if err != nil {
		t.Fatalf("newGame: %v", err)
	}
	if err := game.Startup(); err != nil {
		t.Fatalf("Startup: %v", err)
	}

	start := &input.Snapshot{}
	start.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	advanceTestFrame(t, game, start)
	advanceTestFrame(t, game, &input.Snapshot{})
	if !state.active {
		t.Fatal("start input did not activate the round")
	}
	return game, state
}

func advanceTestFrame(t *testing.T, game *castrum.Game, snapshot *input.Snapshot) {
	t.Helper()
	game.Context().Input = snapshot
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatalf("Advance: %v", err)
	}
}

func findEntity(t *testing.T, world *core.World, components ...any) *core.Entity {
	t.Helper()
	query := core.NewQuery(world).With(components...)
	entry, ok := query.First()
	if !ok {
		t.Fatalf("no entity with components %v", components)
	}
	return core.NewEntity(entry.ID())
}
