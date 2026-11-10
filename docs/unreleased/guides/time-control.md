# Time in Castrum

Castrum has two time resources, one per layer of the hierarchy. Real elapsed
time flows in from the runner and becomes simulation time - the fixed loop's
state - and both are exposed as world resources:

```text
real time: the gametime.Real resource - Frame, Elapsed, Delta,
        |  MaxFrameTime. Wall-clock time: unaffected by pause, scale,
        |  or the clamp. The time of things that stay true while the
        |  game is paused.
        v
simulation time: the gametime.Simulation resource - Tick, Delta, Paused,
        |         Scale. The loop reads Paused and Scale on every advance
        |         and increments Tick on every fixed tick.
        v
group rates: gametime.Groups pause and scale individual timer categories
```

Each layer is a distinct concept, not a redundant knob:

- **Real time** is wall-clock time, exposed as the `gametime.Real`
  resource: `Frame` counts advances since the game started (it increments
  while paused), `Elapsed` accumulates raw time since the game started,
  `Delta` is the current advance's real elapsed time, and `MaxFrameTime` is
  the runtime-adjustable clamp on how much real time one advance may feed the
  simulation. Real time cannot be paused or scaled - that is its purpose.
  It is the time of things that stay true while the game is paused: menu
  animations, playtime counters, session telemetry.
- **Simulation time** is how fast the world flows: pause, slow motion, and
  fast-forward, plus the tick counter. It lives in the
  `gametime.Simulation` resource - `game.Pause`, `Resume`, `Paused`,
  `TimeScale`, and `SetTimeScale` read and write it - so any system can read
  it, saves can snapshot it, and tests can drive it directly. `Delta` is the
  fixed tick interval and `Elapsed()` returns total simulation time as a
  duration (`Tick` ticks of `Delta`).
- **Group rates** scale countdowns per category - dialogue timers can run at
  a different speed than world timers. See the [timer guide](./timer.md).

The time resources are state: they exist between updates, and they are what
a save snapshots. `core.Context` is the per-update view: what is true during
this specific update. Its time fields are assigned from the resources on every
update, so they cannot drift:

- `ctx.Frame` mirrors `Real.Frame`.
- `ctx.Tick` mirrors `Simulation.Tick` during fixed updates.
- `ctx.DeltaTime` mirrors `Simulation.Delta` during fixed updates, and is the
  real delta clamped by `MaxFrameTime` during frame updates. The frame-phase
  value is derived rather than mirrored - the clamp lives at the boundary
  where real time feeds the simulation - which is also why a frame system
  never applies the clamp itself.

A system that runs in both phases reads `ctx.DeltaTime` and gets the correct
time in each. The resources are the source of truth; the context fields are
the loop handing the world its current position.

Two rules keep the model deterministic:

- **Scale affects how often ticks occur, never how long a tick is.** A
  fixed-phase system always sees exactly `Simulation.Delta` per tick, at
  every scale.
- **Gameplay logic reads simulation time; real time is for things that stay
  true while the game is paused.** Real time is frame-rate dependent and
  therefore not deterministic - a mechanic that advances by wall-clock time
  runs faster on a faster display.

There is deliberately no third, in-simulation clock. Game-meaning time - day
counters, phase cycles, schedules - is derived from `Simulation.Tick` in
game code, the same way Bevy and Unity games build their day cycles on the
engine's virtual time. If a derived pattern proves common enough across
games, it can be promoted to an engine utility later.

The completed model in one table:

| Layer | Resource | Context view |
| --- | --- | --- |
| Real - wall clock, pause-immune, nondeterministic | `gametime.Real`: `Frame`, `Elapsed`, `Delta`, `MaxFrameTime` | `ctx.Frame`, `ctx.DeltaTime` (frame phases, clamped) |
| Simulation - fixed loop, deterministic | `gametime.Simulation`: `Tick`, `Delta`, `Elapsed()`, `Paused`, `Scale` | `ctx.Tick`, `ctx.DeltaTime` (fixed phase) |
| Group rates - per-category timers | `gametime.Groups` | - |

## Which control do I use?

