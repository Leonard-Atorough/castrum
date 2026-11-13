# Make a playable round

The tank can drive and aim. Now let's give it something to do: fire at three approaching tanks before one reaches it.

We'll keep this first round deliberately small. The enemies start at known positions, each shot destroys one enemy, and a message reports whether you won or lost. The [finished example](../../../../examples/your-first-game/main.go) builds on the same idea with waves and a reload timer.

## Set up the fight

Create `combat.go` with the following code. This sets up the round and registers its systems. We'll add those systems to the same file in the next sections; wait until the end of the chapter to run the game.

```go
package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
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
	targetScore = 3
)

type shot struct{}
type enemy struct{}

type fight struct {
	player *tank
	score  int
	won    bool
	status *core.Entity
}

func addFight(g *castrum.Game, player *tank) error {
	world := g.World()
	loader := world.MustResource[*asset.Loader]()
	if _, err := loader.Load[asset.TextureData](enemyImage); err != nil {
		return fmt.Errorf("load enemy image: %w", err)
	}
	if err := builtin.Register(loader); err != nil {
		return fmt.Errorf("load font: %w", err)
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

	status, err := world.NewEntity(
		core.Transform{Position: geom.Vector2{Y: 100}},
		render.Sprite{Drawable: render.TextSource{Font: builtin.Font, Text: "score 0 / 3", Size: 20}},
	)
	if err != nil {
		return fmt.Errorf("create status: %w", err)
	}
	f := &fight{player: player, status: status}
	shots := core.NewQuery(world).With(core.Transform{}, shot{})
	enemies := core.NewQuery(world).With(core.Transform{}, enemy{})

	systems := []struct {
		name string
		run  core.System
	}{
		{"fight.fire", core.SystemFunc(f.fire)},
		{"fight.shots", core.SystemFunc(func(ctx *core.Context) error { return f.moveShots(ctx, shots) })},
		{"fight.enemies", core.SystemFunc(func(ctx *core.Context) error { return f.moveEnemies(ctx, enemies) })},
		{"fight.danger", core.SystemFunc(f.checkDanger)},
		{"fight.status", core.SystemFunc(f.showStatus)},
	}
	for _, system := range systems {
		if err := g.AddSystem(core.PhaseFixed, system.name, system.run); err != nil {
			return fmt.Errorf("register %s: %w", system.name, err)
		}
	}
	return nil
}
```

`addFight` loads the enemy image and a built-in font, attaches a collider to the hull, and creates the enemies and status text. Each enemy has a transform, a sprite, a collider, and an `enemy` marker. The marker is just data: it lets a query find enemies without also moving the tank or its shots.

A collider describes an area that can overlap another collider. Layers and masks choose which pairs can interact: shots look for enemies, the hull looks for enemies, and enemies accept both. Castrum's collision system records contacts; it doesn't move the tanks or decide what a hit means. Those are our game rules. The existing [collision guide](../../../unreleased/guides/collision.md) goes into the details.

Our `fight` value holds this round's score and result, while the world holds the entities and components. For a larger game you might represent shared game state differently; for this round, keeping it with the systems that use it makes the rules easy to follow.

## Fire a shot

Append this method to `combat.go`:

```go
func (f *fight) fire(ctx *core.Context) error {
	if f.player.finished || !ctx.Actions.Pressed("fire") {
		return nil
	}
	turret, ok := f.player.turret.Component[core.Transform](ctx.World)
	if !ok {
		return fmt.Errorf("fire: turret has no transform")
	}
	facing := geom.Vector2{Y: -1}.Rotate(turret.Rotation)
	collider, err := collision.NewCollider(collision.Circle{Radius: 3})
	if err != nil {
		return fmt.Errorf("create shot collider: %w", err)
	}
	collider.Layers = collision.Layers(shotLayer)
	collider.Mask = collision.Mask(enemyLayer)
	if _, err := ctx.World.NewEntity(
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

The fire system reads `Pressed` on a fixed tick. That action preserves a press for the next tick, so a quick click between ticks can still fire. The shot starts at the end of the turret and keeps the direction it had when it was created.

## Move shots and count hits

The next system finds all shots, moves them, and checks their contacts. Add it below `fire`:

```go
func (f *fight) moveShots(ctx *core.Context, shots *core.Query) error {
	if f.player.finished {
		return nil
	}
	hull, ok := f.player.hull.Component[core.Transform](ctx.World)
	if !ok {
		return fmt.Errorf("move shots: hull has no transform")
	}
	var removeShots, hitEnemies []core.EntityID
	for entry := range shots.Execute() {
		handle := core.NewEntity(entry.ID())
		contacts, hasContacts := handle.Component[collision.Contacts](ctx.World)
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
		f.score++
	}
	for _, id := range removeShots {
		if err := ctx.World.DestroyEntity(core.NewEntity(id)); err != nil {
			return fmt.Errorf("remove shot: %w", err)
		}
	}
	if f.score >= targetScore {
		f.won = true
		f.player.finished = true
	}
	return nil
}
```

`moveShots` collects the IDs of shots and enemies to remove, then destroys them **after** the query pass. Creating or destroying entities while iterating a query would invalidate the iteration. Distant shots are removed too, so missed shots don't stay in the world forever. If two shots hit the same enemy, the second removal sees that the enemy is already gone and doesn't count it twice.

## Make enemies chase the player

Append this system to point each enemy toward the hull and move it closer:

```go
func (f *fight) moveEnemies(ctx *core.Context, enemies *core.Query) error {
	if f.player.finished {
		return nil
	}
	hull, ok := f.player.hull.Component[core.Transform](ctx.World)
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
}
```

The query selects entities with both a transform and the `enemy` marker. Updating each matching transform changes the world state that the renderer will draw.

## End the round and show the result

These last two methods check whether an enemy reached the hull and update the status text. Add them below `moveEnemies`:

```go
func (f *fight) checkDanger(ctx *core.Context) error {
	if f.player.finished {
		return nil
	}
	contacts, ok := f.player.hull.Component[collision.Contacts](ctx.World)
	if ok && len(contacts.Current) > 0 {
		f.player.finished = true
	}
	return nil
}

func (f *fight) showStatus(ctx *core.Context) error {
	hull, ok := f.player.hull.Component[core.Transform](ctx.World)
	if !ok {
		return fmt.Errorf("show status: hull has no transform")
	}
	if err := f.status.Update(ctx.World, func(t *core.Transform) {
		t.Position = hull.Position.Add(geom.Vector2{Y: 100})
	}); err != nil {
		return fmt.Errorf("position status: %w", err)
	}
	text := fmt.Sprintf("score %d / %d", f.score, targetScore)
	if f.player.finished {
		text = "GAME OVER"
		if f.won {
			text = "YOU WIN"
		}
	}
	return f.status.Update(ctx.World, func(s *render.Sprite) {
		s.Drawable = render.TextSource{Font: builtin.Font, Text: text, Size: 20}
	})
}
```

The shot system runs before the danger system, so destroying the last enemy and winning takes precedence over a danger contact in the same tick. The status follows the hull and changes its text when the round ends.

## Play

In `main.go`, add this immediately after the `addControls` call and before creating the runner:

```go
	if err := addFight(g, player); err != nil {
		return err
	}
```

Run `go run .`. Drive and aim as before; click the left mouse button to fire. The score appears below your tank. Destroy all three enemies for **YOU WIN**, or let one touch the hull for **GAME OVER**. Close and run the game again to play another round.

Next: [add sound and explore where to go from here](sound.md).
