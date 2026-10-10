# Introduction

## What Castrum is

Castrum is a 2D game engine for Go, built on [Ebitengine](https://ebitengine.org). You describe what exists in your game - an entity with a position and a picture, an entity with a looping music file - and the engine renders it, advances it, and plays it. There is no draw loop to write and no `play` call to make.

The engine is built around an entity-component system. Entities are identities, components hold data, and systems update that data over time. The [key concepts](concepts.md) chapter introduces that model; the [first-game tutorial](your-first-game.md) puts it to work and teaches you how to use it.

## What writing a game feels like

- **Describe state.** Create entities and attach components such as `Transform`, `Sprite`, and `audio.Source`.
- **Write rules.** Add systems that read input and update components.
- **Use fixed simulation.** Gameplay systems run at a fixed tick rate, so the same inputs produce the same simulation on different displays.
- **Let the engine handle the platform.** The runner owns the window, drawing surface, input snapshot, and audio provider. Your game code works with the same world and systems above it.

When a component contains invalid data, Castrum returns an error when it enters the world. Errors are values throughout the API; the tutorial returns them to `main`, where the example panics if setup cannot continue.

For a structured summary of what the engine supports today, see the [feature overview](features.md).

## Start building

- [Your first game](your-first-game.md) - build a moving, mouse-aiming tank with music and sound effects.
- [Key concepts](concepts.md) - understand entities, components, systems, queries, the fixed timestep, and resources.
- [Cheat sheet](cheat-sheet.md) - recall the core construction and API patterns.
- [The examples](../../examples) - run complete programs covering rendering, shapes, input, animation, and audio.