| You want to... | Use |
| --- | --- |
| Pause the game (menus, input, and rendering continue) | `game.Pause()` or `Simulation.Pause()` |
| Speed up or slow down the simulation | `game.SetTimeScale` or `Simulation.SetScale` |
| Read elapsed simulation ticks in a system | `ctx.Tick` or the `Simulation` resource |
| Read total simulation time as a duration | `Simulation.Elapsed()` |
| Read the fixed tick interval in a fixed system | `Simulation.Delta` (or `ctx.DeltaTime`) |
| Read whether the game is paused or scaled in a system | the `Simulation` resource |
| Time something that survives pause (menus, playtime, telemetry) | `gametime.Real` - `Frame`, `Elapsed`, and `Delta` |
| Adjust the stall guard at runtime | `Real.MaxFrameTime` |
| Save or restore the time state | snapshot the `Simulation` and `Real` resources' fields |
| Reset time (tests, replays, new game) | reset the `Simulation` resource |
| Slow one category of timers | timer groups via `Groups.SetScale` |
| A day cycle, schedule, or independently frozen time | derive it in game code from `Simulation.Tick` |

## Pause and resume

Call `Pause` to stop fixed ticks and `Resume` to continue them:

```go
if game.Paused() {
	game.Resume()
} else {
	game.Pause()
}
```

Pause does not reset the time scale or discard the fixed-loop accumulator.
When the game resumes, the next fixed tick is due after the remainder that was
already accumulated plus enough new simulation time to reach one tick. Time
spent paused is not recovered.

Frame-phase systems still run while paused. Put pause-menu and resume controls
in the frame phase; a fixed-phase system cannot resume a paused game because
fixed updates are stopped. The frame phase receives unscaled elapsed time
(clamped by `MaxFrameTime`), and input polling and rendering continue. The
always-registered timer and animation systems advance on fixed ticks, so they
stop with the simulation. See the [timer](./timer.md) and
[animation](./animation.md) guides for their behavior.

Caller-owned [tweens](tweening.md) follow the phase that advances them:
fixed-phase movement pauses with simulation, while frame-phase tint or opacity
can continue. Pass the phase's `DeltaTime` without applying time scale again.

Pausing preserves the accumulator, so [`Game.Alpha`](../../../castrum.go) stays
at its current interpolation fraction while paused. A renderer using Alpha
therefore keeps the last interpolated position until fixed updates resume.

## Scale simulation time

`SetTimeScale` changes how much elapsed time is added to the fixed-loop
accumulator. The default scale is `1`; a scale of `0.5` accumulates time at
half speed, and `2` accumulates it twice as fast:

```go
if err := game.SetTimeScale(0.5); err != nil {
	return err
}
```

The scale does not change frame-phase `DeltaTime`, nor the fixed tick interval
seen by fixed-phase systems. A value of zero or less returns an error; use
`Pause` to stop simulation time without discarding accumulated time. Changing
the scale while paused remembers it for when the game resumes.

`Advance` runs frame-phase systems before its fixed ticks. Changes made between
calls control the upcoming advance; changes made by a frame-phase system take
effect for the fixed work later in that same call. At high scales, the
`MaxTicksPerFrame` limit still applies: if the limit is reached, Castrum drops
the remaining backlog rather than running an unbounded number of ticks.

Game pause does not automatically pause audio. To pause audio too, use
`PauseAll` and `ResumeAll` on the game's audio `Mixer`; each play's pause mode
determines how it responds. See the [audio guide](./audio.md) for details.

## How this compares to other engines

Engines with a time story separate two orthogonal concerns: *stepping* (how
often logic runs, decoupled from the display rate) and *flow* (pause and
speed). Bevy names them `Time<Fixed>` and `Time<Virtual>` and derives fixed
updates from virtual time; Castrum composes the same two concerns the same
way - the fixed loop's accumulator is fed by scaled, pausable elapsed time -
but expresses the layers as things you already know: the `gametime.Real`
resource, the `gametime.Simulation` resource, and timer groups. There is no
separate clock-context type to declare; each layer is a value a system
reads. Bevy ships no in-simulation game clock either: day cycles and
schedules are game code on top of virtual time there, and the same is true
here.

## Inspect time in the debug overlay

Pass `ebitrun.WithDebugOverlay()` when creating the runner to draw `fps` and
`tps` in the window. It also shows `paused` when the game is paused and `scale`
when the time scale differs from `1`. The overlay is opt-in and only reports
state; it does not change simulation timing.

## Run the example

From the repository root:

```text
go run ./examples/time-control
```

Use Space or P to pause and resume. Use the up and down arrows (or W and S) to
speed up or slow down the simulation. The frame-phase controls remain
responsive while paused.
