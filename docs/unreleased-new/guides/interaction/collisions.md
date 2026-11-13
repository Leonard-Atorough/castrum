# Detect and respond to collisions

Your player can move through the world, but how do you know when they reach a pickup or touch a hazard? Add colliders to the entities that should interact, then read the contacts the engine detects.

Castrum detects overlaps; it doesn't push entities apart, block movement, or decide what a hit means. Those are your game's rules. This guide adds a collectible to a world with an existing visible player. For controls, see [Give the player controls](controls.md); for entities and systems, see the existing [ECS guide](../../../unreleased/guides/ecs.md).

## Give the player and pickup colliders

Assume `player` is an entity with a `core.Transform` and a visible sprite, but no collider yet. Add a circular collider to it, then create a visible pickup at the given world position:

```go
const (
	playerLayer = 0
	pickupLayer = 1
)

type pickup struct{}

func placePickup(world *core.World, player *core.Entity, position geom.Vector2) error {
	playerCollider, err := collision.NewCollider(collision.Circle{Radius: 16})
	if err != nil {
		return fmt.Errorf("create player collider: %w", err)
	}
	playerCollider.Layers = collision.Layers(playerLayer)
	playerCollider.Mask = collision.Mask(pickupLayer)
	if err := player.AddComponent(world, playerCollider); err != nil {
		return fmt.Errorf("attach player collider: %w", err)
	}

	pickupCollider, err := collision.NewCollider(collision.Circle{Radius: 10})
	if err != nil {
		return fmt.Errorf("create pickup collider: %w", err)
	}
	pickupCollider.Layers = collision.Layers(pickupLayer)
	pickupCollider.Mask = collision.Mask(playerLayer)
	pickupCollider.Trigger = true
	if _, err := world.NewEntity(
		core.Transform{Position: position},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 10, Y: 10}},
			Color:    color.NRGBA{R: 255, G: 220, A: 255},
		},
		pickupCollider,
		pickup{},
	); err != nil {
		return fmt.Errorf("create pickup: %w", err)
	}
	return nil
}
```

This example uses `fmt`, `image/color`, and the `collision`, `core`, `geom`, and `render` packages. Call `placePickup` once during setup, choosing a position the player can reach. If setup fails, return the error rather than starting with a partially configured world. If your player already has a collider, update that component instead: `AddComponent` preserves an existing component rather than replacing it.

The marker component `pickup` lets game logic recognize the entity; the collider supplies its shape. `NewCollider` validates the shape and starts it active. Here the player belongs to `playerLayer` and listens for `pickupLayer`; the pickup does the reverse. **Both** masks must admit the other collider for the pair to produce a contact. Layer numbers range from 0 to 31.

The pickup's `Trigger` marks contacts involving it as triggers. It doesn't change filtering or automatically collect anything. Colliders use local-space circles or boxes, transformed by position and rotation; transform scale does not resize them. Choose dimensions that match the visible objects.

## Collect on contact

Keep a score for the round and register a system that removes contacted pickups:

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

After placing the pickup, register this system and keep `collected` available to your score display:

```go
collected := 0
if err := g.AddSystem(core.PhaseFixed, "player.collect", collectPickups(player, &collected)); err != nil {
	return fmt.Errorf("register pickup collection: %w", err)
}
```

Move the player into the pickup. It should disappear and increment `collected` once. The system checks the marker, not just `Trigger`, so another kind of trigger won't be collected by mistake. If you have a score display, update it after `"player.collect"` so it can show the new value on the same tick.

The engine creates and owns `collision.Contacts`; game code reads it rather than attaching or changing it. `Current` contains the latest overlaps and `Previous` the preceding tick's. These slices are read-only and valid only until the next collision update. This example removes a pickup immediately, so it doesn't need to detect a separate enter edge. If an object stays in place and should react only once per encounter, compare `Current` and `Previous` or use collision enter signals; the existing [collision guide](../../../unreleased/guides/collision.md#read-the-contact-lifecycle) shows both approaches.

## Choose when to respond

Every `castrum.New` game registers collision detection. It runs before user systems in the fixed phase: a player moved by a fixed-phase system is checked at the next fixed tick. Movement in `PhasePreFixed` can be detected in the current tick instead. Collision sees positions at those checks, not a continuous sweep, so a fast object can cross a small target between ticks.

Contacts report the other entity, a representative point, a normal, and penetration. For a solid wall you might use that information to implement your own separation or movement policy; handling multiple contacts is also your game's responsibility. The [collision example](../../../../examples/collision) shows hazards and trigger pickups together.

## Find the first target along a line

For a hitscan weapon or line-of-sight check, use a raycast rather than creating a moving projectile:

```go
func firstTarget(ctx *core.Context, player *core.Entity, from, to geom.Vector2) (collision.RaycastHit, bool, error) {
	queries, err := ctx.World.Resource[*collision.Queries]()
	if err != nil {
		return collision.RaycastHit{}, false, fmt.Errorf("get collision queries: %w", err)
	}
	hit, ok := queries.Raycast(
		geom.Segment{Start: from, End: to},
		collision.RaycastFilter{
			Mask:    collision.Mask(pickupLayer),
			Exclude: []core.EntityID{player.ID()},
		},
	)
	return hit, ok, nil
}
```

Handle the error first, then use the boolean to distinguish a hit from an empty line. A hit includes the entity, point, normal, and distance. Unlike overlap filtering, the ray's mask selects target layers without checking each target's `Collider.Mask`. The index reflects the most recent collision update; before its first successful update it is empty, and later transform changes aren't reflected until the next sync. The [raycast example](../../../../examples/raycast) demonstrates a complete aiming workflow.

If contacts are missing, check that both entities have transforms and active, valid colliders; that both layer/mask directions match; and that a fixed tick has run since the relevant movement. For shape details, contact signals, and more filtering examples, see the existing [collision guide](../../../unreleased/guides/collision.md).
