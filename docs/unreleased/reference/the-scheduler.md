# The scheduler and the fixed loop

The runner supplies elapsed platform time; `Game.Advance` turns it into frame work and fixed simulation ticks. The fixed loop gives simulation a stable `DeltaTime` while rendering can interpolate between completed ticks.

## Startup and a display frame

`Game.Startup` resolves eager resources, then runs the pre-startup, startup, and post-startup schedules once. This lets initial state hooks queue scene work before the startup scene system applies requests. Calling `Startup` a second time returns an error.

A runner calls `Advance(elapsed)` once per display frame. `Advance` clamps the elapsed duration to `MaxFrameTime`, increments the frame counter, publishes that duration as frame `Context.DeltaTime`, then runs pre-frame, frame, and post-frame systems in order. Unless paused, `Advance` adds the clamped duration scaled by simulation time scale to the fixed accumulator after frame work. The default runner draws after the update, not in post-frame.

A frame can run zero fixed ticks. A slow frame runs as many fixed ticks as the accumulator can pay for, up to `MaxTicksPerFrame`. Each fixed tick increments `Context.Tick`, sets `Context.DeltaTime` to the fixed interval, runs pre-fixed, fixed, and post-fixed systems in order, and subtracts one interval from the accumulator.

The defaults are `FixedTPS=60`, `MaxFrameTime=250ms`, and `MaxTicksPerFrame=5`. When the tick cap is reached, the remaining backlog is dropped rather than carried into every later frame. These limits protect stability; they do not make an expensive system cheaper.

## Alpha

`Context.Alpha` is the remainder of the accumulator divided by the fixed interval, in `[0, 1)`. A renderer uses it to display a state between the previous and current simulation states. `render.Collector` interpolates transform position, rotation, and scale; a custom overlay that draws world-space state must apply the value itself if it needs the same effect.

When the catch-up cap drops the backlog, the accumulator is reset and alpha becomes zero. The renderer therefore never interpolates across time that the simulation intentionally discarded.

## Schedule order

Each phase runs engine-first systems, user systems, and then engine-last systems. Arrows in the table give order within a bucket; a dash means the engine has no system there. Signal hooks additionally occupy the outermost opening or closing position, outside those buckets.

| Phase | Engine-first | User systems | Engine-last |
| --- | --- | --- | --- |
| `PhasePreStartup` | Initial state processing (`state`, with state) | Pre-startup setup | — |
| `PhaseStartup` | Apply startup scene requests (`scene-startup`, with scenes) | Startup setup | — |
| `PhasePostStartup` | — | Post-startup work | — |
| `PhasePreFrame` | `BeginFrame` → `engine.input-update` (with bindings) → `state` (with state) | Pre-frame work | — |
| `PhaseFrame` | Snapshot audio backend (`engine.audio-prepare`) → apply scene requests (`scene`, with scenes) | Frame work | `engine.audio` |
| `PhasePostFrame` | — | Post-frame work, including audio-signal consumers | `EndFrame` |
| `PhasePreFixed` | `engine.prev-transform` → `engine.input-tick` (with bindings) | Tick preparation | — |
| `PhaseFixed` | `engine.timer` → `engine.animation` → `engine.collision` | Gameplay and tick-signal consumers | — |
| `PhasePostFixed` | — | Post-tick work | `EndTick` |

Timer, animation, collision, capture, audio, and signal hooks are always registered. Scenes, state, and input require their respective options. Audio reconciliation runs only when a runner attaches a controller to `*audio.Backend`; a headless game needs no controller. An attachment during `PhaseFrame` takes effect on the next frame pass, while an attachment in an earlier phase can take effect in the current frame.

User systems normally run in registration order. `castrum.Before` and `castrum.After` insert a new user system relative to an already registered user system in the same phase; engine systems are not valid targets. Names are unique across all positions in a phase, including engine systems. An insertion into an active phase runs on its next pass; an insertion into a later phase can run in the current frame or tick. Startup-phase registrations are rejected once `Startup` begins.

A system error stops the current schedule and is returned through `Startup` or `Advance`, and for the default runner through `g.Run`. Later systems, including signal-closing hooks, do not run after an error.

The timer, collision, and animation systems do work declared by component presence; an unmatched system costs an empty query per tick. Their registration names and other public built-in names are exported by their owning packages, but these names cannot be used as relative insertion targets.

## Run the relative-ordering example

Run `go run .` from [examples/scheduling](../../../examples/scheduling) using the development workspace. This headless command registers apply/report first, then inserts sample before apply and audit after apply. One frame prints sample 21, apply 42, audit 42, report 42.

The targets already exist in the same user phase. The example uses neither engine-system names nor cross-phase ordering and returns registration or execution errors to the caller. For clock controls, use [Time control](../../../examples/time-control); for coordinated system groups, use [Signal Salvage](../../../examples/signal-salvage).

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

The runner publishes a raw snapshot before `Advance` runs. Frame input updates the frame action view once per display frame. Tick input publishes press and release edges accumulated since the previous tick before user pre-fixed systems, so pre-fixed and fixed consumers see the same tick view. Catch-up ticks after the first do not repeat those edges. A successfully applied, non-empty scene request batch flushes input edges before tick publication, preventing a transition's input from leaking into the new scene.

Read frame-level state in frame systems and apply deterministic gameplay in fixed systems. The [input guide](../guides/input.md) shows the public action APIs.

## Signal rotation

The signal hub rotates with the loop: `BeginFrame` runs before pre-frame systems, `EndFrame` after post-frame systems, and `EndTick` after post-fixed systems. A signal emitted during a fixed tick is visible to tick-phase systems that run after its emitter in that tick, and once to pre-frame, frame, and post-frame systems in the next display frame; then it is dropped. Frame signals live only through that display frame and expire before fixed ticks. Audio reconciliation emits `audio.Finished` after user frame systems, so consume it in post-frame, not in fixed or the next frame. The [signals guide](../guides/signals.md) covers the authoring side: emitting, readers, and consumer systems.

## Why the capture order matters

Previous-transform capture runs before pre-fixed movement. It stores the prior transform; gameplay then writes the current transform; rendering interpolates between the pair using alpha. Pre-fixed movement affects collision detection in the same tick, while fixed movement is detected on the next tick.

The [coordinate system](coordinate-system.md) explains the projection side of alpha, while [performance](../guides/performance.md) covers the cost and tuning consequences of the loop.
