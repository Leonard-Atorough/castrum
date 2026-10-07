# Guides

The guides are Castrum's manual: each chapter explains one engine feature in depth, with code you can adapt and links to the source examples. Read the getting-started chapters first if the world, systems, and fixed phases are new to you.

## Foundation

- [The ECS in depth](ecs.md) - entities, components, systems, queries, resources, and validation.
- [Assets](assets.md) - filesystem roots, loading, atlases, and setup-time failures.
- [Conventions and best practices](conventions.md) - setup, zero values, validation, errors, and state-driven APIs.

## Presentation

- [Rendering](rendering.md) - sprites, shapes, cameras, overlays, ordering, and culling.
- [Animation](animation.md) - atlas clips, playback state, looping, and control.

## Interaction and audio

- [Input](input.md) - action bindings, raw snapshots, and frame versus tick reads.
- [Audio](audio.md) - one-shots, music, streaming, mixer buses, and playback state.

## Shipping and performance

- [Performance](performance.md) - frame budgets, query costs, rendering costs, and profiling decisions.
- [Publishing your game](publishing-your-game.md) - embedded assets and release builds.

## Problem solving

- [Troubleshooting](troubleshooting.md) - common setup, asset, input, rendering, and loop failures.

## Deeper context

The [reference tier](../reference/core-principles.md) explains why the engine is shaped this way and documents details most games do not need day to day.

The [API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) is the authoritative list of exported types and methods.
