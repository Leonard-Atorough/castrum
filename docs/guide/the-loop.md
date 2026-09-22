# The Loop

Castrum separates **how often you simulate** from **how often you draw**. Simulation runs at a fixed rate you choose; rendering runs at whatever rate the display and platform allow. This is the classic fixed-timestep pattern, and it is the single most important idea in the engine.

## Why fixed timestep

Game logic tied to display framerate changes behavior with the machine: the same character jumps a different height on a 60 Hz laptop and a 144 Hz monitor. A fixed tick rate makes simulation deterministic in step size and independent of rendering speed.

## The three phases

Every system is registered into one of three phases:

| Phase | Runs | Use for |
|---|---|---|
| `core.PhaseStartup` | once, before the first frame | loading, world setup |
| `core.PhaseFrame` | once per display frame | input, timers, UI |
| `core.PhaseFixed` | at the fixed rate (default 60/s) | movement, physics, game rules |

## What happens in a frame

Each display frame, the runner calls `game.Advance(elapsed)` with the real time since the previous frame. The engine then:

1. Clamps `elapsed` to `MaxFrameTime` (default 250ms) - a stall (window drag, slow disk) cannot dump unbounded time into the simulation.
2. Runs the `PhaseFrame` systems once.
3. Runs as many fixed ticks as the accumulated time allows - zero on a fast frame, several after a stall.
4. If a single frame owes more than `MaxTicksPerFrame` (default 5) ticks, the backlog is dropped instead of spiraling: the simulation falls behind rather than freezing trying to catch up.

A helpful mental model: the frame phase reads the world; the fixed phase advances it.

## DeltaTime

`ctx.DeltaTime` means different things per phase:

- In `PhaseFrame`: the real elapsed time since the previous frame. Variable.
- In `PhaseFixed`: always exactly `1 / FixedTPS` seconds. Constant - so simple games can mostly ignore it.

```go
game.AddSystem(core.PhaseFrame, "input", core.SystemFunc(func(ctx *core.Context) error {
	// Runs once per display frame.
	_ = ctx.DeltaTime // real elapsed time
	return nil
}))

game.AddSystem(core.PhaseFixed, "movement", core.SystemFunc(func(ctx *core.Context) error {
	// Runs at the fixed rate.
	_ = ctx.DeltaTime // always 1/FixedTPS
	return nil
}))
```

## Choosing a tick rate

`FixedTPS` defaults to 60. Use the default unless you have a reason; when you do:

```go
game := castrum.New(castrum.WithFixedTPS(30))
```

A physics-heavy game might tick at 120 for stability; a turn-based or slow-paced game might tick at 10 and save the CPU. All loop options and their defaults are listed in [options reference](../reference/options.md).

## Alpha and interpolation

When the display refreshes faster than the tick rate, the most recent tick is already a fraction of a tick old. `game.Alpha()` returns that fraction in `[0, 1)`. Draw functions can use it to interpolate positions between the last two ticks, so motion stays smooth at any display rate. See [runners](runner.md#interpolated-drawing).
