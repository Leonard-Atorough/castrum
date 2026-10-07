# Troubleshooting

Start with the boundary named by the error and keep the failure at that boundary while diagnosing it. Castrum reports failures during construction, setup, startup, frame advancement, or rendering; the time of failure is often the most useful clue.

- [Conventions and best practices](conventions.md) - patterns that prevent common failures.

## First, classify the failure

Use this order:

1. **Build failure:** imports, package names, or Go version mismatch.
2. **Construction failure:** `castrum.New` rejected options or bindings.
3. **Setup failure:** an entity, resource, asset, clip, or runner registration returned an error.
4. **Startup failure:** eager resource resolution or a startup system failed.
5. **Loop failure:** a frame, fixed, or draw system returned an error.
6. **Late resolution failure:** a lazy texture, atlas region, or streamed asset failed on first use.

Preserve the original error when adding context:

```go
player, err := g.World().NewEntity(component)
if err != nil {
	return fmt.Errorf("spawning player: %w", err)
}
```

The operation, path, type, or system name in the wrapped error usually points to the next check.

## The game does not build

Check imports and the module before investigating runtime behavior:

```sh
go mod tidy

```

Use package paths from the current examples. If an example refers to an API that is not in the checkout, compare the matching example and the [API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) before adapting it. Check the repository's Go version in `go.mod` and run the command from the module root.

## Construction or setup fails

`castrum.New` applies defaults and validates `FixedTPS`, `MaxFrameTime`, and `MaxTicksPerFrame`. It also validates input bindings when `WithBindings` is present. An invalid option returns an error and no half-built game.

After construction, check each setup boundary independently:

- `World.NewEntity` rejects nil or invalid components.
- `Game.AddSystem` rejects an empty system name.
- asset and atlas registration can fail while reading or decoding files.
- resource registration rejects duplicate types.
- clip registration rejects duplicate names.
- `ebitrun.New` can fail while configuring runner providers.

Do not ignore a setup error and continue to the runner. Keep the operation context so the failing boundary stays visible.

## Startup fails before the window loop

The runner calls `Game.Startup` once before entering its platform loop. `Startup` resolves all resources registered with `ProvideEager`, then runs startup systems. A constructor error or startup-system error stops startup; a second call to `Startup` returns an error.

Use `ProvideEager` when a dependency should fail before startup. Use ordinary `Provide` when lazy resolution is intentional, and remember that its constructor runs on the first `Resource` request rather than at registration.

For a headless test or custom runner, call `Startup` explicitly and inspect its error before calling `Advance`.

## An asset cannot be found

Asset paths resolve through the filesystem passed to `castrum.WithFilesystem`. With the default nil filesystem, paths resolve from the process working directory, not from the source file's directory. With `embed.FS`, the path must be present under the embedded tree.

Check the following:

1. Confirm the path's spelling, case, and extension.
2. Confirm the process working directory, or pass an explicit `fs.FS`.
3. Confirm the file is included by `embed.FS`.
4. Preload required data during setup to move filesystem and decode errors earlier.
5. Check whether the consumer is lazy. A `TextureSource` can fail when the collector first renders it, and an unknown atlas or region can fail on the first rendered frame.

See [Assets](assets.md) for roots, cache behavior, and loading timing, and [Publishing your game](publishing-your-game.md) for embedded files.

## Audio fails to load or play

`ebitrun.New` registers the WAV, MP3, and OGG audio decoders and providers. Construct the runner before explicitly preloading `asset.AudioData`. If audio is loaded by an `audio.Source`, the failure may remain lazy until the provider creates the play.

Check the source ID and loading mode. A source needs a non-empty audio path and a valid positive volume; `audio.NewSource` supplies the normal unity-volume default. Use `audio.LoadEager` for short repeated effects and `audio.LoadStream` for long tracks. Streaming needs a seekable filesystem source.

See [Audio](audio.md) for source fields, playback state, buses, and provider timing.

## Input does not respond

