# Spatial architecture

Castrum's collision system uses a spatial-hash grid: a uniform grid whose occupied cells are stored in a hash map. It indexes the axis-aligned bounds of cached world-space shapes, finds nearby candidate pairs, and tests their exact geometry. Engine-owned contact state exposes the overlaps to gameplay.

This reference describes the grid, its integration with the ECS, and the invariants that preserve contact history. Use the [collision guide](../guides/collision.md) to create colliders and respond to contacts, and the [performance guide](../guides/performance.md) to budget and profile a game.

## Ownership and scope

| Layer                                                   | Responsibility                                                                                   |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `collision.Collider` and `core.Transform`               | Game-authored shape, filter, activation, and transform inputs.                                   |
| [Collision system](../../collision/system.go)           | World-space proxies, dirty detection, candidate pairs, and contact publication.                  |
| [Internal spatial grid](../../internal/spatial/grid.go) | Entity bounds and conservative region queries; no knowledge of exact shapes or gameplay filters. |
| [collision.Contacts](../../collision/contacts.go)       | Engine-written current and previous overlap records for game code to read.                       |

The grid is private implementation machinery, not a game-facing spatial-query API. It is not currently shared with rendering or camera culling.

Collision is detection-only: no response solver, event queue, or continuous collision detection. It tests boxes and circles at the sampled transform, so a fast-moving shape can cross another between ticks without producing a contact. Position and rotation affect geometry; transform scale and transform offset do not. A collider's own local offset does.

With `castrum.WithCollision`, detection runs in the fixed phase before gameplay systems. Movement made by those systems is detected on the following fixed tick. See [the scheduler](the-scheduler.md) for the ordering contract.

## Detection pipeline

Each collision update follows the same stages:

```text
Reconcile Contacts components
    -> Synchronise world-space proxies and grid memberships
    -> Collect candidate pairs
    -> Test exact shapes
    -> Publish current and previous contacts
```

Reconciliation collects entity IDs before adding or removing components; it does not structurally mutate the world during a query pass. Adding fresh contact state or detaching orphaned state also discards the corresponding private contact buffers, so old history is not reused after reconciliation.

Broad-phase results are conservative. Sharing a cell, or even overlapping axis-aligned bounds, does not prove that two rotated boxes or circles overlap.

## Sparse uniform grid

The grid divides world space into square cells. The collision system uses a private cell size of 50 world units. A world position maps to a cell by applying floor division independently to each axis:

```text
cell.X = floor(position.X / cellSize)
cell.Y = floor(position.Y / cellSize)
```

Floor division keeps negative coordinates consistent. At a cell size of 50, an X position of `-1` belongs to cell `-1`, not cell `0`.

Each entity is indexed in every cell covered by its axis-aligned bounds. Minimum and maximum cells are inclusive: a bound ending exactly on a cell boundary includes that boundary's cell, preserving touching contacts.

For example, bounds from `(40, -10)` to `(60, 10)` occupy the cell range `(0, -1)` to `(1, 0)`, covering four cells:

```text
        X = 0    X = 1
Y = 0   entity   entity
Y = -1  entity   entity
```

Cell size changes occupancy and query cost, not which shapes collide. There is no requirement that a collider fit inside one cell or a fixed number of cells. Large bounds simply occupy more cells and cost more to maintain.

### Entries and memberships

The index has two views of the same entry:

- Entity ID -> one entry containing its ID, exact bounds, and occupied cell range.
- Cell coordinate -> a slice of pointers to the entries occupying that cell.

The cell map is sparse: empty regions consume no cells. Each map key identifies a cell by its full coordinates.

A single entry is shared by all its memberships. Updating its bounds does not require replacing a separate entry in every cell.

### Incremental updates

An update validates bounds before changing the index:

- For a new entity, allocate its entry and add each membership.
- If its cell range is unchanged, update the exact bounds and return.
- Otherwise, visit the old range to remove departed memberships, then visit the new range to add newly entered memberships.
- Leave shared memberships intact, and replace the entry's saved range only after both passes.

The saved range describes the old memberships until both passes finish. The new range describes the target memberships. For a single row, movement from cells `0, 1` to cells `1, 2` removes membership in cell `0`, preserves cell `1`, and adds cell `2`. Cells between disjoint old and new ranges are not visited.

Membership reuse applies only to existing entries. A new entry has no memberships, regardless of its initial field values.

Cell buckets are unordered slices. Removing an entry scans the slice, moves the last pointer into the vacated slot, clears the last slot, and shortens the slice. The cell is deleted when it becomes empty. Deletion cost depends on cell occupancy, and result order is intentionally unspecified.

### Queries and deduplication

`Query` and `QueryInto` return entities sharing the query's cells. `QueryOverlapsInto`, used by collision detection, additionally rejects entities whose indexed bounds neither overlap nor touch the query bounds. It still does not test exact shapes.

The `Into` methods reuse the caller's result slice when capacity permits. Invalid query bounds return an empty result.

An entry can occur in several visited cells. The grid emits it only at the first cell shared by the entry's range and the query's range:

```text
firstShared.X = max(query.minCell.X, entry.minCell.X)
firstShared.Y = max(query.minCell.Y, entry.minCell.Y)
```

Both cell ranges are contiguous rectangles, so every intersecting pair of ranges has exactly one such cell. Other memberships are still visited, but they do not emit duplicate IDs.

For an entry spanning cells `(0, 1)` to `(2, 2)` and a query spanning `(1, 0)` to `(3, 2)`, the first shared cell is `(1, 1)`. The query emits the entry there and skips it when visiting the other shared cells.

## Proxy synchronisation

A proxy caches the collider inputs, position, rotation, transformed shape, and enclosing bounds. Synchronisation compares inputs before constructing geometry:

