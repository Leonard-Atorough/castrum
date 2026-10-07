# Performance

Performance work starts with a measurement and a budget. Castrum protects the loop from unbounded catch-up and reuses several internal buffers, but your game still decides how much simulation, query, rendering, asset, and audio work happens in that budget.

Do not begin by changing a limit or adding a cache. First identify whether the cost is in fixed systems, frame systems, entity matching, visible rendering, asset decoding, audio memory, or your own code. Then change the smallest responsible part and measure again.

## Understand the loop budget

The game has a display-frame phase and a fixed simulation phase:

- Frame systems run once for each display frame and receive the elapsed platform time in `Context.DeltaTime`.
- Fixed systems run once for each simulation tick that the accumulator can pay for and receive the fixed tick interval in `Context.DeltaTime`.

`FixedTPS` is 60 by default. A fast display frame may run zero fixed ticks; a slow frame may run several. The engine always runs at most `MaxTicksPerFrame` fixed ticks in one display frame, five by default. It also clamps the elapsed time added to the accumulator with `MaxFrameTime`, which defaults to 250 milliseconds.

These limits are stability guards:

- `MaxFrameTime` prevents a pause, breakpoint, or long stall from adding an unbounded amount of work.
- `MaxTicksPerFrame` prevents a slow frame from starting an endless catch-up spiral.
- When the catch-up cap is reached, the remaining backlog is dropped. Simulation time is intentionally lost rather than carried into every later frame.

Set the options deliberately when the game needs a different simulation contract:

```go
g, err := castrum.New(
  castrum.WithFixedTPS(60),
  castrum.WithMaxFrameTime(250*time.Millisecond),
  castrum.WithMaxTicksPerFrame(5),
)
if err != nil {
  return err
}
```

Do not raise `FixedTPS` merely because rendering looks uneven. Rendering interpolates transforms between fixed ticks; inspect frame work and interpolation first. The [scheduler reference](../reference/the-scheduler.md) describes the complete loop and phase ordering.

## Measure before tuning

Measure the workload that is actually slow:

- Time fixed systems separately from frame systems.
- Record entity counts and the number of entities each important query matches.
- Compare total sprites with visible sprites; collection, culling, sorting, and drawing have different costs.
- Track eager audio memory for long or numerous tracks.
- Reproduce engine-level costs with a small benchmark or test before changing the engine around them.

Castrum does not provide a universal sprite, query, or audio threshold. Hardware, asset sizes, world density, and the runner backend all change the answer. A measurement is more useful when it names the phase, system, entity count, and workload that produced it.

## Keep systems within their phase budget

Use the phase that matches the work:

- Put simulation, movement, and gameplay rules in `PhaseFixed` so they advance by a stable `DeltaTime`.
- Put display-frame work, presentation updates, and raw frame input in `PhaseFrame` when they do not need fixed-step determinism.
- Put one-time setup and required asset loading in startup or setup code.

Every system in a phase runs in registration order. A system that matches a large world therefore pays its cost every time that phase runs. If one system writes state that another reads in the same tick, register the reader after the writer; do not add redundant work to recover from unclear ordering.

The engine's own systems run before user systems. That ordering supports transform interpolation, input publication, animation advancement, and audio reconciliation. Treat it as a contract rather than trying to duplicate those systems in game code.

When one fixed tick cannot finish a large job, make the job incremental: keep its progress in a resource or component and process a bounded amount per tick. A large one-shot loop still blocks the fixed phase even if the overall entity count is reasonable.

## Make queries narrow and reusable

Queries cache their matching archetypes until the world's structure changes, and they reuse their prefetched component columns. Build a query once inside the system closure instead of rebuilding it on every tick:

```go
func damageSystem() core.System {
  var targets *core.Query
  return core.SystemFunc(func(ctx *core.Context) error {
    if targets == nil {
      targets = core.NewQuery(ctx.World).
        With(Health{}, Damage{})
    }
    for e := range targets.Execute() {
      e.Update(func(health *Health) {
        damage, _ := e.Component[Damage]()
        health.Current -= damage.Amount
      })
    }
    return nil
  })
}
```

Use `With` and `Without` for stable component-shape filters. They are evaluated at the archetype level. Use `Where` for a value-dependent condition, but remember that its predicate runs once per matching entity on every pass:

```go
active := core.NewQuery(ctx.World).
  With(Transform{}, Enemy{}).
  Where(func(e core.Entry) bool {
    enemy, _ := e.Component[Enemy]()
    return enemy.Active
  })
```

