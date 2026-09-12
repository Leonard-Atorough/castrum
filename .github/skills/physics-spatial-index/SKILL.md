---
name: physics-spatial-index
description: >
  Implementation plan for Castrum's 2D physics and spatial indexing work.
  Consult when changing collider transforms, rotated or scaled bounds,
  broad-phase collision queries, spatial backend ownership, off-screen
  simulation, or future render culling/frustum support.
---

# Physics and Spatial Index Plan

This skill is the standing implementation reference for the physics and spatial
index work. Keep the changes phased, testable, and consistent with the current
ECS-based Castrum architecture.

## Core decisions

1. **Physics is independent of camera visibility.** Off-screen entities must
   continue to simulate and collide unless an explicit simulation policy later
   puts them to sleep, streams them out, or otherwise disables them.
2. **Collider shapes are local-space geometry.** `Transform.Position`,
   `Transform.Rotation`, and `Transform.Scale` determine world geometry.
3. **Broad phase uses conservative world-space AABBs.** It may return false
   positives, which the narrow phase rejects, but it must not return false
   negatives.
4. **Narrow phase uses exact transformed geometry.** A rotated rectangle is an
   oriented rectangle, not an AABB. Use OBB/SAT logic when rotated rectangle
   collision is supported.
5. **The spatial backend is generic.** It indexes IDs and bounds only. It must
   not depend on ECS worlds, colliders, cameras, or rendering.
6. **Do not create a frustum package yet.** Frustum/visibility work is a future
   render concern. Design the backend so it can support visibility queries later
   without making visibility part of physics.
7. **Do not extract the current point index unchanged.** First establish the
   correct bounds-based contract, then move the implementation behind a generic
   internal package.

## Current problems to preserve as regression targets

- `physics/spatial.go` indexes an entity by one position cell, not by its
  collider bounds.
- A large collider can span several cells and be missed by the current index.
- `Query(position, radius)` is a physics-shaped API and is not an exact
  geometric query.
- `physics/collision.go` translates shapes by position but ignores rotation and
  scale.
- `PhysicsSystem` marks entities dirty based on position changes only, so
  rotation-only, scale-only, and shape-only changes can be missed.
- `Collider.BoundingBox()` describes local geometry but is not transformed for
  world use.
- The narrow phase currently handles circle and rectangle combinations; do not
  claim complete polygon collision support until `intersectsAny` implements it.
- `PhysicsConfig.Enabled` must have tested, observable behavior.
- Renderer bounds and camera visibility are separate concerns and must not gate
  collision simulation.

## Non-goals for the first milestone

Do not include these in the initial physics/index refactor:

- A frustum or visibility package.
- Multi-camera or viewport culling.
- Continuous collision detection.
- Rigid-body collision response, gravity, or constraints.
- Dynamic trees or sweep-and-prune optimization without profiling evidence.
- A public all-purpose spatial-query service.
- Polygon collision unless it is explicitly made part of the milestone.

## Phased implementation

### Phase 0: Establish the contract with tests

Start with focused regression tests before changing implementation.

Primary files:

- `physics/spatial_test.go`
- `physics/collision_test.go`
- A focused geometry test file only if needed

Add tests covering:

- Multi-cell collider bounds are indexed in every overlapped cell.
- Updating an entity removes stale cell memberships.
- Removing an entity removes every membership.
- Negative coordinates still map correctly.
- Query results are deduplicated.
- Invalid bounds containing `NaN` or infinity are rejected.
- A 90-degree rotation swaps a rectangle's world AABB dimensions.
- A 45-degree rotation produces the expected conservative AABB.
- Rotation-only and scale-only changes update the physics proxy.
- Rotated rectangles are not missed because their center is outside a naïve
  point-cell query.
- `PhysicsConfig.Enabled == false` performs no collision processing/events.
- Inactive colliders are removed from the index.
- Two colliders outside any camera view can still collide.

The off-screen test should not require a camera. Its purpose is to establish
that physics is world-space simulation, not render culling.

### Phase 1: Add transformed world-shape calculation

Work first in `physics/collision.go` and its focused tests.

Create an internal world-shape representation containing:

- the exact transformed shape for narrow-phase testing;
- the conservative world-space AABB for broad-phase indexing.

Transformation order:

1. Read local shape geometry.
2. Apply `Transform.Scale`.
3. Apply `Transform.Rotation` around the local origin.
4. Translate by `Transform.Position`.
5. Build the world-space AABB from transformed points or shape bounds.

Preserve local offsets. A circle center or rectangle that is not centered at
local origin must transform correctly.

For circles, decide and test the non-uniform scale policy. The conservative
initial policy is to use the maximum scale component so the result remains a
circle and cannot under-approximate the real extent. Do not silently model an
ellipse as a circle with a smaller radius.

If rotated rectangle colliders are supported in the public contract, implement
OBB collision with a Separating Axis Theorem test. A useful initial shape
matrix is:

