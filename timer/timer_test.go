package timer

import (
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

// tick runs one fixed timer tick at the given tick number and
// interval.
func tick(t *testing.T, world *core.World, sys core.System, number uint64, dt time.Duration) {
	t.Helper()
	if err := sys.Update(&core.Context{World: world, Tick: number, DeltaTime: dt}); err != nil {
		t.Fatalf("timer tick %d: %v", number, err)
	}
}

func timerOf(t *testing.T, world *core.World, entity *core.Entity) Timer {
	t.Helper()
	timer, ok := entity.Component[Timer](world)
	if !ok {
		t.Fatalf("entity %d has no Timer", entity.ID())
	}
	return timer
}

func TestTimerValidate(t *testing.T) {
	valid := []Timer{
		NewTimer(time.Second, false),
		NewTimer(time.Nanosecond, true),
		// Elapsed == Duration is the completed one-shot state: legal.
		{Duration: time.Second, Elapsed: time.Second, CompletedOn: 12},
	}
	for _, timer := range valid {
		if err := timer.Validate(); err != nil {
			t.Fatalf("valid Timer rejected: %v", err)
		}
	}

	for name, timer := range map[string]Timer{
		"zero duration":     {Duration: 0},
		"negative duration": {Duration: -time.Second},
		"negative elapsed":  {Duration: time.Second, Elapsed: -time.Nanosecond},
	} {
		if err := timer.Validate(); err == nil {
			t.Errorf("Timer with %s should fail validation", name)
		}
	}
}

// The Validatable hook fires through the public surface: spawn
// rejects a non-positive duration, and a valid timer spawns running.
func TestTimerValidationAtSpawn(t *testing.T) {
	world := core.NewWorld()

	if _, err := world.NewEntity(NewTimer(0, false)); err == nil ||
		!strings.Contains(err.Error(), "Timer") {
		t.Errorf("zero duration at spawn = %v, want an error naming Timer", err)
	}

	entity, err := world.NewEntity(NewTimer(time.Second, true))
	if err != nil {
		t.Fatalf("valid timer should spawn: %v", err)
	}
	if timer := timerOf(t, world, entity); !timer.Running {
		t.Error("NewTimer should start running")
	}
}

// A one-shot completes on the exact tick its duration elapses and
// then freezes: no re-stamp, no further advance, the completion
// readable forever.
func TestSystem_OneShotCompletesOnBoundaryAndFreezes(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(2*dt, false))
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 1, dt)
	if timer := timerOf(t, world, entity); timer.JustCompleted(1) || timer.CompletedOn != 0 {
		t.Errorf("half-elapsed one-shot completed = %v / %d, want not yet", timer.JustCompleted(1), timer.CompletedOn)
	}

	tick(t, world, sys, 2, dt)
	timer := timerOf(t, world, entity)
	if !timer.JustCompleted(2) {
		t.Errorf("boundary completion = %v, want tick 2", timer.CompletedOn)
	}
	if timer.Running || timer.Elapsed != timer.Duration {
		t.Errorf("completed one-shot should be stopped at its duration, got running=%v elapsed=%v", timer.Running, timer.Elapsed)
	}

	// Later ticks change nothing: the stamp is the record of the fire.
	tick(t, world, sys, 3, dt)
	timer = timerOf(t, world, entity)
	if timer.CompletedOn != 2 || timer.Elapsed != timer.Duration {
		t.Errorf("completed one-shot should be frozen, got tick %d elapsed %v", timer.CompletedOn, timer.Elapsed)
	}
}

// Restart resets a completed one-shot to its beginning, clearing the
// completion stamp so "has it fired" reads false again.
func TestSystem_RestartResetsOneShot(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(2*dt, false))
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 1, dt)
	tick(t, world, sys, 2, dt)
	if !timerOf(t, world, entity).JustCompleted(2) {
		t.Fatal("test setup: one-shot should have completed on tick 2")
	}

	if err := entity.Update(world, func(t *Timer) { t.Restart() }); err != nil {
		t.Fatal(err)
	}
	if timer := timerOf(t, world, entity); timer.CompletedOn != 0 || timer.Elapsed != 0 {
		t.Fatalf("after Restart: stamp = %d elapsed = %v, want both zero", timer.CompletedOn, timer.Elapsed)
	}

	tick(t, world, sys, 3, dt)
	if timer := timerOf(t, world, entity); timer.JustCompleted(3) || timer.JustCompleted(2) {
		t.Errorf("restarted one-shot completed = %v, want the old stamp gone", timer.CompletedOn)
	}
	tick(t, world, sys, 4, dt)
	if timer := timerOf(t, world, entity); !timer.JustCompleted(4) {
		t.Errorf("restarted one-shot should complete again on tick 4, got stamp %d", timer.CompletedOn)
	}
}

