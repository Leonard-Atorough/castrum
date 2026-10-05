# castrum

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance.

You declare what your game contains - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it every frame. There is no draw loop to write and no play call to make.

Castrum uses [Ebitengine](https://ebitengine.org) as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

## Quick start

```sh
mkdir mygame && cd mygame
go mod init mygame
go get github.com/Leonard-Atorough/castrum
```

Then read [your first game](docs/getting-started/your-first-game.md) - a step-by-step tutorial from an empty directory to a moving player with music and sound effects.

To see the engine working right away, run an example from a clone of this repository:

```sh
go run ./examples/wander
```

Each example's doc header says what it shows; the five cover rendering, shapes, input, animation, and audio.

## What works today

| Area | What works |
|---|---|
| Rendering | Sprites and shape primitives from one pipeline: standalone textures and grid atlases, a camera, interpolated motion between ticks, viewport culling, layer/sort-order/Y ordering, overlays on top of the world |
| Simulation | A fixed-timestep loop (60 ticks per second by default) with slow-frame guards, startup/frame/fixed phases |
| Input | Named bindings across keyboard, mouse, and gamepad - and raw per-frame polling when a game wants a key, not an action |
| Animation | Atlas-frame clips with looping, pause, and playback rate, advanced by the engine |
| Audio | Sound effects (decoded and shared) and music (streamed from the file), a master and two group volume buses, per-play pause behavior |
| Structure | Entities, components, queries over the world, typed resources |

Planned, in rough order: timers, text, and UI; screen state (fullscreen, window mutations); publishing and packaging; full transform interpolation. Each lands with its guide chapter - the docs are the arrival record.

## Documentation

Three tiers, by what you came for:

**[Getting started](docs/getting-started/intro.md)** - learn.
- [Introduction](docs/getting-started/intro.md) - what castrum is and where to read next.
- [Key concepts](docs/getting-started/concepts.md) - entities and components, systems and phases, the fixed timestep, and resources.
- [Your first game](docs/getting-started/your-first-game.md) - the step-by-step tutorial.
- [Cheat sheet](docs/getting-started/cheat-sheet.md) - recall, once the basics are read.

**[Guides](docs/guides/rendering.md)** - build, one feature per guide.
- [Rendering](docs/guides/rendering.md), [Input](docs/guides/input.md), [Animation](docs/guides/animation.md), [Audio](docs/guides/audio.md)
- [Conventions](docs/guides/conventions.md), [Performance](docs/guides/performance.md), [Publishing your game](docs/guides/publishing-your-game.md)

**Reference** - dig, for the curious. Most games never need this tier.
- [Core principles](docs/reference/core-principles.md), [Engine design](docs/reference/engine-design.md), [The runner separation](docs/reference/runner-separation.md)

The full API reference lives on [pkg.go.dev](https://pkg.go.dev/github.com/Leonard-Atorough/castrum).

## Status

Castrum is pre-1.0 and moving: the API is unstable, the docs evolve with it, and the examples are kept current. Treat every release as a breaking change until the version says otherwise. See the [changelog](CHANGELOG.md).

## The name

A *castrum* (Latin) was a Roman fortified camp: a garrison built to a standard layout from local materials, quickly, wherever the legions needed to hold ground. The name fits an engine that aims to be the standard-built base your game stands on - small, planned, and hard to knock over.