| Change                                        | Work                                                            |
| --------------------------------------------- | --------------------------------------------------------------- |
| New active collider                           | Build geometry, index its bounds, and mark dirty.               |
| No collision-relevant input change            | Reuse geometry and memberships.                                 |
| Position, rotation, shape, or collider offset | Rebuild geometry; update the grid if bounds differ; mark dirty. |
| Layers, mask, or trigger only                 | Retain geometry and memberships, but mark dirty.                |
| Collider inactive or no longer participating  | Remove its proxy and memberships.                               |

A shape can change while keeping the same bounds. Such a change still marks the proxy dirty and affects exact testing; skipping a grid update does not mean skipping collision detection.

### Identity, ordering, and removal

An entity-ID map owns proxy lookup. Two reusable pointer slices record the previous and next active query order. If the next entity's ID matches the cached row's ID, synchronisation can reuse that pointer without a map lookup; otherwise it falls back to the map.

Query order is an optimisation hint, not identity. Archetype migrations can reorder results, so every positional cache hit verifies the entity ID. The cache retains proxies and IDs, never query `Entry` values.

Each participating proxy receives the system's internal update stamp. If fewer old proxies survived than were active previously, a sweep removes proxies without that stamp. The counter is independent of `Context.Tick`.

A failed synchronisation can already have indexed some new proxies. Before returning that error, the system rebuilds its active cache from the proxy map so a retry can also remove those proxies if their entities disappear. This is recovery bookkeeping, not a promise that failed updates roll back all internal state.

Synchronisation discovers input changes by scanning the collider query every update. Proxy reuse avoids geometry work for unchanged inputs; it does not eliminate this scan.

## Candidate pairs and exact tests

Candidates come from two sources:

1. Dirty proxies query neighbouring overlapping bounds, with layer/mask filtering before pair insertion.
2. Existing contact pairs are included again, even when neither proxy changed.

Every pair uses canonical entity-ID ordering, and a reusable tested set deduplicates discoveries. Existing contacts appear from both partners' perspectives, so only one side contributes the old pair.

Retesting existing pairs keeps stationary overlaps alive and lets changed, disabled, or removed partners end a contact. A missing proxy or disallowed filter produces no new contact for that pair.

The [narrow phase](../../collision/collision.go) uses oriented-rectangle tests and circle distance/closest-point tests. Touching counts as contact. Each hit produces two records with the same point and penetration, opposite normals, and a trigger flag set when either collider is a trigger.

## Contact history and buffer reuse

Private contact state exists only for entities with contacts being built or history still to clear. Empty, settled entities do not need per-update contact snapshots or component writes.

Each tracked entity has three slices:

| Slice      | Role                                     |
| ---------- | ---------------------------------------- |
| `current`  | Last published current contacts.         |
| `previous` | Last published previous contacts.        |
| `spare`    | Unpublished scratch for the next result. |

The scratch slice is reset to length zero and refilled during exact testing. Neither published slice is used as scratch while constructing the result.

Before publication, contacts are sorted by the other entity's ID. On a write, the buffers rotate:

```text
published Current  <- spare
published Previous <- old current
new spare          <- old previous
```

Both proposed published slices are compared with the existing state. If both are equal, the system skips the component write and does not rotate the buffers. Comparing only current contacts would lose lifecycle transitions:

| Update           | Current | Previous | Meaning      |
| ---------------- | ------- | -------- | ------------ |
| First overlap    | Hit     | Empty    | Enter        |
| Same overlap     | Hit     | Hit      | Stay         |
| Separation       | Empty   | Hit      | Exit         |
| Following update | Empty   | Empty    | Exit cleared |

Once both published slices are empty, the private state is removed. Destroyed entities lose their private state when publication finds no contact component.

The published slices are borrowed, read-only views valid until the next collision update. Game code retaining history must copy the slices, not just the `Contacts` struct. The [collision guide](../guides/collision.md#read-the-contact-lifecycle) covers their use.

## Performance characteristics

The main costs are separate:

- **Input discovery:** a linear collider-query scan, even in an unchanged world. Ordered proxy reuse reduces lookups, not the number of entities inspected.
- **Geometry:** shape construction for new or geometry-changed proxies.
- **Index maintenance:** old/new cell-range traversal and departed-cell bucket scans when a range changes; unchanged ranges only update bounds.
- **Candidate discovery:** cells and entries visited by dirty queries, plus existing contact pairs. Dense cells and very large bounds increase work.
- **Contacts:** exact tests, sorting per-entity results, comparisons, and publication when values change.

The grid does not guarantee subquadratic collision detection. Many overlapping bounds can produce a quadratic number of candidate pairs, and a scene can genuinely contain a quadratic number of contacts. Persistent overlaps also have a cost in a static world because old pairs are retested.

Scratch slices and pair-deduplication storage are reused between updates. Contact buffers retain their capacity while that entity's contact state remains tracked. Changed geometry, buffer growth, new grid memberships, and contact publication can still allocate; reuse is not a blanket zero-allocation guarantee.

The [collision benchmarks](../../benchmark/collision_bench_test.go) exercise empty static worlds, moving and sparsely moving populations, and persistent overlaps. Those workloads distinguish the input-scan floor from index maintenance and contact processing. The moving tick cases include gameplay transform writes; the detector-only case excludes those writes from its measurements.

## Verification

[Grid tests](../../internal/spatial/grid_test.go) cover membership preservation, origin and negative cells, invalid updates, deduplication, allocation reuse, and randomized comparisons with expected cell-range and bounds-overlap results.

[Collision-system tests](../../collision/system_test.go) cover input changes, filters and activation, removal and reactivation, touching bounds, contact history, allocation scaling, error recovery, and comparison with full-pair detection across movement and archetype migrations.