// A repeating timer fires the instant its duration has fully
// elapsed - the exact boundary counts - and carries the overshoot
// into the next interval instead of dropping it.
func TestSystem_RepeatingCarriesRemainder(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 2 * time.Millisecond
	duration := 3 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(duration, true))
	if err != nil {
		t.Fatal(err)
	}

	// Tick 2 overshoots by a full step: 2*dt past the duration.
	tick(t, world, sys, 1, dt)
	if timer := timerOf(t, world, entity); timer.CompletedOn != 0 {
		t.Fatalf("tick 1 completed = %d, want none", timer.CompletedOn)
	}
	tick(t, world, sys, 2, dt)
	timer := timerOf(t, world, entity)
	if !timer.JustCompleted(2) {
		t.Fatalf("overshooting tick 2 should complete, stamp = %d", timer.CompletedOn)
	}
	if want := 2*dt - duration; timer.Elapsed != want {
		t.Errorf("remainder = %v, want %v carried into the next interval", timer.Elapsed, want)
	}

	// Tick 3 lands the elapsed time exactly on the boundary: it
	// completes - an exclusive boundary would miss it - and wraps
	// to zero.
	tick(t, world, sys, 3, dt)
	timer = timerOf(t, world, entity)
	if !timer.JustCompleted(3) {
		t.Errorf("exact-boundary tick 3 should complete, stamp = %d", timer.CompletedOn)
	}
	if timer.Elapsed != 0 {
		t.Errorf("boundary wrap elapsed = %v, want 0", timer.Elapsed)
	}
}

// A repeating timer shorter than the tick interval completes once
// per tick with its remainder bounded - it cannot accumulate drift.
func TestSystem_RepeatingShorterThanTick(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 3 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(dt/3, true))
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 1, dt)
	timer := timerOf(t, world, entity)
	if !timer.JustCompleted(1) {
		t.Fatalf("sub-tick repeating timer should complete on tick 1, stamp = %d", timer.CompletedOn)
	}
	if timer.Elapsed >= timer.Duration {
		t.Errorf("sub-tick repeating timer elapsed = %v, want the wrapped remainder below %v", timer.Elapsed, timer.Duration)
	}

	tick(t, world, sys, 2, dt)
	if timer := timerOf(t, world, entity); !timer.JustCompleted(2) {
		t.Errorf("sub-tick repeating timer should complete again on tick 2, stamp = %d", timer.CompletedOn)
	}
}

// A stopped timer does not advance, and resuming by setting Running
// continues from the same elapsed time.
func TestSystem_StoppedTimerHoldsItsProgress(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(3*dt, false))
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 1, dt)
	// Pausing is writing the Running field; there is no stop method
	// to wrap it.
	if err := entity.Update(world, func(t *Timer) { t.Running = false }); err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 2, dt)
	tick(t, world, sys, 3, dt)
	timer := timerOf(t, world, entity)
	if timer.CompletedOn != 0 || timer.Elapsed != dt {
		t.Errorf("stopped timer advanced: stamp = %d elapsed = %v, want held at dt", timer.CompletedOn, timer.Elapsed)
	}

	if err := entity.Update(world, func(t *Timer) { t.Running = true }); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys, 4, dt)
	tick(t, world, sys, 5, dt)
	if timer := timerOf(t, world, entity); !timer.JustCompleted(5) {
		t.Errorf("resumed timer should complete on tick 5 from its held progress, stamp = %d", timer.CompletedOn)
	}
}

