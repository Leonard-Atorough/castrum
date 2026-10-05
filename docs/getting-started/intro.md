# Introduction

Castrum is a 2D game engine for Go, built on [Ebitengine](https://ebitengine.org). You declare what your game contains - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it every frame. There is no draw loop to write and no play call to make. The model underneath is an entity-component system, small enough to read in one sitting: [key concepts](concepts.md).

This page is the engine's charter: what castrum is aiming at, what is planned, what it deliberately does not do, and the principles the API is held to. No code here - the [tutorial](your-first-game.md) starts that.

## Design goals

- **Declare, don't command.** A game writes state; the engine makes the screen and the speakers match it next tick. No imperative draw, play, or advance calls cross the engine boundary - those decisions belong to the engine, on the engine's schedule.
- **Deterministic simulation.** The game logic runs on a fixed timestep, the same steps on a 144 Hz monitor and a 30 fps laptop. Same inputs, same result.
- **Small surface, strong defaults.** Components are valid in their zero value wherever possible, so spawning is struct literals, not configuration. Every public API earns its place; nothing is added on anticipation.
- **A backend-free core.** The engine (`castrum`, `core`) compiles without knowing what a window is. The platform - window, draw surface, input, audio codecs - lives behind a runner that drives the game. Ebitengine is the current runner's backend, chosen for development speed, not as a permanent commitment.
- **Fail fast at the boundary.** Bad state is rejected at spawn, at registration, at construction - with an error naming the rule that was broken - instead of misbehaving mid-game.

## Planned features

Closest first: four features the previous prototype already proved out, rebuilt on this engine's model with significant refinement. In rough order of arrival:

- **Timers** - a Timer component the engine advances. Completion is state to read - like a finished audio play - not an event to catch: the prototype fired an event bus; the rebuild makes it a field.
- **Events** - typed event dispatch, redesigned for the single-threaded contract. The prototype's bus carried a mutex and reflection; the rebuild drops both.
- **Entity hierarchy** - parent/child entities: attach, reparent, and tear down with the parent. The prototype's version was ID bookkeeping; the rebuild makes the relationship carry weight.
- **Scenes** - groups of entities with load and unload lifecycle, for moving between levels and menus.

Then, further out:

- **Text and UI** - the pieces a first game grows into after timers.
- **Screen state** - fullscreen, window mutations, monitor handling.
- **Publishing and packaging** - the shipping story beyond a plain Go binary.
- **Full transform interpolation** - rotation and scale, joining position today.

Each of these lands with its guide chapter; the docs are the arrival record. The feature list for what works _today_ is on the [README](../../README.md).

## Intentional limits

What castrum does not do today. Some are deliberate architecture, some are simply not built yet - the planned list above is the not-yet part.

- **Single-threaded.** Everything runs on the loop's thread. There are no goroutines in game code against the world or its resources, by design: the engine protects its own state, and one thread is what makes that guarantee cheap.
- **2D only.** Positions, rotations, and scales in a plane.
- **One runner.** `ebitrun`, backed by Ebitengine. Headless and alternative runners do not exist yet - the boundary permits them, but nothing has earned them.
- **Position interpolation only.** Motion between ticks is interpolated for position; rotation and scale snap. Full interpolation is planned.
- **Audio formats.** MP3, OGG Vorbis, and WAV (8- and 16-bit PCM).
- **Desktop platforms.** The engine targets the platforms Ebitengine does; other than the common desktop targets, nothing is tested here.
- **Pre-1.0.** The API is unstable and every release should be treated as a breaking change until the version says otherwise.

## Principles

The commitments every chapter of these docs - and the engine itself - is held to:

- **State, not commands.** The engine's systems reconcile the world to what components declare. Playback control, pausing, retargeting: all field writes.
- **The zero value is a contract.** A component in its zero value is valid and sensible, or its spawn fails loudly with a message stating the range. There is no third state where a component sits in the world misbehaving.
- **Errors are values.** Engine APIs return errors; nothing panics across the API surface. The one accepted panic is `main`, because a game that fails to construct has nothing to recover into.
- **Earned surface.** A feature arrives when a second real consumer exists, an abstraction when a second implementation does. The API is small on purpose, and the internal design notes are the paper trail.

The [conventions guide](../guides/conventions.md) carries these as day-to-day rules; the [reference tier](../reference/core-principles.md) carries the design reasoning behind them.

## Read next

- [Key concepts](concepts.md) - entities and components, systems and phases, the fixed timestep, and resources: the four ideas every castrum game is built from.
- [Your first game](your-first-game.md) - hands-on, step by step: from an empty directory to a moving player with music and sound effects.
- [The examples](../../examples) - five runnable programs; each one's doc header says what it shows.

## Closing thoughts

> We hope you enjoy building with castrum as much as we enjoyed creating it. This started as a single developer's passion project and can only grow with the community's support.
