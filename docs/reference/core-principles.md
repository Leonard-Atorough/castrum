# Core principles

The engine-details tier is for readers who need to reason about Castrum's behavior, not just call its API. The [guides](../guides/conventions.md) teach game authors what to do; the generated [Go API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum) lists exported names and signatures. These pages explain the contracts and design choices between those two layers.

The reference pages are deliberately optional. Start with [getting started](../getting-started/concepts.md) for the mental model, use the [guides](../guides/index.md) to build features, and come here when scheduling, storage, asset loading, rendering, or runner boundaries need a closer look.

## The game declares, the engine decides

The game writes state; the engine makes it true. An entity with a `Sprite` renders. An entity with an audio `Source` plays. To steer either, the game writes fields, and the engine reconciles the world to match on its own schedule.

This split keeps scheduling where it belongs. Which tick, which frame, drawn interpolated by how much - these are engine decisions, made with the whole frame in view. A game issuing draw calls and play calls would be making them blind.

So the declare-and-reconcile shape is architecture, and the convenience is a side effect: a sprite appears without drawing code because the renderer is watching the world. The [runner boundary](runner-separation.md) supplies platform services, while the [scheduler](the-scheduler.md) decides when reconciliation happens.

## The zero value is a contract

Every component sits in one of two states: valid in its zero value, or rejected loudly at the moment it enters the world. That two-state rule is what makes struct-literal spawning safe - write the fields you care about, leave the rest at zero, and trust the result.

The reasoning is simple: silent misbehavior is the most expensive failure an engine can hand a game. A component that produces wrong pixels fails far from its cause, in a shipped game. A component that fails at spawn fails at the author's desk, with the error naming the rule that was broken.

The three classes a field can fall into - valid zero, no valid zero, zero-as-default - and the test for telling them apart are the [conventions guide's](../guides/conventions.md) to teach.

## Errors are values; state has one owner

Engine APIs return errors, and everything runs on the loop's thread. Both rules exist for one reason: castrum games should fail debuggably. A returned error naming the failing system beats a panic three frames later. One thread beats a race report.

The day-to-day rules - the error template, where the one accepted panic lives, and what the loop ownership boundary asks of game code - are in the [conventions guide](../guides/conventions.md). The [world and storage](world-and-storage.md) page explains the public ownership model in more detail.

## Keep the public seams small

Castrum's stable vocabulary is the `Game`, `World`, components, systems, queries, resources, and `Runner`. The internal storage and scheduling packages implement those concepts but are not extension points. A feature that needs backend access belongs at the runner boundary; a feature that describes game state belongs in components or resources.

This separation keeps the public API understandable and lets internal storage or scheduling change without changing how a game declares state. When extending the engine, begin with the public seam a feature needs rather than exposing an internal package merely because it is convenient.

## Reference map

- [Engine design](engine-design.md) - package ownership and dependency direction.
- [Runner separation](runner-separation.md) - the platform boundary and default runner.
- [World and storage](world-and-storage.md) - entity, component, query, and resource behavior.
- [The scheduler](the-scheduler.md) - phases, fixed ticks, interpolation alpha, and ordering.
- [The asset pipeline](asset-pipeline.md) - filesystem resolution, decoders, caching, and streaming.
- [The coordinate system](coordinate-system.md) - world space, projection, logical size, and interpolation.
