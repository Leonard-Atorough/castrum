# castrum

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance.

You declare what your game contains - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it every frame. There is no draw loop to write and no play call to make.

Castrum uses [Ebitengine](https://ebitengine.org) as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

## Why Castrum

| Area       | What works                                                                                                                                                                                                     |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Rendering  | Sprites, shapes, and text from one pipeline: standalone textures and grid atlases, fonts, a camera, interpolated motion between ticks, viewport culling, layer/sort-order/Y ordering, overlays on top of the world |
| Simulation | A fixed-timestep loop (60 ticks per second by default) with slow-frame guards, startup/frame/fixed phases                                                                                                      |
| Input      | Named bindings across keyboard, mouse, and gamepad - and raw per-frame polling when a game wants a key, not an action                                                                                          |
| Animation  | Atlas-frame clips with looping, pause, and playback rate, advanced by the engine                                                                                                                               |
| Collision  | Overlap detection for boxes and circles over layer bitmasks, with contacts as component state - detection only, no solver                                                                                      |
| Timers     | One-shot and repeating timers as components, with tick-stamped completion you read as state                                                                                                                    |
| Audio      | Sound effects (decoded and shared) and music (streamed from the file), a master and two group volume buses, per-play pause behavior                                                                            |
| Structure  | Entities, components, queries over the world, typed resources                                                                                                                                                  |

See the [roadmap](roadmap/README.md) for the proposed feature sets, known gaps, and longer-term directions. Release targets are planning proposals rather than dated commitments; the guides and API reference describe what the current version actually supports.

## Start here

- [Documentation](docs/getting-started/intro.md) - choose the introduction, concepts, first-game tutorial, or the full guides from there.
- [Examples](examples) - run complete programs covering rendering, shapes, input, animation, and audio.
- [API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) - browse exported types and methods.

## Status

Castrum is pre-1.0 and moving: the API is unstable, the docs evolve with it, and the examples are kept current. Treat every release as a breaking change until the version says otherwise. See the [changelog](CHANGELOG.md).

## The name

A _castrum_ (Latin) was a Roman fortified camp: a garrison built to a standard layout from local materials, quickly, wherever the legions needed to hold ground. The name fits an engine that aims to be the standard-built base your game stands on - small, planned, and hard to knock over.

## How this is made

Castrum is developed using an AI-assisted workflow: a person owns the design, the direction, and every line that lands; AI helps implement, test, and document it; the person reviews the whole diff, and nothing merges without passing gates nobody can talk past - vet, tests, coverage, and benchmarks. The contribution policy in [CONTRIBUTING.md](CONTRIBUTING.md) draws the same line for others: AI assistance is welcome, delegation is not.

We declare this because it is true, and because we would rather read a one-line "built with AI assistance, reviewed by me" than guess. If castrum is part of something you make, transparency in your own style is encouraged - say what the machine did and what you did. The tooling has changed how software gets written and honesty and integrity is how we as a community build trust in each other, and in the projects we create.
