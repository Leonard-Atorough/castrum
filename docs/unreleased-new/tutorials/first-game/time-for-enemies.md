# Time for enemies

The tank can fire, but the shots have nothing to hit. Let's create three opponents and have them chase the player. Then we'll decide what happens when their colliders overlap.

Continue with the game from [controlling the player](controlling-the-player.md).

## Create the enemies

An enemy needs a transform and a sprite, just like the player. We'll also give it a marker so a system can find enemies without moving every sprite in the world.

Create `enemies.go`:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

const (
	enemyImage  = "sprites/Panzer 4/ww2_top_view_hull2.png"
	shotLayer   = 1
	enemyLayer  = 2
	playerLayer = 3
)

type enemy struct{}

type round struct {
	score int
}

func addEnemies(g *castrum.Game, player *tank) error {
	world := g.World()
	if _, err := world.MustResource[*asset.Loader]().Load[asset.TextureData](enemyImage); err != nil {
		return fmt.Errorf("load enemy image: %w", err)
	}

	hullCollider, err := collision.NewCollider(collision.Circle{Radius: 16})
	if err != nil {
		return fmt.Errorf("create hull collider: %w", err)
	}
	hullCollider.Layers = collision.Layers(playerLayer)
	hullCollider.Mask = collision.Mask(enemyLayer)
	if err := player.hull.AddComponent(world, hullCollider); err != nil {
		return fmt.Errorf("attach hull collider: %w", err)
	}

	for _, position := range []geom.Vector2{{Y: -300}, {X: 280, Y: -170}, {X: -280, Y: -170}} {
		collider, err := collision.NewCollider(collision.Circle{Radius: 16})
		if err != nil {
			return fmt.Errorf("create enemy collider: %w", err)
		}
		collider.Layers = collision.Layers(enemyLayer)
		collider.Mask = collision.Mask(shotLayer, playerLayer)
		if _, err := world.NewEntity(
			core.Transform{Position: position},
			render.Sprite{Drawable: render.TextureSource{Texture: enemyImage}},
			collider,
			enemy{},
		); err != nil {
			return fmt.Errorf("create enemy: %w", err)
		}
	}
	return nil
}
```

In `main.go`, add this after the camera system registration:

```go
	if err := addEnemies(g, player); err != nil {
		return err
	}
```

Run `go run .`. You should see enemy tanks around the player. They don't move yet, and shots still pass through them.

The enemy image is loaded during setup, just like the player's images. Each enemy also has a **collider**: a circle used to detect overlaps. Layers say what an entity is; masks say what it can interact with. The hull listens for enemies, and enemies listen for the hull and shots. We'll give shots their own colliders shortly.

## Make the enemies chase

Add `math` to the imports in `enemies.go`, then add this function:

```go
func moveEnemies(world *core.World, player *tank) core.System {
	enemies := core.NewQuery(world).With(core.Transform{}, enemy{})
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("move enemies: hull has no transform")
		}
		for entry := range enemies.Execute() {
			entry.Update(func(t *core.Transform) {
				toward := hull.Position.Sub(t.Position)
				t.Rotation = math.Atan2(toward.X, -toward.Y)
				t.Position = t.Position.Add(toward.Normalize().Mul(35 * ctx.DeltaTime.Seconds()))
			})
		}
		return nil
	})
}
```

In `main.go`, register it immediately after the `addEnemies` call:

```go
	if err := g.AddSystem(core.PhaseFixed, "enemy.move", moveEnemies(g.World(), player)); err != nil {
		return fmt.Errorf("register enemy movement: %w", err)
	}
```

Run the game again. The enemies should turn toward the hull and move closer. Try driving away: they're slower than the player. They don't end the round when they reach you yet.

The query finds entities with both a transform and the `enemy` marker. Each tick, the system moves those entities toward the current hull position. The player and shots are not selected.

## Give shots a collider

Add `github.com/Leonard-Atorough/castrum/collision` to the imports in `shots.go`. Replace `spawnShot` with:

```go
func spawnShot(world *core.World, turret core.Transform) error {
	facing := geom.Vector2{Y: -1}.Rotate(turret.Rotation)
	collider, err := collision.NewCollider(collision.Circle{Radius: 3})
	if err != nil {
		return fmt.Errorf("create shot collider: %w", err)
	}
	collider.Layers = collision.Layers(shotLayer)
	collider.Mask = collision.Mask(enemyLayer)
	if _, err := world.NewEntity(
		core.Transform{Position: turret.Position.Add(facing.Mul(64)), Rotation: turret.Rotation},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 3, Y: 3}},
			Color:    color.NRGBA{R: 245, G: 220, B: 130, A: 255},
		},
		collider,
		shot{},
	); err != nil {
		return fmt.Errorf("create shot: %w", err)
	}
	return nil
}
```

Shots and enemies can now produce contacts. Castrum detects the overlap, but it doesn't decide what a hit means or remove either entity. That's our game's rule.

## Respond to hits

Replace `moveShots` in `shots.go` with this version:

```go
func moveShots(world *core.World, player *tank, progress *round) core.System {
	shots := core.NewQuery(world).With(core.Transform{}, shot{})
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("move shots: hull has no transform")
		}
		var removeShots, hitEnemies []core.EntityID
		for entry := range shots.Execute() {
			contacts, hasContacts := core.NewEntity(entry.ID()).Component[collision.Contacts](ctx.World)
			if hasContacts && len(contacts.Current) > 0 {
				for _, contact := range contacts.Current {
					hitEnemies = append(hitEnemies, contact.Other)
				}
				removeShots = append(removeShots, entry.ID())
				continue
			}
			entry.Update(func(t *core.Transform) {
				facing := geom.Vector2{Y: -1}.Rotate(t.Rotation)
				t.Position = t.Position.Add(facing.Mul(800 * ctx.DeltaTime.Seconds()))
			})
			transform, _ := entry.Component[core.Transform]()
			offset := transform.Position.Sub(hull.Position)
			if math.Hypot(offset.X, offset.Y) > 1200 {
				removeShots = append(removeShots, entry.ID())
			}
		}

		for _, id := range hitEnemies {
			target := core.NewEntity(id)
			if _, ok := target.Component[enemy](ctx.World); !ok {
				continue
			}
			if err := ctx.World.DestroyEntity(target); err != nil {
				return fmt.Errorf("remove enemy: %w", err)
			}
			progress.score++
		}
		for _, id := range removeShots {
			if err := ctx.World.DestroyEntity(core.NewEntity(id)); err != nil {
				return fmt.Errorf("remove shot: %w", err)
			}
		}
		return nil
	})
}
```

In `main.go`, replace the `shot.move` registration with:

```go
	progress := &round{}
	if err := g.AddSystem(core.PhaseFixed, "shot.move", moveShots(g.World(), player, progress)); err != nil {
		return fmt.Errorf("register shot movement: %w", err)
	}
```

Run `go run .`. Each enemy should disappear when a shot hits it. Your score increases too, although we can't see it yet.

The system records contacts and removes the hit entities after the query pass. If two shots hit the same enemy, the second lookup sees that it is already gone, so it isn't counted twice. The existing [collision guide](../../../unreleased/guides/collision.md) explains layers, masks, and contact timing in more depth.

Next: [show the score](adding-a-simple-hud.md).
