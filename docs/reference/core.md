# Core Reference

The `core` package holds the types game logic sees every tick: phases, the context, and the system interface.

## Phases

`core.Phase` identifies when a system runs.

| Constant | Runs | Use for |
|---|---|---|
| `core.PhaseStartup` | once, before the first frame | loading, world setup |
| `core.PhaseFrame` | once per display frame, before fixed ticks | input, timers, UI |
| `core.PhaseFixed` | at the fixed simulation rate | movement, physics, game rules |

## Context

`core.Context` is passed to every system and draw function.

| Field | Type | Meaning |
|---|---|---|
| `World` | `*core.World` | the game world; see the [world reference](world.md) |
| `Tick` | `uint64` | fixed simulation steps since startup |
| `Frame` | `uint64` | display frames since startup |
| `DeltaTime` | `time.Duration` | elapsed frame time in `PhaseFrame`; the constant tick interval in `PhaseFixed` |
| `Alpha` | `float64` | fixed-loop remainder in `[0, 1)`, for interpolation; set by the runner for draw functions |

## System

```go
type System interface {
	Update(ctx *Context) error
}

// SystemFunc adapts an ordinary function:
core.SystemFunc(func(ctx *core.Context) error { return nil })
```

A returned error stops the schedule and propagates to the runner's `Run`, wrapped with the phase and registration name. See [systems](../guide/systems.md).
