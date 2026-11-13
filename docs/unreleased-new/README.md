# Castrum manual

Castrum is a 2D game engine for Go. This manual is for the work of making a game: getting something on screen, giving it behavior, and bringing it together into a project you can share.

These pages describe the development checkout, not necessarily the latest released module. If you're using a release, choose its matching [documentation and examples](../README.md) instead.

This manual is being rebuilt. Until a topic has a new guide, the links below take you to the **existing unreleased documentation**. Those pages haven't yet been reorganized or edited to match the new manual.

## Start here

New to Castrum? Start with the existing [introduction](../unreleased/getting-started/intro.md), then follow [Your first game](../unreleased/getting-started/your-first-game.md) to make a moving, mouse-aiming tank. The [starter project](../../templates/starter/README.md) gives you a project layout to build on.

If you'd like to understand the pieces as you go, the existing [key concepts](../unreleased/getting-started/concepts.md) introduce the world, entities, components, and systems.

## Build your game

### Put things on screen

- [Draw and position an image](guides/rendering/images.md), then use [a camera](guides/rendering/cameras.md) to choose what the player sees.
- [Draw and animate a spritesheet](guides/rendering/spritesheets.md) to give a character or object changing poses.
- Explore the existing [rendering guide](../unreleased/guides/rendering.md) for shapes, text, ordering, and overlays.

### Bring assets into your game

- [Load game assets](guides/assets/loading.md) and choose where your game reads files from.
- [Diagnose an asset or rendering failure](guides/assets/troubleshooting.md) when a file or drawable doesn't work as expected.
- Consult the existing [assets guide](../unreleased/guides/assets.md) for codecs, saving, and cache policies.

### Create game behavior

- Use the existing [ECS guide](../unreleased/guides/ecs.md) to work with entities, components, queries, and resources; see [scheduling systems](../unreleased/guides/system-scheduling.md) to decide when your systems run.
- [Give the player controls](guides/interaction/controls.md), then [detect and respond to collisions](guides/interaction/collisions.md).
- Organize game modes and reusable world content with [state](../unreleased/guides/state.md) and [scenes](../unreleased/guides/scenes.md).
- Use [signals](../unreleased/guides/signals.md) to communicate between systems, [timers](../unreleased/guides/timer.md) for countdowns, and [time control](../unreleased/guides/time-control.md) to pause or scale time.
- Add transitions with [tweening](../unreleased/guides/tweening.md), sound and music with [audio](../unreleased/guides/audio.md), and player preferences with [settings](../unreleased/guides/settings.md).

### Finish and share

The existing guides cover [debugging](../unreleased/guides/debug.md), [performance](../unreleased/guides/performance.md), and [publishing your game](../unreleased/guides/publishing-your-game.md). There's also a general [troubleshooting guide](../unreleased/guides/troubleshooting.md).

## Understand Castrum

- [2D space and the camera](concepts/2d-space.md) explains how world, camera, and logical-screen coordinates fit together.
- [How assets reach your game](concepts/assets-and-loading.md) explains filesystem roots, sources, loaders, and caches.
- The existing [core principles](../unreleased/reference/core-principles.md) provide more background on the engine's design.

## Examples and API

The [examples](../../examples) are complete programs you can run and adapt. They use the development checkout; follow [Develop examples against the checkout](../../CONTRIBUTING.md#develop-examples-against-the-checkout) to set them up.

For exported types, functions, and their contracts, use the [Go API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum). Make sure the version selected there matches the engine you are using.
