package timer

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

func newTimerHarness(t *testing.T) (*core.World, core.System) {
	t.Helper()
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Groups, error) {
		return NewGroups(), nil
	}); err != nil {
		t.Fatalf("provide timer groups: %v", err)
	}
	return world, NewSystem()
}

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

func TestSystem_OneShotCompletesOnBoundaryAndFreezes(t *testing.T) {
	world, sys := newTimerHarness(t)
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

	// Re-enabling a completed one-shot cannot make it fire again;
	// Restart is the only way to clear its completed state.
	if err := entity.Update(world, func(t *Timer) { t.Running = true }); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys, 3, dt)
	timer = timerOf(t, world, entity)
	if timer.Running || timer.CompletedOn != 2 || timer.Elapsed != timer.Duration {
		t.Errorf("completed one-shot should remain stopped and frozen, got running=%v tick=%d elapsed=%v",
			timer.Running, timer.CompletedOn, timer.Elapsed)
	}
}

func TestSystem_RestartResetsOneShot(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_RepeatingCarriesRemainder(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_RepeatingShorterThanTick(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_StoppedTimerHoldsItsProgress(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_MultipleTimersPerActor(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_EmptyWorld(t *testing.T) {
	world, sys := newTimerHarness(t)
	dt := 100 * time.Millisecond

	tick(t, world, sys, 1, dt)
	tick(t, world, sys, 2, dt)
	tick(t, world, sys, 3, dt)
}

func TestTimerCompletionReads(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestHasCompletedRepeatingNeverReadsCompleted(t *testing.T) {
	world, sys := newTimerHarness(t)
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

func TestSystem_GroupPauseHoldsAndResumeContinues(t *testing.T) {
	world, sys := newTimerHarness(t)
	groups, err := world.Resource[*Groups]()
	if err != nil {
		t.Fatal(err)
	}
	dt := 100 * time.Millisecond
	spawn := func(group string, running bool) *core.Entity {
		timer := NewTimer(time.Second, false)
		timer.Group = group
		timer.Running = running
		e, err := world.NewEntity(timer)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	combat := spawn("combat", true)
	ui := spawn("ui", true)
	free := spawn("", true)
	stopped := spawn("combat", false)

	groups.Pause("combat")
	for tickNumber := range 5 {
		tick(t, world, sys, uint64(tickNumber+1), dt)
	}
	for name, e := range map[string]*core.Entity{
		"combat": combat, "stopped in combat": stopped,
	} {
		if got := timerOf(t, world, e); got.Elapsed != 0 {
			t.Errorf("%s elapsed = %v after 5 ticks of a paused group, want 0", name, got.Elapsed)
		}
	}
	if want := 5 * dt; timerOf(t, world, ui).Elapsed != want || timerOf(t, world, free).Elapsed != want {
		t.Errorf("ui = %v, ungrouped = %v, want both advancing at %v", timerOf(t, world, ui).Elapsed, timerOf(t, world, free).Elapsed, want)
	}

	groups.Resume("combat")
	tick(t, world, sys, 6, dt)
	if got := timerOf(t, world, combat); got.Elapsed != dt {
		t.Errorf("combat elapsed = %v after resume, want %v - held time is not recovered", got.Elapsed, dt)
	}
	if got := timerOf(t, world, stopped); got.Running || got.Elapsed != 0 {
		t.Errorf("per-timer stop did not survive the group cycle: running=%v elapsed=%v", got.Running, got.Elapsed)
	}
}

func TestSystem_GroupScaleAdvancesProportionally(t *testing.T) {
	world, sys := newTimerHarness(t)
	groups, err := world.Resource[*Groups]()
	if err != nil {
		t.Fatal(err)
	}
	if err := groups.SetScale("slow", 0.5); err != nil {
		t.Fatal(err)
	}
	if err := groups.SetScale("fast", 2); err != nil {
		t.Fatal(err)
	}
	dt := 100 * time.Millisecond
	slow := NewTimer(time.Second, false)
	slow.Group = "slow"
	slowEntity, err := world.NewEntity(slow)
	if err != nil {
		t.Fatal(err)
	}
	fast := NewTimer(time.Second, true)
	fast.Group = "fast"
	fastEntity, err := world.NewEntity(fast)
	if err != nil {
		t.Fatal(err)
	}
	freeEntity, err := world.NewEntity(NewTimer(time.Second, false))
	if err != nil {
		t.Fatal(err)
	}

	for tickNumber := range 5 {
		tick(t, world, sys, uint64(tickNumber+1), dt)
	}
	if got := timerOf(t, world, slowEntity); got.Elapsed != 250*time.Millisecond || got.CompletedOn != 0 {
		t.Errorf("half-speed one-shot = %v elapsed / tick %d, want %v / 0",
			got.Elapsed, got.CompletedOn, 250*time.Millisecond)
	}
	fastTimer := timerOf(t, world, fastEntity)
	if !fastTimer.JustCompleted(5) || fastTimer.Elapsed != 0 {
		t.Errorf("double-speed repeating = tick %d elapsed %v, want completion on tick 5 with full carryover",
			fastTimer.CompletedOn, fastTimer.Elapsed)
	}
	if got := timerOf(t, world, freeEntity); got.Elapsed != 5*dt || got.CompletedOn != 0 {
		t.Errorf("ungrouped control = %v / tick %d, want %v / 0", got.Elapsed, got.CompletedOn, 5*dt)
	}
}

func TestGroups_RegistryContract(t *testing.T) {
	groups := NewGroups()
	if groups.Paused("unknown") || groups.Scale("unknown") != 1 {
		t.Error("unknown group should read unpaused at scale 1")
	}
	groups.Pause("combat")
	if !groups.Paused("combat") {
		t.Error("Pause should record the paused state")
	}
	groups.Resume("combat")
	if groups.Paused("combat") {
		t.Error("Resume should clear the paused state")
	}
	if err := groups.SetScale("combat", 0.5); err != nil || groups.Scale("combat") != 0.5 {
		t.Errorf("SetScale(0.5) = %v, scale %v, want no error at 0.5", err, groups.Scale("combat"))
	}
	for name, bad := range map[string]float64{
		"zero scale":     0,
		"negative scale": -1,
		"nan scale":      math.NaN(),
		"infinite scale": math.Inf(1),
	} {
		if err := groups.SetScale("combat", bad); err == nil {
			t.Errorf("SetScale with %s should fail", name)
		}
	}
	if err := groups.SetScale("", 1); err == nil {
		t.Error("SetScale with an empty name should fail")
	}
	if groups.Scale("combat") != 0.5 {
		t.Errorf("scale after rejected values = %v, want 0.5 unchanged", groups.Scale("combat"))
	}
}

func TestSystem_RequiresGroupsResource(t *testing.T) {
	sys := NewSystem()
	err := sys.Update(&core.Context{World: core.NewWorld(), Tick: 1, DeltaTime: time.Second})
	if err == nil || !strings.Contains(err.Error(), "timer groups") {
		t.Errorf("system without groups = %v, want an error naming the timer groups resource", err)
	}
}

// A finite but enormous scale can still overflow the tick contribution
// when multiplied out: the tick fails with an error naming the group
// instead of converting an unrepresentable value into elapsed time.
func TestSystem_UnrepresentableGroupScaleFailsTick(t *testing.T) {
	world, sys := newTimerHarness(t)
	groups, err := world.Resource[*Groups]()
	if err != nil {
		t.Fatal(err)
	}
	// 1e12 * 100ms overflows int64 nanoseconds by three orders of
	// magnitude. The scale is finite, so only the advancement guard
	// catches it.
	if err := groups.SetScale("huge", 1e12); err != nil {
		t.Fatal(err)
	}
	huge := NewTimer(time.Second, false)
	huge.Group = "huge"
	if _, err := world.NewEntity(huge); err != nil {
		t.Fatal(err)
	}

	err = sys.Update(&core.Context{World: world, Tick: 1, DeltaTime: 100 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "not representable") {
		t.Errorf("tick under an unrepresentable scale = %v, want an error naming the contribution", err)
	}
}
