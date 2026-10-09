# Collision

Castrum collision is overlap detection, not a physics solver. Give an entity a `collision.Collider` and a `core.Transform`; the engine detects overlaps and writes them to `collision.Contacts`. Your systems decide whether a contact blocks movement, deals damage, collects a pickup, or does nothing.

The engine registers collision detection in the fixed phase before gameplay systems, when the game is created with `castrum.WithCollision()`. It tests the transforms at the start of that tick, so movement your systems make is detected on the following fixed tick. You do not register the collision system yourself.

For a complete runnable example, including walls, rotating hazards, trigger pickups, and contact lifecycle handling, run:

```text
go run ./examples/collision
```

For entity, component, and system basics, see [The ECS in depth](ecs.md).

## Create a collider

Enable the collision system when creating the game: `castrum.WithCollision()` registers it in the fixed schedule under `collision.SystemName`. Without the option, colliders never produce contacts. `castrum.WithDefaultSystems()` enables it together with the timer and animation systems.

The supported local-space shapes are `collision.Box` and `collision.Circle`. Create a collider with `collision.NewCollider`, then spawn it with a transform:

```go
collider, err := collision.NewCollider(collision.Circle{Radius: 12})
if err != nil {
	return err
}

player, err := world.NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 80}},
	collider,
)
if err != nil {
	return err
}
```

`NewCollider` starts the collider active, on layer 0, listening to every layer. Its shape and offset are validated immediately; the world validates the collider again when it is stored. Rectangles must have finite, strictly increasing bounds, circles must have a finite center and positive finite radius, and offsets must be finite. Invalid values are rejected, not clamped or repaired. The shape interface is sealed, so applications cannot add custom shape types.

Shapes are defined relative to the entity origin. `Offset` shifts the shape in local space; the system then applies the entity's rotation and position. Scale is intentionally ignored, including for circles. If a hitbox must change size, update its shape explicitly.

## Filter pairs with layers and masks

`Layers` says which layers a collider belongs to; `Mask` says which layers it listens to. A pair is tested only when both colliders listen to at least one layer the other belongs to. A one-sided mask is not enough.

Use named layer constants to keep assignments readable:

```go
const (
	playerLayer = 0
	hazardLayer = 2
)

playerCollider.Layers = collision.Layers(playerLayer)
playerCollider.Mask = collision.Mask(hazardLayer)

hazardCollider.Layers = collision.Layers(hazardLayer)
hazardCollider.Mask = collision.Mask(playerLayer)
```

`collision.Layers(0, 2)` and `collision.Mask(0, 2)` build `uint32` bitmasks from layer numbers. Layer indexes must be from 0 through 31; an out-of-range index panics. You can also assign raw `uint32` bitmasks directly.

An empty `Layers` value means the collider belongs to no layers; an empty `Mask` means it listens to none. Either makes it collide with nothing. This differs from `Active = false`, which opts the collider out of detection entirely. `Trigger` does not change filtering or suppress contacts; it marks the resulting contacts for game logic.

## Understand contact geometry

The narrow phase tests rotated rectangles as oriented rectangles and circles against rectangles by their closest point. Touching counts as a contact, with zero penetration at an ordinary touching boundary. Each `collision.Contact` reports:

- `Other`: the other entity's ID.
- `Point`: a representative point from the shape-pair test; it is not a contact manifold.
- `Normal`: a direction from this collider toward `Other`. The other collider's contact has the opposite normal.
- `Penetration`: overlap depth along the normal.
- `Trigger`: true if either collider in the pair has `Trigger` set.

For a circle fully inside a rectangle, penetration is the distance from the circle center to the nearest rectangle boundary plus the circle radius. It describes the depth needed to separate the shapes through that face.

## Read the contact lifecycle

The collision system attaches and owns `Contacts` for entities with both a collider and a transform. It writes the component each fixed tick and removes it if the collider or transform is removed. Game code should read `Contacts`; it should not attach, mutate, or remove it.

`Contacts.Current` contains this tick's contacts, while `Contacts.Previous` contains the prior tick's. Both slices are sorted by `Contact.Other`. Compare the `Other` IDs to derive lifecycle edges:

- **Enter:** in `Current`, not in `Previous`.
- **Stay:** in both.
- **Exit:** in `Previous`, not in `Current`.

For example, an enter helper can return the new contacts for a system to process:

```go
func entered(contacts collision.Contacts) []collision.Contact {
	var result []collision.Contact
	for _, hit := range contacts.Current {
		found := false
		for _, previous := range contacts.Previous {
			if previous.Other == hit.Other {
				found = true
				break
			}
		}
		if !found {
			result = append(result, hit)
		}
	}
	return result
}
```

Use the `Trigger` flag to distinguish interactions such as pickups from solid contacts. Because contacts are computed before gameplay systems run, those systems can read the current results, but changes they make to transforms will affect detection on the next fixed tick.

## Choose what a contact means

Collision detection never moves entities or resolves overlaps. A game that wants a player stopped must read the contact data and apply its own movement or separation policy. The contact normal points toward the other collider, so moving away from it generally means moving opposite the normal; handling several simultaneous contacts is also the game's responsibility.

A trigger is still an overlap: it produces ordinary contact data with `Trigger` set when either side is marked. Use that flag to implement interactions without treating them as blocking collisions. The [collision example](../../examples/collision/main.go) demonstrates both trigger pickups and solid hazard contacts.

## Where to go next

The [collision example](../../examples/collision/main.go) shows layered walls, rotating hazards, trigger pickups, and enter-edge handling in one game. For the component ownership and system-ordering rules behind `Collider` and `Contacts`, see [The ECS in depth](ecs.md). For broader fixed-phase and query-cost decisions, see [Performance](performance.md).
