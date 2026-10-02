# Castrum Documentation

This is the index file for the castrum game engine markdown docs. These docs will be used to provide constantly updating information on using the game engine while its in an unstable phase and will form the basis of the online documentation once the engine reaches a more stable release.

This file is the starting point for the castrum documentation, providing an overview and links to more detailed sections on using the game engine.

## Overview

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance. The engine is currently in an unstable phase, and the documentation will evolve alongside it.

Castrum uses Ebitengine as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

## Getting started

New to Castrum? This tier takes you from install to a running game.

- **Introduction** (planned) - install Castrum, open a window, and put a sprite on screen. The engine renders the world itself; there is no drawing code to write at this stage.
- **Key concepts** (planned) - entities and components, systems and phases, resources, and the fixed timestep.
- **Writing castrum code** (planned) - the authoring patterns: options, spawning entities, and the two creation surfaces.
- **Cheat sheet** (planned) - recommended patterns, for recall once the basics are read.

## Guides

One actionable engine feature per guide.

- **Rendering** (planned) - sprites and atlases, shapes, the camera, and overlays (the one place game code draws).
- **Input** (planned) - raw snapshot polling and bindings-driven actions.
- **Animation** (planned) - frame animation over atlas regions.
- **Audio** (planned) - playing sounds and music, volume buses, and pause behavior.
- **Conventions** (planned) - the engine's rules: zero values, errors, and validation.
- **Performance** (planned) - measured costs and recommendations.
- **Publishing your game** (planned) - embedding assets and building releases.

## Reference

For the curious: how the engine is shaped and why. Most games never need this tier.

- **Core principles** (planned) - backend-free core, the zero-value culture, errors everywhere.
- **Engine design** (planned) - the world, entities, and schedules.
- **The runner separation** (planned) - the engine/backend boundary, and what a runner actually does: the platform loop, the window, input, and the draw surface.

The full API reference lives on [pkg.go.dev](https://pkg.go.dev/github.com/Leonard-Atorough/castrum).

## Examples

[The examples](examples.md) are runnable from the repository root - one paragraph each:

```sh
go run ./examples/wander
```