- Circle vs Circle: exact.
- Circle vs AABB: exact.
- Circle vs OBB: exact.
- AABB vs AABB: exact.
- AABB vs OBB: SAT.
- OBB vs OBB: SAT.
- Polygon: separate follow-up unless explicitly required now.

### Phase 2: Change the spatial index to bounds

Update `physics/spatial.go` and `physics/spatial_test.go` only after the
world-shape bounds exist.

Replace the conceptual API:

```go
Update(entityID, position)
Query(position, radius)
```

with a bounds-oriented API:

```go
Update(entityID, bounds)
Query(bounds)
```

The grid must:

- accept a valid AABB;
- index every cell overlapped by that AABB;
- remove old memberships before adding new memberships;
- remove all memberships on entity removal;
- deduplicate query results;
- return conservative candidates;
- retain buffer reuse through a `QueryInto`-style method where useful.

The index must not perform narrow-phase collision tests.

### Phase 3: Integrate bounds into `PhysicsSystem`

Update `physics/physics.go` so `syncIndex` computes a transformed world shape
for every active collider.

The synchronization path should:

1. Query active colliders and transforms.
2. Compute the transformed shape and its world AABB.
3. Update the spatial proxy with that AABB.
4. Detect changes to position, rotation, scale, shape, and active state.
5. Mark changed entities dirty.
6. Query candidates using the dirty entity's AABB.
7. Run exact narrow-phase checks for candidate pairs.
8. Preserve enter/stay/exit pair-cache behavior.

Replace or expand `lastPositions`; position alone is not sufficient proxy state.
A cached world bounds/proxy state is preferable to silently relying on position
comparisons.

`PhysicsConfig.QueryRadius` should stop being the primary collision boundary.
Keep it temporarily only if compatibility requires it, and treat it as an
explicit optional search limit rather than the default correctness mechanism.
The normal path should query by collider AABB.

Honor `PhysicsConfig.Enabled` explicitly and add a regression test.

### Phase 4: Verify system ordering

Check the registration and scheduling path in `castrum.go` and the system
manager. The intended order is:

```text
input
  -> gameplay movement
  -> transform changes
  -> physics broad phase and narrow phase
  -> collision events
  -> render extraction/culling
  -> drawing
```

If physics currently runs before movement, either assign an explicit priority
or document and enforce the required ordering. Do not fix this by coupling
physics to rendering.

### Phase 5: Extract the generic backend

Only after the bounds-based index works inside physics, extract it to:

```text
internal/spatial/
    grid.go
    grid_test.go
```

The package may depend on `geom.Rect` and IDs, but must not import:

- `components`;
- `ecs.World`;
- `physics`;
- `render`;
- camera types.

Keep a private physics adapter/interface, evolving the current
`spatialIndexer` seam toward bounds operations, for example:

```go
type spatialIndexer interface {
    Update(id ecs.EntityID, bounds geom.Rect) error
    QueryInto(bounds geom.Rect, ids []ecs.EntityID) []ecs.EntityID
    Remove(id ecs.EntityID)
}
```

The backend owns cell membership. Physics owns collider lifecycle, filtering,
exact collision, pair state, and events.

### Phase 6: Defer visibility and frustum work

When render culling needs the backend, use this direction:

```text
internal/spatial
  generic AABB index

render
  camera
  viewport
  visibility query
  layer filtering
```

For a rotated 2D camera, first query using the conservative AABB of the
rotated viewport, then optionally perform an exact visibility test. Do not let
camera visibility determine whether physics runs.

A separate frustum package is justified only when the engine needs rotated
camera visibility, multiple viewports, convex visibility regions, portals,
visibility masks, or similar render-specific behavior.

## File-level work breakdown

1. **Contract tests**
   - `physics/spatial_test.go`
   - `physics/collision_test.go`
2. **World-shape transformation**
   - `physics/collision.go`
   - focused collision tests
3. **Bounds-based grid**
   - `physics/spatial.go`
   - `physics/spatial_test.go`
4. **Physics integration**
   - `physics/physics.go`
   - collision/event tests
5. **Backend extraction**
   - new `internal/spatial` package
   - physics adapter and backend tests
6. **Visibility follow-up**
   - `render`, camera components, and possibly `internal/spatial`
   - deferred until the physics milestone is stable

Keep each phase independently buildable and testable. Do not perform the
backend extraction in the same change as the first geometry correction unless
there is a concrete ownership blocker.

## Milestone definition of done

The first physics milestone is complete when:

- rotated and scaled colliders have correct world geometry;
- broad-phase indexing uses transformed AABBs;
- large and multi-cell colliders cannot be missed;
- rotation-only and scale-only changes trigger re-evaluation;
- exact narrow phase rejects broad-phase false positives;
- off-screen entities collide exactly like visible entities;
- inactive colliders leave the index;
- `PhysicsConfig.Enabled` has tested behavior;
- the spatial backend has no camera or renderer dependency;
- `go vet ./...` and `go test ./...` pass.

The intended stopping point is a generic internal bounds index plus a correct
physics adapter. Do not broaden it into a frustum package until render
visibility has a concrete requirement.
