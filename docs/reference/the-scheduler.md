# The scheduler and the fixed loop

The runner supplies elapsed platform time; `Game.Advance` turns it into frame work and fixed simulation ticks. The fixed loop gives simulation a stable `DeltaTime` while rendering can interpolate between completed ticks.

## Startup and a display frame

`Game.Startup` resolves eager resources, runs the pre-startup schedule, then runs the startup schedule once. This lets initial state hooks queue scene work before the startup scene system applies requests. Calling `Startup` a second time returns an error.

A runner calls `Advance(elapsed)` once per display frame. `Advance` clamps the elapsed duration to `MaxFrameTime`, increments the frame counter, publishes that duration as frame `Context.DeltaTime`, then runs pre-frame systems followed by frame systems. The engine updates bound actions before user pre-frame systems; the frame scene system then applies requests before user frame systems. `Advance` adds the clamped duration to the fixed accumulator after frame work.

A frame can run zero fixed ticks. A slow frame runs as many fixed ticks as the accumulator can pay for, up to `MaxTicksPerFrame`. Each fixed tick increments `Context.Tick`, sets `Context.DeltaTime` to the fixed interval, runs the fixed schedule, and subtracts one interval from the accumulator.

The defaults are `FixedTPS=60`, `MaxFrameTime=250ms`, and `MaxTicksPerFrame=5`. When the tick cap is reached, the remaining backlog is dropped rather than carried into every later frame. These limits protect stability; they do not make an expensive system cheaper.

## Alpha

`Context.Alpha` is the remainder of the accumulator divided by the fixed interval, in `[0, 1)`. A renderer uses it to display a position between the previous and current simulation states. `render.Collector` interpolates transform position; a custom overlay that draws world-space state must apply the value itself if it needs the same effect.

When the catch-up cap drops the backlog, the accumulator is reset and alpha becomes zero. The renderer therefore never interpolates across time that the simulation intentionally discarded.

## Schedule order

Systems in one phase run in registration order. The engine defines ordering between phases: pre-startup before startup, and pre-frame before frame before fixed ticks. Within startup and frame schedules, `castrum.New` registers scene systems before user systems, so scene requests queued by pre-phase hooks are applied before ordinary startup or frame systems. A system error stops the current schedule and is returned through `Startup` or `Advance`, and for a normal runner through `g.Run`.

The engine's normal registrations are:

| System                  | Phase | Registered by            | Role                                                    |
| ----------------------- | ----- | ------------------------ | ------------------------------------------------------- |
| `engine.prev-transform` | fixed | always                   | Captures transform position before gameplay changes it. |
| `engine.animation`      | fixed | `WithAnimation`          | Advances animation state and updates sprites.           |
| `engine.collision`      | fixed | `WithCollision`         | Detects contacts and records their lifecycle.           |
| `engine.timer`          | fixed | `WithTimer`              | Advances timers and stamps completions.                 |
| `engine.input-tick`     | fixed | with bindings            | Publishes tick-scoped input edges.                      |
| `engine.input-update`   | pre-frame | with bindings            | Publishes the frame-scoped action view before state hooks. |
| `engine.audio`          | frame | the runner               | Reconciles audio source state with runner players.      |
| `scene-startup`         | startup | `WithScenes`           | Applies requests queued before startup.                 |
| `scene`                 | frame | `WithScenes`             | Applies queued scene requests before user frame systems. |

The optional subsystems are opt-in: `castrum.WithAnimation`, `castrum.WithCollision`, and `castrum.WithTimer` each register their system, and `castrum.WithDefaultSystems` registers all three. Every system's registration name is exported as a constant from the package that owns the system - `core.PrevTransformSystemName`, `input.FrameSystemName`, `input.TickSystemName`, `animation.SystemName`, `collision.SystemName`, `timer.SystemName`, `ebitrun.AudioSystemName` - so no call site hand-types the strings; the names identify registrations in error messages and will serve as the handles for future ordering constraints.

## System groups

`Game.AddSystemToGroup` assigns a system to a named group. Groups start
enabled, regardless of whether scenes are enabled. `Game.SetGroupEnabled`
changes a registered group's state across phases and applies it to systems
registered in the group later. It affects systems that have not yet run in
the current phase. An empty or unregistered group returns an error.

Scene loading, switching, updating, and removal do not change group states.
Use explicit group control when systems should pause together, or let systems
query for the components they need.

## Input timing

The runner publishes a raw snapshot before `Advance` runs. The frame input system updates the frame action view once per display frame. The fixed input system publishes the press and release edges accumulated since the previous tick, so every system in one fixed tick sees the same tick view.

Read frame-level state in frame systems and apply deterministic gameplay in fixed systems. The [input guide](../guides/input.md) shows the public action APIs.

## Signal rotation

The signal hub rotates with the loop: `Advance` brackets both pre-frame and frame work (`BeginFrame` before pre-frame, `EndFrame` after frame), then calls `EndTick` after each fixed run. A signal emitted during a fixed tick is visible to fixed systems registered after its emitter in that tick, and once to pre-frame and frame systems in the next display frame; then it is dropped. Frame signals live only through that display frame. The [signals guide](../guides/signals.md) covers the authoring side: emitting, readers, and consumer systems.

## Why the capture order matters

The previous-transform system must run before gameplay movement. It stores the prior position; gameplay then writes the current position; rendering interpolates between the pair using alpha. Registering a competing capture after movement would make both values equal and remove visible interpolation.

The [coordinate system](coordinate-system.md) explains the projection side of alpha, while [performance](../guides/performance.md) covers the cost and tuning consequences of the loop.
