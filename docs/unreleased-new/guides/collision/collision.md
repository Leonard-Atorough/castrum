# Collision

Collision lets your game find out when objects overlap or what lies along a line. You might collect an item when the player touches it, damage the player when they enter a hazard, or cast a ray to see what a weapon hits. Castrum detects these interactions; your game decides what they mean. It does not provide a physics solver or automatically move overlapping objects apart.

This guide adds collision to an existing game. It uses a player and pickups to show how colliders, filtering, and contact responses fit together, then looks at raycasts. The [collision example](../../../../examples/collision) brings players, walls, hazards, and pickups together in a runnable game; the [raycast example](../../../../examples/raycast) shows an aimed query.

## Give objects collision shapes

An entity needs both a `core.Transform` and a `collision.Collider` to participate. The transform places the object in the world; the collider describes the shape to test. A sprite is not required for collision, and its visible dimensions do not determine the collider's size.

For example, add a circular collider to an existing player. This excerpt assumes an error-returning setup function with a game `g` and a player handle. It uses `fmt` and the `collision` package:

```go
const (
	playerLayer = 0
	pickupLayer = 1
)

playerCollider, err := collision.NewCollider(collision.Circle{Radius: 16})
if err != nil {
	return fmt.Errorf("create player collider: %w", err)
}
playerCollider.Layers = collision.Layers(playerLayer)
playerCollider.Mask = collision.Mask(pickupLayer)
if err := player.AddComponent(g.World(), playerCollider); err != nil {
	return fmt.Errorf("attach player collider: %w", err)
}
```

`NewCollider` validates the shape and creates an active collider. This assumes the player does not already have one: `AddComponent` preserves an existing component rather than replacing it. Update the existing component if you need to change its shape or filter.

Now create a pickup that can interact with the player. `pickup` is a marker component your game uses to recognize the item; `position` is a `geom.Vector2`. This excerpt also uses `core`:

```go
type pickup struct{}
```

```go
pickupCollider, err := collision.NewCollider(collision.Circle{Radius: 10})
if err != nil {
	return fmt.Errorf("create pickup collider: %w", err)
}
pickupCollider.Layers = collision.Layers(pickupLayer)
pickupCollider.Mask = collision.Mask(playerLayer)
pickupCollider.Trigger = true
if _, err := g.World().NewEntity(
	core.Transform{Position: position},
	pickupCollider,
	pickup{},
); err != nil {
	return fmt.Errorf("create pickup: %w", err)
}
```

Add a sprite too if the pickup should be visible. The marker identifies the item to your game; the collider supplies its shape and filter.

Circles and boxes are defined in the entity's local space. Position and rotation move the collider with its transform, but transform scale does not resize it. Choose dimensions that match the interaction you want, and change the shape explicitly if its size must change.

## Choose what can interact

Each collider has **layers**, describing what it is, and a **mask**, describing what it listens for. Both colliders must listen to at least one layer the other belongs to. Above, the player listens for pickups and the pickup listens for the player; a one-sided mask would not produce a contact.

Layer numbers run from 0 to 31. `collision.Layers` and `collision.Mask` build bitmasks from those numbers; an out-of-range number panics. A collider with no layers or an empty mask cannot produce contacts, while an inactive collider is excluded from detection altogether.

The pickup is marked as a trigger so game code can distinguish it from something it might treat as solid. A trigger still produces contacts: the flag does not change filtering, collect the item, or suppress the overlap.

## Respond to a contact

Castrum maintains a `collision.Contacts` component for entities with a collider and transform. A fixed-phase system can read the player's current contacts and apply a game rule. This example removes contacted pickups and counts them:

```go
func collectPickups(player *core.Entity, collected *int) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		contacts, ok := player.Component[collision.Contacts](ctx.World)
		if !ok {
			if !player.HasComponent[core.Transform](ctx.World) || !player.HasComponent[collision.Collider](ctx.World) {
				return fmt.Errorf("collect pickups: player has no transform or collider")
			}
			return nil
		}
		for _, contact := range contacts.Current {
			other := core.NewEntity(contact.Other)
			if _, ok := other.Component[pickup](ctx.World); !ok {
				continue
			}
			if err := ctx.World.DestroyEntity(other); err != nil {
				return fmt.Errorf("collect pickup: %w", err)
			}
			*collected++
		}
		return nil
	})
}
```

Register the returned system in `core.PhaseFixed` and handle the `AddSystem` error. Keep the counter where your score display can read it. When the player overlaps a pickup, the system destroys that entity and increments the count. The marker check prevents an unrelated contact from being collected. Contacts may not exist before the first collision update; the system reports a missing transform or collider instead of silently waiting for an entity that cannot participate.

Each contact identifies the other entity and supplies a point, normal, penetration depth, and trigger flag. The normal points from this collider toward the other one. For a solid response, your game can use the normal and penetration to choose how to move an entity away; resolving several contacts is also your game's responsibility.

### React once, or while objects overlap

`Contacts.Current` holds the latest overlaps; `Contacts.Previous` holds the preceding tick's. Compare their `Other` entity IDs to distinguish an **enter** (new this tick), a **stay** (in both), and an **exit** (only in the previous set). The pickup example destroys the item, so it does not need a separate enter check. A hazard that remains in the world may need one to avoid applying damage every tick.

The engine also emits `collision.Entered` and `collision.Exited` signals, once for each side of a pair. Use these when you want reactions to lifecycle changes; read contacts when you need the full current set or stay behavior. The [signals guide](../../../unreleased/guides/signals.md) explains how to consume signals.

The engine owns `Contacts`; game code should read rather than modify it. Its slices are borrowed, read-only views valid until the next collision update. Copy the slices if you need to retain them longer.

## Find something along a line

A raycast answers a different question from an overlap: what is the first collider along this segment? It is useful for hitscan weapons and line of sight. It does not create a projectile or decide what a hit means.

From a system, resolve the collision queries resource and cast between two world-space positions. This helper excludes the player and accepts a mask for the target layers:

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

The helper uses `fmt` and the `collision`, `core`, and `geom` packages. Pass a mask such as `collision.Mask(enemyLayer)`, handle the error, and check the boolean before using the hit. The result gives you the entity ID, point, normal, and distance, so you can apply your game's rule.

The ray mask selects target `Collider.Layers`; a zero mask selects nothing. Unlike an overlap, a ray is not a collider pair, so the target's `Collider.Mask` is not checked. Inactive colliders and excluded entity IDs are not hits. To aim from the cursor, convert it to a world position as shown in the [input guide](../input/input.md#aim-at-a-world-position).

## Understand when collision is detected

Every `castrum.New` game registers collision detection before user systems in the fixed phase. Movement in `PhaseFixed` is therefore detected on the next fixed tick. Movement in `PhasePreFixed` can be detected in the current tick. Collision checks positions at those times, not a continuous sweep: a fast object can cross a small target between ticks.

Raycasts query the most recently synchronized collider data. Before the first successful collision update the index is empty; changes to transforms after an update appear in the index at the next sync. A frame-phase query sees the last completed collision update, not necessarily every change made since then.

If a contact is missing, check that both entities have transforms and active colliders, that both layer/mask directions allow the pair, and that a fixed tick has run since the relevant movement. For a missing ray hit, check its target mask and when the collision index last synchronized.
