# castrum

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance.

You declare what your game contains - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it every frame. There is no draw loop to write and no play call to make.

Castrum uses [Ebitengine](https://ebitengine.org) as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

## Why Castrum

| Area       | What works                                                                                                                                                                                                     |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Rendering  | Sprites and shape primitives from one pipeline: standalone textures and grid atlases, a camera, interpolated motion between ticks, viewport culling, layer/sort-order/Y ordering, overlays on top of the world |
| Simulation | A fixed-timestep loop (60 ticks per second by default) with slow-frame guards, startup/frame/fixed phases                                                                                                      |
| Input      | Named bindings across keyboard, mouse, and gamepad - and raw per-frame polling when a game wants a key, not an action                                                                                          |
| Animation  | Atlas-frame clips with looping, pause, and playback rate, advanced by the engine                                                                                                                               |
| Audio      | Sound effects (decoded and shared) and music (streamed from the file), a master and two group volume buses, per-play pause behavior                                                                            |
| Structure  | Entities, components, queries over the world, typed resources                                                                                                                                                  |

Planned, in rough order: timers, text, and UI; screen state (fullscreen, window mutations); publishing and packaging; full transform interpolation. Each lands with its guide chapter - the docs are the arrival record.

## Start here

- [Documentation](docs/getting-started/intro.md) - choose the introduction, concepts, first-game tutorial, or the full guides from there.
- [Examples](examples) - run complete programs covering rendering, shapes, input, animation, and audio.
- [API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) - browse exported types and methods.

## Status

Castrum is pre-1.0 and moving: the API is unstable, the docs evolve with it, and the examples are kept current. Treat every release as a breaking change until the version says otherwise. See the [changelog](CHANGELOG.md).

## The name

A _castrum_ (Latin) was a Roman fortified camp: a garrison built to a standard layout from local materials, quickly, wherever the legions needed to hold ground. The name fits an engine that aims to be the standard-built base your game stands on - small, planned, and hard to knock over.
