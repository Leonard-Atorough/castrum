# Time in Castrum

Castrum separates elapsed frame time from fixed-step simulation time. Pause or scale the simulation to control gameplay; frame-phase systems can continue to run menus and other presentation while the simulation is paused.

```go
if game.Paused() {
	game.Resume()
} else {
	game.Pause()
}

if err := game.SetTimeScale(0.5); err != nil {
	return err
}
```

Put pause and resume controls in a frame-phase system, because fixed systems do not run while the simulation is paused. The [time-control example](../../../examples/time-control/main.go) demonstrates these controls alongside fixed-step movement.

## Choose a clock

| You want to...                                     | Use                                                    |
| -------------------------------------------------- | ------------------------------------------------------ |
| Pause, resume, or change simulation speed          | `game.Pause`, `game.Resume`, or `game.SetTimeScale`    |
| Advance gameplay by one fixed step                 | `ctx.DeltaTime` in a fixed system                      |
| Read simulation ticks or time                      | `ctx.Tick` in a fixed system, or `gametime.Simulation` |
| Animate presentation while paused                  | `ctx.DeltaTime` in a frame system                      |
| Read unclamped elapsed time supplied by the runner | `gametime.Real`                                        |
| Pause or scale a category of timers                | `gametime.Groups`; see the [timer guide](./timer.md)   |
| Build a day cycle or schedule                      | Derive it in game code from simulation ticks           |

`castrum.New` provides both time resources to the world. Retrieve them when a system needs more than the phase-specific context:

```go
sim := game.World().MustResource[*gametime.Simulation]()
real := game.World().MustResource[*gametime.Real]()
```

The resources serve different purposes:

- **Real time:** `Real.Frame` counts calls to `Advance`, including calls while paused. `Real.Delta` is the elapsed duration supplied to the latest call, and `Real.Elapsed` is the sum of those supplied durations. Neither is scaled or clamped. The Ebitengine runner supplies elapsed time between updates, but a custom runner or test can supply different durations. This is not an independent clock that tracks time before the runner starts or while no advances occur.
- **Simulation time:** `Simulation.Tick` counts fixed updates, `Delta` is the fixed interval, and `Paused` and `Scale` control the fixed loop. `Simulation.Elapsed()` returns `Tick` multiplied by the current `Delta`; it is not a history of elapsed intervals if `Delta` changes.
- **Timer group rates:** `gametime.Groups` pauses or scales named timer categories. Groups affect timers on fixed ticks, so they cannot make a timer advance while the entire simulation is paused.

The context is the view supplied to a system for its current phase:

- `ctx.Frame` is set from `Real.Frame` at the start of each advance.
- `ctx.Tick` is set from `Simulation.Tick` when a fixed tick starts. A direct change to `Simulation.Tick` does not immediately update the context.
- `ctx.DeltaTime` is the clamped, unscaled elapsed duration in frame systems, and `Simulation.Delta` in fixed systems. A frame system should not apply the clamp or simulation scale again.

Fixed steps give gameplay a consistent interval, but they do not by themselves guarantee deterministic replay: inputs, update ordering, frame clamping, and dropped backlog can affect the result. Elapsed-time-based frame motion can maintain the same speed across display rates, but the timing and number of its updates depend on the runner.

## Pause and resume

`Pause` stops adding time to the fixed-loop accumulator and stops fixed ticks; `Resume` allows them to continue. Pause preserves both the scale and the accumulator remainder. While an advance remains paused, its elapsed time is not accumulated for later recovery.

Frame-phase systems continue while paused, as do input polling and rendering with the Ebitengine runner. The always-registered timer and animation systems advance on fixed ticks, so they stop with the simulation. See the [timer](./timer.md) and [sprite animation](./spritesheets.md#timing) guides for their behavior.

Caller-owned [tweens](./tweening.md) follow the phase that advances them: fixed-phase movement pauses with simulation, while frame-phase tint or opacity can continue. Pass the phase's `DeltaTime` without applying time scale again.

Pausing preserves the accumulator, so [`Game.Alpha`](../../../castrum.go) stays at its current interpolation fraction while paused, provided the fixed interval does not change. A renderer using Alpha therefore keeps the last interpolated position until fixed updates resume.

Game pause does not automatically pause audio. To pause audio too, use `PauseAll` and `ResumeAll` on the game's audio `Mixer`; each play's pause mode determines how it responds. See the [audio guide](./audio.md) for details.

## Scale simulation time

`SetTimeScale` changes how much elapsed time is added to the fixed-loop accumulator. The default scale is `1`; a scale of `0.5` accumulates time at half speed, and `2` accumulates it twice as fast. Scale changes neither frame-phase `DeltaTime` nor the duration of a fixed tick.

The scale must be positive and finite. `SetTimeScale` returns an error for zero, negative, NaN, or infinite values without changing the existing scale. Use `Pause` to stop simulation time. Changing the scale while paused remembers it for when the game resumes.

An enormous finite scale can still produce an unrepresentable elapsed duration. `Advance` reports that contribution or an overflowing accumulator sum as an error without changing the accumulator. Frame systems have already run; their effects are not rolled back.

`Advance` runs pre-frame, frame, and post-frame systems before checking pause and adding scaled elapsed time. Changes made between calls control the upcoming advance; changes made in a frame-phase system affect fixed work later in the same call. If a frame-phase system resumes the game, that call can accumulate its full supplied elapsed duration, subject to the clamp. Pause or scale changes made during a fixed tick do not interrupt the fixed work already being processed in that advance.

## Configure the fixed loop

`castrum.New` accepts `WithFixedTPS`, `WithMaxFrameTime`, and `WithMaxTicksPerFrame`. Their defaults are 60 ticks per second, 250 milliseconds, and five ticks per advance. Invalid nonpositive values are rejected when the game is constructed; a tick rate that cannot produce a positive duration is also rejected.

`Real.MaxFrameTime` limits the elapsed duration used by frame systems and, before scaling, the fixed-loop accumulator. `Real.Delta` and `Real.Elapsed` retain the unclamped values. The clamp can be adjusted at runtime through the resource, but direct field writes are not validated: keep it positive. Likewise, keep `Simulation.Delta` positive and use `SetScale` or `game.SetTimeScale` instead of writing `Simulation.Scale` directly. Direct changes to the fixed interval also change how `Elapsed()` is calculated and how the existing accumulator is interpreted.

The tick limit prevents an unbounded catch-up loop. When the limit is reached, Castrum drops the remaining accumulator, including any fractional remainder. Consequently, a large elapsed time or high scale can produce fewer ticks than the scaled elapsed duration would otherwise imply.

## Save, restore, and reset time

You can snapshot fields from `Simulation` and `Real`, but they do not include the fixed-loop accumulator. Restoring those fields cannot reproduce the exact next-tick timing or interpolation fraction, and resetting `Simulation` does not clear the accumulator. Other gameplay state, timers, and input also need their own save or replay strategy.

There is no public accumulator restore or reset operation. For a fresh time state, create a new game rather than zeroing a live `Simulation`: its zero value has an invalid fixed interval and scale. Restore resource fields only with valid values, and account for the context synchronization described above.

## Inspect time in the debug overlay

Pass `ebitrun.WithDebugOverlay()` when creating the runner to draw `fps` and `tps` in the window. It also shows `paused` when the game is paused and `scale` when the time scale differs from `1`. The overlay is opt-in and only reports state; it does not change simulation timing.

## Run the example

From the repository root:

```text
go run ./examples/time-control
```

Use Space or P to pause and resume. Use the up and down arrows (or W and S) to speed up or slow down the simulation. The frame-phase controls remain responsive while paused.