Keep the `With` set to the components the loop actually needs. Prefetching unrelated components increases the data moved into the query's working columns and makes the system's dependency less clear.

Do not spawn, destroy, add, or remove components while a query is executing. Structural changes move rows under the iterator. Write existing components with `SetComponent` or `Update`, or collect entity IDs and apply structural changes after the pass. A query supports one active iteration at a time, so nested passes need separate query values.

When many entities start with the same component set, use `World.NewEntities` rather than repeating setup logic. This improves setup clarity and avoids rebuilding the same component list in game code; measure before treating it as a runtime optimization.

## Reduce rendering work

The renderer collects every non-hidden entity with `Sprite`, `Transform`, and the engine-managed previous transform. For each candidate it may resolve a texture or atlas region, calculate bounds, cull against the camera, and add the visible item to a sorted list.

The collector reuses its working and sorted buffers across frames. That avoids an allocation per sprite in steady state, but visible sprites still cost collection, sorting, texture resolution, and backend drawing. The main levers are:

- Do not keep unnecessary drawable entities alive when they are no longer part of the scene.
- Use `Hidden` when an entity should remain in the world but should not be collected or drawn.
- Keep off-screen populations reasonable; culling prevents their draw calls, but they still cost query matching and bounds checks.
- Use `Layer` and `SortOrder` intentionally, and avoid creating needless visible items solely to control ordering.
- Use shapes for simple geometry when an image asset is not needed, and choose texture or atlas regions according to the visual result you need.
- Keep debug and HUD drawing bounded; runner overlays run once per display frame after the world.

The renderer sorts visible items by layer, then sort order, then interpolated world Y. Large visible sets make both collection and sorting worth measuring. An atlas can reduce texture resources and organize regions, but it does not make each visible sprite free: every visible atlas region is still a draw item.

Smooth motion comes from transform interpolation. Raising the fixed tick rate increases simulation work; it is not a general rendering optimization.

## Manage asset and audio costs

Use the asset server's typed cache for values that should be reused. Repeated `Load[T]` calls for the same asset identity, type, and format return the cached value, and concurrent loads of that key are coalesced. Preload required assets during setup when first-use decoding would be visible or when failure should prevent startup.

Decoded audio is a common memory cost. `asset.AudioData` stores 16-bit stereo PCM, so its approximate memory rate is:

```text
sample rate * 4 bytes per second
```

At 44,100 Hz, one minute is about 10.1 MiB before other overhead. Use eager audio for short effects that play often and share their decoded samples. Use streaming audio for long music or ambience that does not need the complete PCM buffer resident; see [Audio](audio.md).

Streaming still keeps one open source per active play and needs a seek-capable filesystem for looping or resets. It trades memory for ongoing file and decode work, so measure active stream counts and storage behavior rather than assuming it is always cheaper.

## Avoid accidental per-tick work

Keep reusable state in the system closure, a resource, or a component instead of rebuilding it every tick. Common avoidable costs include:

- Reconstructing queries, lookup tables, or asset IDs in a hot loop.
- Re-decoding an asset instead of using `Load[T]` or an appropriate playback mode.
- Allocating temporary slices for every entity when a bounded reusable buffer would do.
- Running work for inactive entities that could be excluded structurally or by a narrow query.
- Doing display-only work in every fixed tick.

Do not optimize by retaining `core.Entry` values after query iteration; entries point into reusable query scratch. Retain entity IDs or handles instead, then resolve them when the next operation needs them.

## When to change the engine limits

Change `FixedTPS` when the simulation itself needs a different tick interval and can afford the additional work. Change `MaxTicksPerFrame` only when the game's tolerance for catch-up and dropped time requires it. Change `MaxFrameTime` when the maximum recoverable elapsed interval is part of the game's design.

Changing these values cannot make an expensive system cheaper. A higher catch-up cap may make a slow frame spend even longer catching up; a lower cap drops more simulation time. Treat the resulting behavior as a gameplay and stability decision, then verify it with a representative workload.

## Where to go next

- [The fixed timestep](../getting-started/concepts.md) - ticks, `DeltaTime`, and interpolation.
- [The scheduler](../reference/the-scheduler.md) - the full loop mechanics and system ordering.
- [Rendering](rendering.md) - drawable, camera, culling, and sorting behavior.
- [Assets](assets.md) - typed caching, preloading, and streaming.
- [Audio](audio.md) - eager and streaming playback decisions.
- [Conventions](conventions.md) - the state and validation rules that keep hot paths predictable.
