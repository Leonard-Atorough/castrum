# Scenes

This guide is an outline for a future walkthrough of Castrum's scene system.
For a runnable example, see [Salvage Bay](../../examples/salvage-bay/main.go).

## Mental model

- Explain blueprints as recipes and instances as managed scene lifetimes.
- Contrast scene-owned entities with entities created directly in the world.
- Link to [The ECS in depth](ecs.md) for entities, systems, and resources.

## Enable scenes and create a blueprint

- Show `castrum.WithScenes()` and retrieving `*scene.Director` from the world.
- Define a blueprint with named entity specifications and components.
- Show a system querying components created by a scene.

## Load and switch scenes

- Queue a scene during setup and explain startup and frame processing.
- Contrast `Load`, `Switch`, and `Remove`; show how to retain an instance ID.
- Explain pending instances, structural snapshots, and shared references.

## Compose scenes

- Explain child entity ownership and nested scene instances.
- Explain that system scheduling is independent of scene lifetimes.

## Inspect and update instances

- Show `Get` and `ByName`, then replace a live instance with `Update`.
- Explain what a detached instance view does and does not isolate.

## Hooks and error handling

- Describe Enter and Exit ordering and the effects of hook errors.
- Distinguish request validation errors from errors during application.
- Explain that failed requests are consumed and unattempted requests stay queued.

## Assets and current limitations

- Explain that `Blueprint.Assets` does not load assets; link to the
  [assets guide](assets.md) for asset loading.
- Document any remaining limitations when the guide is written.
