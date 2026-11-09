# Clock

`clock.Clock` is the engine's virtual game time. It advances on fixed ticks, independently of the fixed-timestep engine tick counter, so game time can be paused, scaled, and restored without touching the simulation loop's contract.

Unlike the fixed-loop pause and scale described in [time control](time-control.md), which stop or stretch the whole simulation, the clock is a layer *inside* the simulation: fixed systems keep running while the clock is paused. Use the loop controls to pause the game; use the clock when gameplay logic needs its own notion of elapsed time.

The clock carries no calendar structure. Days, phases, and schedules are game-level concepts derived from `Clock.Tick`; the engine only provides the pausable, saveable tick counter they are built on.

## Register the clock

```go
if err := game.AddSystem(core.PhaseFixed, clock.SystemName, clock.NewSystem()); err != nil {
	return err
}
```

`clock.NewSystem()` provisions the `*clock.Clock` resource on its first update, so a game that wants the defaults only registers the system. To configure the clock up front, provide the resource yourself:

```go
if err := game.World().Provide(func(*core.World) (*clock.Clock, error) {
	return clock.New(), nil
}); err != nil {
	return err
}
```

Any system can then read it:

```go
c, err := ctx.World.Resource[*clock.Clock]()
if err != nil {
	return err
}
day := c.Tick / ticksPerDay
```

## Pause and scale

The clock has two controls, both safe to change at any time:

- `Paused` stops the clock from accumulating time. Fixed systems keep running.
- `Scale` sets the clock's relative speed. `1` advances one game tick per fixed tick, `2` advances two, `0.5` accumulates one game tick every other fixed tick, and `0` stops the clock. Fractional scales accumulate their remainder, so `0.5` never loses half-ticks. Scale must be a finite non-negative number; the clock system reports an error otherwise.

```go
c.Paused = true  // stop game time
c.Scale = 2       // double game time
```

## Save and restore

A clock is fully defined by its exported fields: `Tick`, `Paused`, and `Scale`. Reading them snapshots the clock; writing them to a fresh `*clock.Clock` restores it. The internal fractional carry is deliberately not part of a save: a partially accumulated fraction does not matter after a restore.

## What the clock is not

- It is not a scheduler. "Run this in N ticks" belongs to tick-stamped scheduling, not the clock.
- It is not a state machine. React to game-time boundaries with your own systems reading the resource, or drive a `state.Machine` from them.
- It is not wall-clock time. The clock counts ticks, not seconds; converting ticks to durations is a game-level constant.
