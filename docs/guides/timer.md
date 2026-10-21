# Timers

Timers are countdowns stored on entities as `timer.Timer`. Castrum advances each running timer on fixed ticks. You can read progress and completion from the component, or react to a `timer.Completed` signal when an interval finishes. Timers are not removed automatically.

The [timer example](../../examples/timer/main.go) shows one-shot, repeating, and paused timers in a running game:

```text
go run ./examples/timer
```

## Create a timer

`timer.NewTimer(duration, repeating)` returns a timer that starts running. Durations are `time.Duration` values, so `3*time.Second` means three seconds of simulation time, regardless of how many fixed ticks it takes.

```go
countdown, err := game.World().NewEntity(
	timer.NewTimer(3*time.Second, false),
)
if err != nil {
	return err
}
```

Each entity can hold one `Timer`. To run several timers for the same actor, put each timer on its own entity and keep the entity handles.

Castrum registers the timer system when the game is created with `castrum.WithTimer()` (or `castrum.WithDefaultSystems()`, which also enables the animation and collision systems). A timer is validated when it enters storage, including when an update writes it back: its duration must be positive and elapsed time cannot be negative. Invalid timers are rejected with an error; `NewTimer` itself does not return an error.

## Pause and resume

`NewTimer` starts the countdown immediately. Pause or resume it by setting `Running` inside an entity update:

```go
err := countdown.Update(game.World(), func(t *timer.Timer) {
	t.Running = false // Pause without losing elapsed time.
})
if err != nil {
	return err
}
```

Set `Running` to `true` to continue from the saved elapsed time. Use `Restart` to begin again from zero instead:

```go
err := countdown.Update(game.World(), func(t *timer.Timer) {
	t.Restart()
})
if err != nil {
	return err
}
```

## Timer groups

Use a timer's `Group` field to put it in one named group. The `timer.Groups` resource controls whether timers in each group accumulate time and at what rate. `castrum.WithTimer()` provides the resource along with the timer system; `castrum.WithDefaultSystems()` includes it too.

Retrieve the resource after creating the game:

```go
groups := game.World().MustResource[*timer.Groups]()
```

`MustResource` is appropriate when the game was created with `WithTimer` or `WithDefaultSystems`, which provide the resource. If you register `timer.NewSystem()` yourself, provide a `*timer.Groups` resource (for example, one returned by `timer.NewGroups()`) as well; the timer system returns an error on its first update if it is missing.

Use the groups resource from the game's single-threaded loop; it is not safe for concurrent access.

### Assign timers to a group

Set `Timer.Group` before storing the timer, or change it later with `Entity.Update`. An empty group name leaves the timer ungrouped. Group names need no registration, and each timer belongs to at most one group; groups do not nest.

```go
combatTimer := timer.NewTimer(2*time.Second, true)
combatTimer.Group = "combat"
combat, err := game.World().NewEntity(combatTimer)
if err != nil {
	return err
}
```

### Pause or scale a group

Pausing and resuming changes the group state, not individual `Timer` components:

```go
groups.Pause("combat")
// Later:
groups.Resume("combat")

if err := groups.SetScale("ui", 0.5); err != nil {
	return err
}
```

A paused group does not accumulate time. When resumed, each running timer continues from its saved `Elapsed` value. A scale multiplies the fixed tick time accumulated by timers in that group; for example, `0.5` accumulates half as much time per tick. Use `Pause` to stop a group: `SetScale` returns an error for an empty group name or a scale of zero or less.

`Paused` and `Scale` can read a group that has not been configured: they report `false` and `1`, respectively, without creating group state. `Pause`, `Resume`, and a valid `SetScale` call create or update that state as needed.

Group settings do not override an individual timer's `Running` field. A timer with `Running == false` stays stopped even when its group resumes. Conversely, pausing a group holds an unfinished running timer without clearing `Running`. The timer system still needs to run on a fixed tick for any timer to advance.

Group pause does not record a completion; any existing `CompletedOn` value remains. Scaling changes how quickly a timer reaches its duration, but completion is still recorded on the fixed tick when it occurs, and `JustCompleted(ctx.Tick)` still reports only that tick.

## Read progress and completion

Read the stored component to inspect progress:

```go
t, ok := countdown.Component[timer.Timer](game.World())
if ok {
	fmt.Printf("%v / %v\n", t.Elapsed, t.Duration)
}
```

`Elapsed` advances by the fixed tick interval while `Running` is true. A one-shot stops at its duration when it completes and remains on the entity.

The component records completion as a tick stamp. Use `JustCompleted` for a one-tick check and `HasCompleted` for the lasting state of a finished one-shot:

```go
if t.JustCompleted(ctx.Tick) {
	// React to the completion on this tick.
}
if t.HasCompleted() {
	// The one-shot has finished, even on later ticks.
}
```

`CompletedOn` holds the tick of the most recent completion. It stays set after a one-shot finishes and clears when `Restart` is called. `JustCompleted(ctx.Tick)` is true only on the tick recorded in that field. The tick counter starts at one, so zero means the timer has never completed. For repeating timers, `HasCompleted` is always false; use `JustCompleted` to observe each tick on which they fire.

## Repeating timers

A repeating timer starts another interval immediately and carries elapsed overshoot forward. This prevents its schedule from drifting when the duration is not an exact multiple of the fixed tick interval.

If a repeating timer is shorter than one tick, it may complete multiple intervals during that tick. `CompletedOn` records the tick, not a count of completions within it, so `JustCompleted` reports whether it fired on that tick rather than how many times.

## Signals

When the timer system is enabled with `castrum.WithTimer()` or `castrum.WithDefaultSystems()`, it emits a `timer.Completed` signal when a timer completes. The signal carries the timer entity ID and fixed tick. A repeating timer emits at most one completion signal per fixed tick, even if it completes several intervals in that tick.

Use a consumer system when you want to react to a particular timer without checking its component every tick:

```go
hub := g.World().MustResource[*signal.Hub]()

if err := g.AddSystem(core.PhaseFixed, "game.countdown_complete", hub.Consumer(
	func(_ *core.Context, completed timer.Completed) error {
		if completed.Entity == countdown.ID() {
			// Start the next game action.
		}
		return nil
	},
)); err != nil {
	return err
}
```

The [signals guide](signals.md) explains the windows, ordering, and reader rules behind this pattern.

## More than one timer per actor

Because component storage has one slot per type on each entity, attach each timer to a separate entity. Keep those handles with the actor's other game state:

```go
cooldown, err := game.World().NewEntity(timer.NewTimer(2*time.Second, false))
if err != nil {
	return err
}
buff, err := game.World().NewEntity(timer.NewTimer(30*time.Second, false))
if err != nil {
	return err
}
```

Each timer advances independently. Destroy an entity when its timer is no longer needed.
