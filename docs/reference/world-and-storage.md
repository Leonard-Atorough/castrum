# The world and storage

The public world model is a typed collection of entities, components, queries, and resources. The storage implementation may change; these are the behaviors a system can rely on.

## Entities

An entity has a durable `EntityID` and a lightweight `*Entity` handle. The handle carries liveness for convenience but no world storage; component methods receive the `World` explicitly.

Entity IDs are sequential and are not recycled during a world's lifetime. `World.DestroyEntity` removes the entity from storage and kills the supplied handle. `Entity.Kill` only marks the handle dead; it does not remove the entity or its components. Use `DestroyEntity` when the entity should disappear from queries.

A query `Entry` is not an entity handle. Its ID is safe to retain, but the entry itself points at query-owned scratch columns and is valid only during its iteration. Create a fresh handle with `core.NewEntity(id)` when an entity method is needed later.

## Components and structural changes

An entity holds at most one component value of each Go type. `Component` reads it, `SetComponent` replaces it, and `Update` applies a function to a copy and writes the result back. `AddComponent` and `RemoveComponent` change the entity's component set while preserving the remaining values.

Castrum validates components before they enter storage. That includes spawning and writes through `SetComponent`, `Update`, and `AddComponent`. A failed write leaves the prior stored value unchanged.

Entities with the same component set are stored together. This is observable through query behavior and performance, but the storage layout itself is private. Adding or removing a component moves an entity between sets, which is why structural changes are different from value writes.

Do not spawn, destroy, add, or remove components during a query iteration. A structural change moves rows under the iterator. Collect IDs, finish the pass, and apply structural changes between passes. Value writes through `Entry.SetComponent` and `Entry.Update` are supported during iteration.

## Queries

A `Query` combines structural filters and an optional value predicate:

- `With` requires component types and makes those values available through each `Entry`.
- `Without` excludes entities with listed component types.
- `Where` runs once per matching entity and is not cached.
- `Execute` yields matching entries; `First` returns the first match.

Queries cache matching component sets until the world's structure changes. They reuse prefetched columns and allow one active iteration at a time. Matching sets are visited deterministically, and entities retain storage order within each set while the world is unchanged.

Entries expose only the component types listed in `With`. Keep the query reusable and narrow; the [performance guide](../guides/performance.md) explains the cost model.

## Resources

`World.Provide` registers a typed constructor for lazy resolution. The first `World.Resource[T]` call resolves it and caches the instance. `ProvideEager` marks a constructor for eager resolution; `World.ResolveEager` resolves all such resources and returns the first error.

`castrum.Game` provides the engine's asset server, clip store, and mixer during construction. The runner adds backend resources such as texture and audio providers. Resource registration must happen before startup when a startup system depends on it.

Only one resource of a given type can be registered in a world. Constructors receive the world, so they can depend on other resources, but circular resolution returns an error.

## Where to go next

- [The ECS in depth](../guides/ecs.md) - the authoring vocabulary.
- [The scheduler](the-scheduler.md) - when systems using the world run.
- [Core principles](core-principles.md) - why state, validation, and ownership work this way.
