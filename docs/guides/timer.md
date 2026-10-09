# Timers

Timers are countdowns stored on entities as `timer.Timer`. Castrum advances each running timer on fixed ticks; games read the component to observe progress and completion. Timers do not emit events or remove themselves.

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

## Read progress and completion

Read the stored component to inspect progress:

```go
t, ok := countdown.Component[timer.Timer](game.World())
if ok {
	fmt.Printf("%v / %v\n", t.Elapsed, t.Duration)
}
```

`Elapsed` advances by the fixed tick interval while `Running` is true. A one-shot stops at its duration when it completes and remains on the entity.

Completion is recorded as a tick stamp, not an event. Use `JustCompleted` for a one-tick reaction and `HasCompleted` for the lasting state of a finished one-shot:

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
