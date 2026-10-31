package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/scene"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/state"
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

	if state.score != coreGoal {
		t.Fatalf("score = %d, want %d", state.score, coreGoal)
	}
	assertPendingPhase(t, state.machine, phaseWon)
	advanceTestFrame(t, game, &input.Snapshot{})
	if !state.machine.Is(phaseWon) {
		t.Fatalf("phase = %v, want won", state.machine.Current())
	}
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

	assertPendingPhase(t, state.machine, phaseLost)
	advanceTestFrame(t, game, &input.Snapshot{})
	if !state.machine.Is(phaseLost) {
		t.Fatalf("phase = %v, want loss on pulse contact", state.machine.Current())
	}
}

func TestRoundLosesWhenClockCompletes(t *testing.T) {
	game, state := startTestRound(t)
	clock := findEntity(t, game.World(), roundClock{}, timer.Timer{})

	if err := clock.Update(game.World(), func(t *timer.Timer) { t.Duration = time.Millisecond }); err != nil {
		t.Fatalf("shorten round clock: %v", err)
	}
	advanceTestFrame(t, game, &input.Snapshot{})

	assertPendingPhase(t, state.machine, phaseLost)
	advanceTestFrame(t, game, &input.Snapshot{})
	if !state.machine.Is(phaseLost) {
		t.Fatalf("phase = %v, want timeout loss", state.machine.Current())
	}
}

func TestBriefingStartsThroughStateResource(t *testing.T) {
	game, progress, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	machine := game.World().MustResource[*state.Machine[gamePhase]]()
	if machine != progress.machine || !machine.Is(phaseBriefing) {
		t.Fatal("configured state machine was not provided")
	}
	director := game.World().MustResource[*scene.Director]()
	if _, ok := director.ByName("briefing"); ok {
		t.Fatal("briefing was loaded before the initial state hook")
	}
	if err := game.Startup(); err != nil {
		t.Fatal(err)
	}
	if _, ok := director.ByName("briefing"); !ok {
		t.Fatal("initial enter hook did not load the briefing during startup")
	}
	advanceTestFrame(t, game, &input.Snapshot{})
	if !machine.Is(phaseBriefing) || progress.score != 0 {
		t.Fatal("idle briefing changed game progress")
	}
	start := &input.Snapshot{}
	start.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	advanceTestFrame(t, game, start)
	assertPendingPhase(t, machine, phasePlaying)
	if _, ok := director.ByName("arena"); ok {
		t.Fatal("start request switched scenes before pre-frame")
	}
	advanceTestFrame(t, game, &input.Snapshot{})
	if !machine.Is(phasePlaying) {
		t.Fatal("start transition was not applied")
	}
	if _, ok := director.ByName("arena"); !ok {
		t.Fatal("playing enter hook did not switch scenes in the same frame")
	}
}

func TestRestartResetsRound(t *testing.T) {
	for _, won := range []bool{false, true} {
		t.Run(map[bool]string{false: "loss", true: "win"}[won], func(t *testing.T) {
			game, progress := startTestRound(t)
			progress.score = 3
			game.World().MustResource[*signal.Hub]().Emit(roundEnded{won: won})
			advanceTestFrame(t, game, &input.Snapshot{})
			advanceTestFrame(t, game, &input.Snapshot{})
			director := game.World().MustResource[*scene.Director]()
			if _, ok := director.ByName("result"); !ok {
				t.Fatal("round did not show a result")
			}
			// Start cannot skip the result's restart-to-briefing flow.
			start := &input.Snapshot{}
			start.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
			advanceTestFrame(t, game, start)
			if _, pending := progress.machine.Pending(); pending {
				t.Fatal("start accepted on result screen")
			}
			restart := &input.Snapshot{}
			restart.Keys[input.KeyEnter] = input.State{Pressed: true, Held: true}
			advanceTestFrame(t, game, restart)
			assertPendingPhase(t, progress.machine, phaseBriefing)
			advanceTestFrame(t, game, &input.Snapshot{})
			if !progress.machine.Is(phaseBriefing) {
				t.Fatal("restart did not enter briefing")
			}
			if _, ok := director.ByName("arena"); ok {
				t.Fatal("previous arena survived restart")
			}
			advanceTestFrame(t, game, start)
			advanceTestFrame(t, game, &input.Snapshot{})
			if !progress.machine.Is(phasePlaying) || progress.score != 0 {
				t.Fatalf("restarted round phase = %v, score = %d", progress.machine.Current(), progress.score)
			}
			got := 0
			for range core.NewQuery(game.World()).With(dataCore{}).Execute() {
				got++
			}
			if got != coreGoal {
				t.Fatalf("restarted cores = %d, want %d", got, coreGoal)
			}
		})
	}
}

func TestFinalCoreBeatsTimeoutAndStopsCatchUpTicks(t *testing.T) {
	game, progress := startTestRound(t)
	progress.score = coreGoal - 1
	clock := findEntity(t, game.World(), roundClock{}, timer.Timer{})
	if err := clock.Update(game.World(), func(t *timer.Timer) { t.Duration = time.Millisecond }); err != nil {
		t.Fatal(err)
	}
	lateUpdates := 0
	if err := game.AddSystemToGroup(core.PhaseFixed, roundGroup, "test.late", core.SystemFunc(func(*core.Context) error {
		lateUpdates++
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	game.World().MustResource[*signal.Hub]().Emit(coreCollected{})
	game.Context().Input = &input.Snapshot{}
	if err := game.Advance(100 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	assertPendingPhase(t, progress.machine, phaseWon)
	if lateUpdates != 0 || progress.score != coreGoal {
		t.Fatalf("late updates = %d, score = %d, want 0 and %d", lateUpdates, progress.score, coreGoal)
	}
	advanceTestFrame(t, game, &input.Snapshot{})
	if !progress.machine.Is(phaseWon) {
		t.Fatal("timeout overrode the winning outcome")
	}
}

func assertPendingPhase(t *testing.T, machine *state.Machine[gamePhase], want gamePhase) {
	t.Helper()
	if next, ok := machine.Pending(); !ok || next != want {
		t.Fatalf("pending phase = %v, %v, want %v, true", next, ok, want)
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
	if !state.machine.Is(phasePlaying) {
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