First decide which input surface the system uses:

- `ctx.Input` is the raw per-frame snapshot. It does not require `WithBindings`, but it is nil when no runner has published input.
- `ctx.Actions` is the named `input.ActionMap`. It exists only when `castrum.WithBindings` was supplied.

For named actions, check the action name, physical input type, modifier, axis direction, and whether the system runs after the engine's input update system in the same phase. Read action edges in fixed systems when the action must reach simulation ticks; the engine's input systems carry the frame view into the fixed tick.

For raw input, check the phase and the snapshot contents. Frame input is published by the runner before frame systems run. A raw cursor or key read in a headless test needs a test snapshot assigned to `g.Context().Input`.

The [input guide](input.md) and [input example](../../examples/input) show named actions and raw mouse input together.

## An entity is not visible

Check the rendering path in order:

1. Confirm the entity has both `Transform` and `Sprite`.
2. Confirm `Sprite.Drawable` is non-nil. A zero `Sprite` is valid but invisible.
3. For `TextureSource`, verify the texture path and filesystem. For `AtlasSource`, verify the registered atlas and region.
4. Confirm the entity is inside the active camera viewport and that the camera zoom is positive.
5. Check `Hidden`, color alpha, layer, and sort order.
6. Inspect the returned error. Lazy texture and atlas resolution failures can surface during the first rendered frame.

Every game starts with an engine camera. A user-spawned `Camera{Primary: true}` takes precedence, so check that its transform and zoom are intentional and that there is only one active user primary. See [Rendering](rendering.md) for drawable bounds, camera selection, ordering, and culling.

## A system runs at the wrong time

`PhaseFrame` runs once per display frame with elapsed frame time. `PhaseFixed` runs zero or more times after it with the fixed tick interval. `PhaseStartup` runs once before the loop. Check that the system is registered in the phase matching its responsibility and that it uses that phase's `ctx.DeltaTime`.

Within a phase, systems run in registration order. Register a reader after the system that writes the value it consumes. Engine systems are registered during `New`, before game systems are added; this matters for transform snapshots and configured input systems.

For visual stutter, do not immediately increase `FixedTPS`. Fixed motion is rendered between the previous and current transform states. Inspect frame cost, interpolation inputs, and the camera before changing the simulation rate.

## The loop stops or a system behaves late

`Game.Advance` runs frame systems first, then as many fixed ticks as elapsed time requires. It clamps elapsed time to `MaxFrameTime` (250ms by default), and drops excess backlog after `MaxTicksPerFrame` fixed ticks (5 by default). A long pause can therefore produce fewer simulation ticks than the wall-clock duration suggests.

A returned system error stops the current `Advance` and propagates through the runner. Keep the registered system name in the error when diagnosing which system stopped the loop. `Game.Run` itself can be called only once; a second call returns an error.

The [scheduler reference](../reference/the-scheduler.md) describes the phase order and catch-up rules.

## A query gives unexpected results

Check the filter before changing the loop body:

- `With` requires each listed component.
- `Without` excludes each listed component.
- `Where` evaluates once per matching entity and can read only `With` types.
- `Entry.Component`, `Entry.SetComponent`, and `Entry.Update` access only prefetched `With` types.

Build the query once inside the system closure. Keep the `EntityID` if an entry must be used after the pass; entries use shared iteration storage and are not meant to be retained. Do not nest iterations over the same query.

Do not structurally change the world during `Execute`: adding or removing components, spawning, or destroying entities moves storage rows under the iterator. Collect IDs, finish the pass, then apply structural changes. Value writes through `Entry.Update` and `Entry.SetComponent` are supported.

## Where to look next

- [The ECS in depth](ecs.md) - entity, component, resource, system, and query contracts.
- [Assets](assets.md), [Rendering](rendering.md), and [Audio](audio.md) - feature-specific failure timing.
- [The scheduler](../reference/the-scheduler.md) - exact phase and fixed-loop behavior.
- [The API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) - exported signatures and types.
