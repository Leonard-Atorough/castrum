# Features

Castrum is a 2D game engine for Go. You compose entities from data, update that data in systems, and let the engine handle simulation, rendering, input, and audio. This page is a map of the current feature set; follow the linked guides for setup details and examples.

## Build a game world

- **Entities and components** compose game state. Systems update it, queries select entities, and typed resources share world-level services. See [key concepts](concepts.md) and [the ECS guide](../guides/ecs.md).
- **Simulation phases and system groups** provide startup, per-frame, and fixed-tick work. The fixed timestep defaults to 60 ticks per second; named groups can be enabled or disabled across phases. See [key concepts](concepts.md) and [the scheduler reference](../reference/the-scheduler.md).
- **Scenes** define entity blueprints and manage scene-owned entity lifetimes in the shared world. Scene asset IDs are metadata; the scene system does not load them. See the [scene guide](../guides/scenes.md).

## Add gameplay behavior

- **Input** supports named actions bound to keyboard, mouse, and gamepad controls, as well as raw input snapshots. See the [input guide](../guides/input.md).
- **Signals** provide typed event streams with separate tick and frame delivery windows. Timers, animation, collision, and audio emit engine signals. See the [signals guide](../guides/signals.md).
- **Timers** provide one-shot and repeating countdowns as entity components. Read elapsed time and completion state from the component. See the [timer guide](../guides/timer.md).
- **Collision** detects box and circle overlaps using layers and masks, then records contacts as component state. It detects overlaps only; it does not resolve physics. See the [collision guide](../guides/collision.md).
- **Animation** advances atlas-frame clips with looping, pause, and playback-rate controls. See the [animation guide](../guides/animation.md).

Timers, collision, and animation are optional systems; enable them individually or use `castrum.WithDefaultSystems()`. The [cheat sheet](cheat-sheet.md) lists the available setup options.

## Render the game

- **Sprites and drawables** support standalone textures, atlas regions, rectangles, circles, lines, and text from loaded fonts. See the [rendering guide](../guides/rendering.md) and [assets guide](../guides/assets.md).
- **Cameras and ordering** support interpolated position, rotation, uniform zoom, anchors, world bounds, viewport culling, draw layers and sort order, Y ordering, overlays, and the published resolved view for picking. See the [rendering guide](../guides/rendering.md).

## Add sound

- **Audio sources** support decoded sound effects, streamed music, per-play pause behavior, and a mixer with a master bus and two group buses. See the [audio guide](../guides/audio.md).

## Debug and develop

- **Debugging** provides an in-memory panel for status lines and recent messages, plus an optional runner overlay for frame and tick rates. See the [debugging guide](../guides/debug.md).
- **Starter project** provides a copyable, asset-free Go module with a runnable game and a recommended `cmd/` and `internal/` layout. See the [starter README](../../templates/starter/README.md).

## Explore working examples

The [examples](../../examples) directory contains runnable programs covering rendering, shapes, input, animation, audio, collision, timers, signals, and scenes.
