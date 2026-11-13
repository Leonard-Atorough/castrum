# Detect and respond to overlaps

Give two entities colliders when their overlap should matter to the game. Castrum records the contact; your game decides whether it collects an item, deals damage, blocks movement, or does nothing. Collision detection is not a physics solver and does not move entities apart.

The examples assume an existing game `g`, a player entity with a `core.Transform`, and an error-returning setup function. The [collision example](../../../../examples/collision) shows moving players, hazards, and pickups in a complete program.

## Configure the colliders

Use named layer indexes to describe what can interact. This example gives the player a circular collider that listens for pickups:

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

The excerpt uses `fmt` and the `collision` package. It assumes the player does not already have a collider: `AddComponent` preserves an existing component. If one is already present, update it instead. `NewCollider` validates the shape and creates an active collider.

For a pickup, configure the other side and add it to an entity with a transform. `pickup` is a marker type defined by your game:

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

This excerpt also uses `core` and a `geom.Vector2` named `position`. Add a sprite if the pickup should be visible. The marker identifies the item to your game; the collider supplies its shape and filter.

`Layers` says what a collider belongs to; `Mask` says what it listens to. **Both** masks must admit the other collider for the pair to produce a contact. Layer indexes range from 0 through 31. `Trigger` marks contacts involving that collider but does not change filtering, collect the item, or suppress contacts.

Colliders use local-space circles or boxes, transformed by position and rotation. Transform scale does not resize a collider; choose its dimensions to match the object.

## Apply a game rule

The collision system owns `collision.Contacts`. Read the player's current contacts from a fixed-phase system and decide what to do with them. For example, this system removes entities carrying the `pickup` marker and counts them:

```go
func collectPickups(player *core.Entity, collected *int) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		contacts, ok := player.Component[collision.Contacts](ctx.World)
		if !ok {
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

Register the system in `core.PhaseFixed`, keep the counter where your score display can read it, and handle registration errors. A contacted pickup disappears and is counted once. The marker check prevents an unrelated trigger from being collected. Contacts may not exist before the first collision update; if they remain missing, check that the player still has both a transform and collider.

`Current` is the latest set of overlaps and `Previous` is the preceding tick's. The slices are borrowed, read-only views valid until the next collision update. This example destroys the pickup, so it doesn't need a separate enter check. For an object that remains in the world, compare the two sets or consume enter/exit signals to react once per encounter; see [contact lifecycle](../../../unreleased/guides/collision.md#read-the-contact-lifecycle).

## Account for detection timing

Every `castrum.New` game registers collision detection before user systems in the fixed phase. Movement performed by a fixed-phase system is therefore detected on the next fixed tick. Movement in `PhasePreFixed` can be detected in the current tick. Collision checks positions at those times, not a continuous sweep: a fast object can cross a small target between ticks.

To implement a solid response, use the contact's normal and penetration as inputs to your own movement or separation policy; resolving multiple contacts is also your game's responsibility. For a line-of-sight or hitscan query, use [a raycast](raycasts.md) instead.

If a contact is missing, check both transforms, collider activation, both layer/mask directions, and whether a fixed tick has run since the relevant movement. The existing [collision guide](../../../unreleased/guides/collision.md) covers contact geometry and signals in more depth.
