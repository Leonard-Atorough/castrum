# Create and change game objects

A player, an enemy, or a projectile can be an entity in your game world. An entity is an identity; its **components** hold the data that makes it useful. Give an enemy a position and a picture, for example, and the renderer can show it. Add health, and your own game rules can decide what happens when it takes damage.

This guide builds an enemy with health, changes that health, and removes the enemy when it is no longer needed. It assumes you have a game `g` created by `castrum.New`, as in [Your first game](../../../unreleased/getting-started/your-first-game.md). The snippets use the development API and belong in Go code that returns errors.

## Give an enemy some health

A component is a Go value. Define one for the data you want to keep on each enemy:

```go
type Enemy struct{}

type Health struct {
	Current int
	Maximum int
}
```

Create an enemy in the game's world:

```go
enemy, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 40}},
	render.Sprite{Drawable: render.RectShape{Size: geom.Vector2{X: 32, Y: 32}}},
	Enemy{},
	Health{Current: 3, Maximum: 3},
)
if err != nil {
	return fmt.Errorf("create enemy: %w", err)
}
```

Import `fmt` and the `core`, `geom`, and `render` packages from Castrum. Put the type at package level and the creation code in an error-returning setup function. Once the runner is started, the enemy appears as a square at `(100, 40)` in the world. The returned `enemy` is a handle for working with that identity later; the world owns its component data.

`Enemy` is an empty marker component. It lets systems find enemies without accidentally selecting another entity that also has health. You can combine components as your game needs them. A transform without a sprite still has a position but is not drawn. A sprite's drawable says what to draw; the transform places it. See [Draw and position an image](../rendering/images.md) when you want to use an image instead of a shape.

## Change health

Use the handle and the owning world to update a component:

```go
func damageEnemy(world *core.World, enemy *core.Entity, damage int) error {
	if damage < 0 {
		return fmt.Errorf("damage must not be negative: %d", damage)
	}
	if err := enemy.Update(world, func(h *Health) {
		h.Current = max(0, h.Current-damage)
	}); err != nil {
		return fmt.Errorf("damage enemy: %w", err)
	}
	return nil
}
```

Calling `damageEnemy(g.World(), enemy, 1)` changes the enemy's health from three to two. The sprite doesn't change by itself: health is game data, and your own systems decide how it affects play.

`Update` reads a copy, applies the function, and writes the result back. It reports an error if the entity or component is missing or the write fails validation. Reading a component also returns a value; changing that value alone doesn't replace it in the world. Fields inside a component can still refer to shared data, so a copy of a slice or map is not a deep copy.

## Reject invalid health

If a value should always obey a rule, give the component a `Validate` method:

```go
func (h Health) Validate() error {
	if h.Maximum <= 0 || h.Current < 0 || h.Current > h.Maximum {
		return fmt.Errorf("health %d must be between 0 and maximum %d, which must be positive", h.Current, h.Maximum)
	}
	return nil
}
```

Now creating an enemy with a maximum of zero returns an error. Writes through the entity handle, including `Update`, also validate before committing; a rejected update leaves the stored component unchanged. This method belongs to `Health` at package level. The `damageEnemy` function clamps at zero, so its normal damage update remains valid.

Components are non-pointer values, even when they implement validation. A zero value doesn't have to be valid: here `Health{}` is useful as a type marker in a query, but it isn't valid health to put on an entity.

## Add and remove a condition

Some conditions are useful only while they are present. For example, define an empty component to mark a stunned enemy:

```go
type Stunned struct{}
```

Attach it when the enemy is stunned:

```go
if err := enemy.AddComponent(g.World(), Stunned{}); err != nil {
	return fmt.Errorf("stun enemy: %w", err)
}
```

Later, remove it when the stun ends:

```go
if err := enemy.RemoveComponent[Stunned](g.World()); err != nil {
	return fmt.Errorf("end stun: %w", err)
}
```

These are separate moments in the game, not consecutive setup steps. Adding or removing a component changes which queries match the entity. The [movement system](systems.md) uses this marker to leave stunned enemies in place. If you are iterating a query when you decide to change a component set, finish the pass first.

## Remove an enemy

When an enemy should leave the game, destroy it through its world:

```go
if err := g.World().DestroyEntity(enemy); err != nil {
	return fmt.Errorf("remove enemy: %w", err)
}
```

Its components are removed from the world, so it no longer appears in queries or rendering. `DestroyEntity` also marks the **supplied handle** dead. Other handles for the same ID have their own liveness flag; don't use `IsAlive` on another handle as proof that the entity is still in the world. `Kill` only changes a handle's flag and does not remove the entity.

In a real game you usually won't hold a separate variable for every enemy. [Update a group of entities](systems.md) shows how a system finds enemies, changes their state, and removes defeated ones safely.