// The multi-timer pattern: one actor with no Timer of its own, two
// timer entities serving it, advancing independently - the actor's
// cooldown and pulse coexist.
func TestSystem_MultipleTimersPerActor(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	actor, err := world.NewEntity(core.Transform{})
	if err != nil {
		t.Fatal(err)
	}
	cooldown, err := world.NewEntity(NewTimer(2*dt, false))
	if err != nil {
		t.Fatal(err)
	}
	pulse, err := world.NewEntity(NewTimer(dt, true))
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys, 1, dt)
	if got := timerOf(t, world, pulse).CompletedOn; got != 1 {
		t.Errorf("pulse stamp = %d, want 1", got)
	}
	if got := timerOf(t, world, cooldown).CompletedOn; got != 0 {
		t.Errorf("cooldown stamp = %d, want none yet", got)
	}

	tick(t, world, sys, 2, dt)
	if got := timerOf(t, world, cooldown).CompletedOn; got != 2 {
		t.Errorf("cooldown stamp = %d, want 2", got)
	}
	if got := timerOf(t, world, pulse).CompletedOn; got != 2 {
		t.Errorf("pulse stamp = %d, want 2 again", got)
	}

	// Tick 3: the repeating pulse fires again; the one-shot cooldown
	// stays frozen at its only completion.
	tick(t, world, sys, 3, dt)
	if got := timerOf(t, world, pulse).CompletedOn; got != 3 {
		t.Errorf("pulse stamp = %d, want 3", got)
	}
	if got := timerOf(t, world, cooldown).CompletedOn; got != 2 {
		t.Errorf("cooldown stamp = %d, want frozen at 2", got)
	}

	// The actor never needed a Timer; the timers reference nothing
	// but their own entities.
	if actor.HasComponent[Timer](world) {
		t.Error("actor should carry no Timer of its own")
	}
}

// The engine registers the reconciler for every game: a world
// without timers ticks clean.
func TestSystem_EmptyWorld(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	tick(t, world, sys, 1, dt)
	tick(t, world, sys, 2, dt)
	tick(t, world, sys, 3, dt)
}

// JustCompleted reads true only on the tick of the completion, while
// HasCompleted keeps reading true after it - the record that stays
// until Restart clears the stamp.
func TestTimerCompletionReads(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(2*dt, false))
	if err != nil {
		t.Fatal(err)
	}

	// Before any completion: neither read reports one.
	timer := timerOf(t, world, entity)
	if timer.HasCompleted() || timer.JustCompleted(1) {
		t.Error("a fresh timer should not report any completion")
	}

	tick(t, world, sys, 1, dt)
	tick(t, world, sys, 2, dt)

	// On the completion tick both reads agree, and ticks before the
	// completion never read as just-completed.
	timer = timerOf(t, world, entity)
	if !timer.JustCompleted(2) || timer.JustCompleted(1) || !timer.HasCompleted() {
		t.Errorf("after completing on tick 2: just(2)=%v just(1)=%v has=%v, want edge on 2 and has true",
			timer.JustCompleted(2), timer.JustCompleted(1), timer.HasCompleted())
	}

	// On later ticks the edge is gone but the record survives,
	// without anything needing to reset.
	tick(t, world, sys, 3, dt)
	timer = timerOf(t, world, entity)
	if timer.JustCompleted(3) || !timer.HasCompleted() {
		t.Error("after the completion tick the edge should be gone while HasCompleted stays true")
	}

	// Restart clears both readings.
	if err := entity.Update(world, func(t *Timer) { t.Restart() }); err != nil {
		t.Fatal(err)
	}
	if timer = timerOf(t, world, entity); timer.HasCompleted() {
		t.Error("after Restart HasCompleted should read false")
	}
}

// A repeating timer never holds a completed state: it fires and
// starts its next interval, so HasCompleted stays false while
// JustCompleted reports each fire.
func TestHasCompletedRepeatingNeverReadsCompleted(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	dt := 100 * time.Millisecond

	entity, err := world.NewEntity(NewTimer(dt, true))
	if err != nil {
		t.Fatal(err)
	}

	for tickNumber := uint64(1); tickNumber <= 3; tickNumber++ {
		tick(t, world, sys, tickNumber, dt)
		timer := timerOf(t, world, entity)
		if !timer.JustCompleted(tickNumber) {
			t.Errorf("tick %d: repeating timer should fire every tick, stamp = %d", tickNumber, timer.CompletedOn)
		}
		if timer.HasCompleted() {
			t.Errorf("tick %d: a repeating timer never reads as completed", tickNumber)
		}
	}
}
