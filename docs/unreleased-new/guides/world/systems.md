# Update a group of entities

One enemy is easy to address by its handle. When a level has many enemies, you can give each one the same components and write a system that works on all of them.

We'll move enemies with a velocity, then remove any whose health has reached zero. This builds on the `Enemy` and `Health` components in [Create and change game objects](entities.md). The snippets assume a game `g` from `castrum.New`, use the development API, and return errors to the caller.

## Describe motion

Put this component at package level:

```go
type Velocity struct {
	PerSecond geom.Vector2
}
```

Import `github.com/Leonard-Atorough/castrum/geom`. Add a `Velocity` value when creating an enemy, alongside its transform, sprite, and health. For example, `Velocity{PerSecond: geom.Vector2{X: 60}}` means 60 world units per second to the right. Nothing moves yet; a component is data, not a system.

## Move matching entities

A **query** finds entities with the components a system needs. Add this function at package level:

```go
func moveEnemies(world *core.World) core.System {
	enemies := core.NewQuery(world).With(core.Transform{}, Velocity{}, Health{}, Enemy{}).Without(Stunned{})
	return core.SystemFunc(func(ctx *core.Context) error {
		for entry := range enemies.Execute() {
			velocity, _ := entry.Component[Velocity]()
			entry.Update(func(t *core.Transform) {
				t.Position = t.Position.Add(velocity.PerSecond.Mul(ctx.DeltaTime.Seconds()))
			})
		}
		return nil
	})
}
```

Import `github.com/Leonard-Atorough/castrum/core`. Register the system during setup:

```go
if err := g.AddSystem(core.PhaseFixed, "enemy.move", moveEnemies(g.World())); err != nil {
	return fmt.Errorf("register enemy movement: %w", err)
}
```

Import `fmt` for the registration and return the error from your setup function. After the runner starts, every entity with a transform, velocity, health, and the enemy marker moves according to its velocity, unless it has `Stunned`. Entities missing any required component are left alone.

The query is created once and reused each tick. The zero values in `With` and `Without` identify component types; they are not attached to an entity. `entry.Component[Velocity]` succeeds here because `Velocity` is listed in `With`. Only types listed in `With` are available through the entry, even if an entity has other components.

`PhaseFixed` is a good fit for movement: it runs on simulation ticks, and `DeltaTime` is the tick interval. A display frame can have zero, one, or several fixed ticks. For presentation that must run once per display frame, use a frame phase instead. The existing [scheduling guide](../../../unreleased/guides/system-scheduling.md) covers the other phases and their ordering.

### A note on validation

`entry.Update` writes the selected value directly and does not call the component's `Validate` method. Keep the values you produce valid. If a change needs the validation and error reporting of `Entity.Update`, retain the entity IDs and perform those validated writes after the query pass. [Create and change game objects](entities.md#reject-invalid-health) shows a validated component and an entity-handle update.

## Remove defeated enemies

An enemy with zero health should disappear. Add this system:

```go
func removeDefeated(world *core.World) core.System {
	enemies := core.NewQuery(world).With(Health{}, Enemy{})
	return core.SystemFunc(func(ctx *core.Context) error {
		var defeated []core.EntityID
		for entry := range enemies.Execute() {
			health, _ := entry.Component[Health]()
			if health.Current == 0 {
				defeated = append(defeated, entry.ID())
			}
		}
		for _, id := range defeated {
			if err := ctx.World.DestroyEntity(core.NewEntity(id)); err != nil {
				return fmt.Errorf("remove defeated enemy: %w", err)
			}
		}
		return nil
	})
}
```

The function also needs `fmt` imported. The enemy marker keeps this system from removing other entities that also have health, such as the player.

Don't destroy, spawn, or add or remove components while a query is being iterated. Such structural changes can invalidate the pass. Instead, the example saves IDs, finishes the pass, and then removes the entities. A query entry is only valid during iteration; an ID is safe to keep and can be turned into a handle afterward.

## Put the work in order

If cleanup should happen before movement, register cleanup first and place movement immediately after it. This replaces the earlier standalone movement registration:

```go
if err := g.AddSystem(core.PhaseFixed, "enemy.cleanup", removeDefeated(g.World()), castrum.InGroup("gameplay")); err != nil {
	return fmt.Errorf("register enemy cleanup: %w", err)
}
if err := g.AddSystem(core.PhaseFixed, "enemy.move", moveEnemies(g.World()), castrum.InGroup("gameplay"), castrum.After("enemy.cleanup")); err != nil {
	return fmt.Errorf("register enemy movement: %w", err)
}
```

Import `github.com/Leonard-Atorough/castrum` as well. `After` places movement immediately after the already registered cleanup system, so an enemy already at zero health is removed before movement finds it. A system that applies damage needs to run before cleanup if its damage should be considered by that cleanup pass.

Both systems belong to the `gameplay` group. To stop them together later, call `g.SetGroupEnabled("gameplay", false)` and handle its error. This doesn't destroy their entities or automatically change a scene. Groups start enabled; a change also affects later systems in the current phase.

`castrum.Before` and `castrum.After` place a newly registered system relative to an existing user system in the same phase; they do not declare permanent dependencies or reorder engine-owned systems. A later insertion after the same target goes closest to it. For a runnable ordering example, see [scheduling](../../../../examples/scheduling/main.go).

Next: [share a score between systems](resources.md).
