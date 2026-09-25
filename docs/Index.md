# Castrum Documentation

This is the index file for the castrum game engine markdown docs. These docs wil be used to provide constantly updating information on using the game engine while its in an unstable phase and will form the basis of the online documentation once the engine reaches a more stable release.

This file is the starting point for the castrum documentation, providing an overview and links to more detailed sections on using the game engine.

## Overview

Castrum is a game engine designed to provide a flexible and efficient framework for developing games. It focuses on a clear separation between configuration, simulation, and rendering, allowing developers to fine-tune their game's behavior and performance. The engine is currently in an unstable phase, and the documentation will evolve alongside it.

Castrum uses Ebitengine as its underlying rendering and window management library, leveraging its capabilities to handle graphics, input, and other low-level tasks efficiently. To facilitate this, castrum wraps Ebitengine functionality within its own abstractions, providing a more structured and game-focused interface for developers.

## Guides

Start here if you are new to Castrum. Each guide builds on the previous one.

1. [Getting started](guide/getting-started.md) - install Castrum, open your first window, and put a sprite on screen.
2. [The loop](guide/the-loop.md) - the fixed-timestep core: phases, DeltaTime, and the spiral-of-death guards.
3. [Systems](guide/systems.md) - writing game logic and organizing it into schedules.
4. [Resources](guide/resources.md) - typed, shared state on the world.
5. [Runners](guide/runner.md) - the runner contract, windows, overlays, and the engine renderer.

## Reference

Quick lookup for the public API.

- [Options](reference/options.md) - every `With*` option, its default, and its panic conditions.
- [Core](reference/core.md) - phases, `Context` fields, and the `System` interface.
- [World](reference/world.md) - resource registration, resolution, and error conditions.