# The scheduler and the fixed loop

The runner supplies elapsed platform time; `Game.Advance` turns it into frame work and fixed simulation ticks. The fixed loop gives simulation a stable `DeltaTime` while rendering can interpolate between completed ticks.

## Startup and a display frame

`Game.Startup` resolves eager resources, then runs the pre-startup, startup, and post-startup schedules once. This lets initial state hooks queue scene work before the startup scene system applies requests. Calling `Startup` a second time returns an error.

A runner calls `Advance(elapsed)` once per display frame. `Advance` clamps the elapsed duration to `MaxFrameTime`, increments the frame counter, publishes that duration as frame `Context.DeltaTime`, then runs pre-frame, frame, and post-frame systems in order. The engine's input and scene systems normally run before systems added later to their phases, unless a system is inserted before them. `Advance` adds the clamped duration to the fixed accumulator after frame work.

A frame can run zero fixed ticks. A slow frame runs as many fixed ticks as the accumulator can pay for, up to `MaxTicksPerFrame`. Each fixed tick increments `Context.Tick`, sets `Context.DeltaTime` to the fixed interval, runs pre-fixed, fixed, and post-fixed systems in order, and subtracts one interval from the accumulator.

The defaults are `FixedTPS=60`, `MaxFrameTime=250ms`, and `MaxTicksPerFrame=5`. When the tick cap is reached, the remaining backlog is dropped rather than carried into every later frame. These limits protect stability; they do not make an expensive system cheaper.

## Alpha

`Context.Alpha` is the remainder of the accumulator divided by the fixed interval, in `[0, 1)`. A renderer uses it to display a position between the previous and current simulation states. `render.Collector` interpolates transform position; a custom overlay that draws world-space state must apply the value itself if it needs the same effect.

When the catch-up cap drops the backlog, the accumulator is reset and alpha becomes zero. The renderer therefore never interpolates across time that the simulation intentionally discarded.

## Schedule order

Systems added normally to one phase run in registration order. The `castrum.Before` and `castrum.After` registration options insert a new regular system relative to an existing regular system in that phase; they do not move existing systems. The engine defines ordering between phases: pre-startup, startup, post-startup; pre-frame, frame, post-frame; then pre-fixed, fixed, post-fixed for every due tick. Within startup and frame schedules, `castrum.New` registers scene systems before user systems normally added later, so scene requests queued by pre-phase hooks are applied before those systems. Relative insertion can put a user system before an ordinary engine system, but cannot target the private signal boundary hooks. Startup-phase registrations are rejected once `Startup` begins. A system error stops the current schedule and is returned through `Startup` or `Advance`, and for a normal runner through `g.Run`.

The engine's normal registrations are:

| System                  | Phase | Registered by            | Role                                                    |
| ----------------------- | ----- | ------------------------ | ------------------------------------------------------- |
| `engine.prev-transform` | fixed | always                   | Captures transform position before normally appended gameplay systems. |
| `engine.animation`      | fixed | always on                | Advances animation state and updates sprites.           |
| `engine.collision`      | fixed | always on                | Detects contacts and records their lifecycle.           |
| `engine.timer`          | fixed | always on                | Advances timers and stamps completions.                 |
| `engine.input-tick`     | fixed | with bindings            | Publishes tick-scoped input edges.                      |
| `engine.input-update`   | pre-frame | with bindings            | Publishes the frame-scoped action view before state hooks. |
| `engine.signal-begin-frame` | pre-frame | always | Opens the frame signal window before user systems. |
| `engine.signal-end-frame` | post-frame | always | Closes the frame signal window after user systems. |
| `engine.signal-end-tick` | post-fixed | always | Rotates tick signals after user systems. |
| `engine.audio`          | frame | the runner               | Reconciles audio source state with runner players.      |
| `scene-startup`         | startup | `WithScenes`           | Applies requests queued before startup.                 |
| `scene`                 | frame | `WithScenes`             | Applies queued scene requests before normally appended frame systems. |

The timer, collision, and animation systems are always registered: their work is declared by component presence, so the component is the registration and an unmatched system costs an empty query per tick. Scenes and state machines are instance-driven and stay opt-in via `WithScenes` and `WithState`. Public built-in systems export their registration names from the package that owns them - `core.PrevTransformSystemName`, `input.FrameSystemName`, `input.TickSystemName`, `animation.SystemName`, `collision.SystemName`, `gametime.TimerSystemName`, `ebitrun.AudioSystemName` - so call sites do not need to hand-type those names. Internal schedule hooks may use private names.

## System groups

The `castrum.InGroup` option assigns a system to a named group. Groups start
enabled, regardless of whether scenes are enabled. `Game.SetGroupEnabled`
changes a registered group's state across phases and applies it to systems
registered in the group later. It affects systems that have not yet run in
the current phase. An empty or unregistered group returns an error.

Scene loading, switching, updating, and removal do not change group states.
Use explicit group control when systems should pause together, or let systems
query for the components they need.

## Input timing

The runner publishes a raw snapshot before `Advance` runs. The frame input system updates the frame action view once per display frame. The fixed input system publishes the press and release edges accumulated since the previous tick; systems that run after it in that tick see the same tick view. Pre-fixed systems and systems inserted before the fixed input system run before this publication.

Read frame-level state in frame systems and apply deterministic gameplay in fixed systems. The [input guide](../guides/input.md) shows the public action APIs.

## Signal rotation

The signal hub rotates with the loop: `BeginFrame` runs before pre-frame systems, `EndFrame` after post-frame systems, and `EndTick` after post-fixed systems. A signal emitted during a fixed tick is visible to tick-phase systems that run after its emitter in that tick, and once to pre-frame, frame, and post-frame systems in the next display frame; then it is dropped. Frame signals live only through that display frame. The [signals guide](../guides/signals.md) covers the authoring side: emitting, readers, and consumer systems.

## Why the capture order matters

The previous-transform system must run before gameplay movement. It stores the prior position; gameplay then writes the current position; rendering interpolates between the pair using alpha. Registering a competing capture after movement would make both values equal and remove visible interpolation.

The [coordinate system](coordinate-system.md) explains the projection side of alpha, while [performance](../guides/performance.md) covers the cost and tuning consequences of the loop.
