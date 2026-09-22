# Systems

A system is a unit of game logic: a function or type with an `Update(ctx *core.Context) error` method, registered to run in a specific [phase](the-loop.md#the-three-phases).

## Two ways to write one

For a quick function, wrap it in `core.SystemFunc`:

```go
game.AddSystem(core.PhaseFixed, "movement", core.SystemFunc(func(ctx *core.Context) error {
	// ctx gives you the world, tick/frame counters, and DeltaTime.
	return nil
}))
```

When the system carries state or benefits from a name, use a type:

```go
type MovementSystem struct {
	Gravity float64
}

func (m MovementSystem) Update(ctx *core.Context) error {
	// Use m.Gravity and ctx.World here.
	return nil
}

game.AddSystem(core.PhaseFixed, "movement", MovementSystem{Gravity: 9.8})
```

Both are the same to the engine. Start with `SystemFunc` and grow into a struct when the logic needs it.

## Registration

`game.AddSystem(phase, name, system)` binds a system into a phase's schedule:

- The **name** identifies the registration. It appears in error messages and will serve as the handle for future ordering constraints. An empty name returns an error.
- **Order** is registration order, within a phase. Register `input` before `movement` if `movement` reads what `input` wrote.

Errors returned from `AddSystem` are setup errors - handle them like any other startup failure:

```go
if err := game.AddSystem(core.PhaseFixed, "movement", MovementSystem{}); err != nil {
	panic(err)
}
```

## Errors stop the schedule

If a system returns an error, the schedule stops immediately - later systems in the same phase do not run - and the error is wrapped with the phase and system name:

```
fixed system "movement": <your error>
```

The error propagates up through the runner's `Run`, so one `panic` or `log.Fatal` in `main` is enough to handle every failure in the game. Use errors for the exceptional: a failed asset load, an inconsistent world state. Ordinary game outcomes (enemy off screen, empty inventory) are not errors.

## What lives in ctx

`core.Context` carries everything a system should need per step: `World` (shared state, see [resources](resources.md)), `Tick` and `Frame` counters, `DeltaTime`, and `Alpha`. The full field list is in the [core reference](../reference/core.md).
