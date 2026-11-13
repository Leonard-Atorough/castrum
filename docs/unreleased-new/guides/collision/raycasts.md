# Find a target along a line

Use a raycast for a hitscan weapon, line of sight, or another question about the first collider along a segment. Unlike a moving projectile, a raycast queries a line immediately; it doesn't create an entity or decide what a hit means.

In a system, resolve the collision queries resource and cast from a world-space start to an end. This helper excludes the player and accepts a mask for the target layers:

```go
func firstTarget(ctx *core.Context, player *core.Entity, from, to geom.Vector2, targetMask uint32) (collision.RaycastHit, bool, error) {
	queries, err := ctx.World.Resource[*collision.Queries]()
	if err != nil {
		return collision.RaycastHit{}, false, fmt.Errorf("get collision queries: %w", err)
	}
	hit, ok := queries.Raycast(
		geom.Segment{Start: from, End: to},
		collision.RaycastFilter{
			Mask:    targetMask,
			Exclude: []core.EntityID{player.ID()},
		},
	)
	return hit, ok, nil
}
```

The helper uses `fmt` and the `collision`, `core`, and `geom` packages. Pass a mask such as `collision.Mask(enemyLayer)`, handle the error, then check the boolean before using the hit. A hit includes the entity ID, point, normal, and distance; use them to apply your game's rule.

The ray mask selects target `Collider.Layers`. It does **not** check the target's `Collider.Mask`: unlike an [overlap between two colliders](contacts.md), a ray isn't another collider in a pair. Inactive colliders and excluded IDs aren't hits. A zero mask selects nothing.

## Know which positions are queried

The query uses the collision system's most recently synchronized collider data. Before its first successful fixed update, the index is empty. A transform changed after the latest collision update appears in the index on the next sync, not immediately. A frame-phase query therefore sees the last completed collision update.

To aim from the cursor, convert it to a world position using the [mouse guide](../input/pointer.md). The [raycast example](../../../../examples/raycast) shows the whole workflow, including a visual line and hit feedback.
