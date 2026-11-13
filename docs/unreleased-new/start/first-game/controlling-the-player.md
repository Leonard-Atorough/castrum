# Controlling the player

The tank is on screen, but it doesn't respond yet. Let's give it controls, make the camera follow it, and fire a shot. Keep the files from [defining the player](defining-the-player.md).

## Name the controls

In `main.go`, add `github.com/Leonard-Atorough/castrum/input` and `github.com/Leonard-Atorough/castrum/core` to the imports. Then replace the `castrum.New` call with:

```go
	bindings := input.Bindings{
		"forward":  []input.Input{input.KeyInput{Key: input.KeyW}.Binding()},
		"backward": []input.Input{input.KeyInput{Key: input.KeyS}.Binding()},
		"left":     []input.Input{input.KeyInput{Key: input.KeyA}.Binding()},
		"right":    []input.Input{input.KeyInput{Key: input.KeyD}.Binding()},
		"fire":     []input.Input{input.MouseButtonInput{Button: input.MouseButtonLeft}.Binding()},
	}
	g, err := castrum.New(castrum.WithTitle("Tank game"), castrum.WithBindings(bindings))
```

Keep the existing error check immediately after it. These names are **actions**: the game's rules can ask for `"forward"` without needing to know which physical key was pressed. The existing [input guide](../../../unreleased/guides/input.md) covers other controls and rebinding.

## Drive the hull

A **system** is a piece of behavior the game runs on a schedule. Our first system reads the movement actions and updates the hull's transform.

Create `controls.go`:

```go
package main

import (
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func movePlayer(player *tank) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return player.hull.Update(ctx.World, func(t *core.Transform) {
			facing := geom.Vector2{Y: -1}.Rotate(t.Rotation)
			if ctx.Actions.Held("forward") {
				t.Position = t.Position.Add(facing.Mul(100 * dt))
			}
			if ctx.Actions.Held("backward") {
				t.Position = t.Position.Sub(facing.Mul(100 * dt))
			}
			if ctx.Actions.Held("left") {
				t.Rotation -= 3 * dt
			}
			if ctx.Actions.Held("right") {
				t.Rotation += 3 * dt
			}
		})
	})
}
```

We need the handle returned by `addTank` now. In `main.go`, replace the existing `addTank` block with:

```go
	player, err := addTank(g)
	if err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "tank.move", movePlayer(player)); err != nil {
		return fmt.Errorf("register movement: %w", err)
	}
```

Run `go run .`. **W/S** should move the hull forward and backward; **A/D** should turn it. The turret stays behind for now—we'll fix that next.

This system runs on fixed simulation ticks. Multiplying by `DeltaTime` makes the speed a distance per second rather than a distance per tick. The hull's initial facing is up, along negative Y; rotating that vector gives us its current forward direction.

## Aim the turret

We want the turret to stay on the hull and point toward the mouse. The mouse position is on screen, while the hull position is in the world. The camera's last rendered view lets us convert between them.

Add `errors`, `fmt`, `math`, and `github.com/Leonard-Atorough/castrum/render` to the imports in `controls.go`. Then add this function below `movePlayer`:

```go
func aimTurret(player *tank) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("aim: hull has no transform")
		}
		view, err := ctx.World.Resource[*render.CameraView]()
		if errors.Is(err, core.ErrResourceNotRegistered) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("aim: get camera view: %w", err)
		}
		cursor := view.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		aim := cursor.Sub(hull.Position)
		return player.turret.Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position
			t.Rotation = math.Atan2(aim.X, -aim.Y)
		})
	})
}
```

The turret needs the hull's new position each tick, so movement must run before aiming. In `main.go`, replace the `tank.move` registration with these two registrations:

```go
	if err := g.AddSystem(core.PhaseFixed, "tank.aim", aimTurret(player)); err != nil {
		return fmt.Errorf("register aiming: %w", err)
	}
	if err := g.AddSystem(core.PhaseFixed, "tank.move", movePlayer(player), castrum.Before("tank.aim")); err != nil {
		return fmt.Errorf("register movement: %w", err)
	}
```

We register aiming first so it exists as a target, then `castrum.Before("tank.aim")` places movement immediately before it. The target must already be registered in the same phase. Run the game again: the turret should stay on the hull and follow your mouse while you drive.

Before the first frame is drawn, there may not be a camera view yet. The system waits for one rather than treating that normal startup condition as an error. For a closer look at screen and world coordinates, see [2D space and the camera](../../concepts/2d-space.md).

## Follow the tank

The tank can now drive off screen. Add this function to `controls.go`:

```go
func followPlayer(camera *core.Entity, player *tank) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("follow: hull has no transform")
		}
		return camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position
		})
	})
}
```

Register it in `main.go` after the other two systems:

```go
	if err := g.AddSystem(core.PhaseFixed, "camera.follow", followPlayer(g.MainCamera(), player), castrum.After("tank.aim")); err != nil {
		return fmt.Errorf("register camera follow: %w", err)
	}
```

`castrum.After` works the same way as `Before`, placing the new system immediately after an already registered target in the same phase. Here the camera follows after movement and aiming. Run the game and drive away from the tile: the tank stays centered while the tile moves out of view. The camera changes what you see, not the position of the tile. The [camera guide](../../guides/rendering/cameras.md) shows other framing choices.

These options insert systems at registration time; they don't establish permanent dependencies. If you add more systems later, check how their insertions affect the order. The existing [scheduling guide](../../../unreleased/guides/system-scheduling.md) goes into more detail.

## Fire a shot

We'll create a small circle at the end of the turret. Create `shots.go`:

```go
package main

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

type shot struct{}

func fireShot(player *tank) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Actions.Pressed("fire") {
			return nil
		}
		turret, ok := player.turret.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("fire: turret has no transform")
		}
		return spawnShot(ctx.World, turret)
	})
}

func spawnShot(world *core.World, turret core.Transform) error {
	facing := geom.Vector2{Y: -1}.Rotate(turret.Rotation)
	if _, err := world.NewEntity(
		core.Transform{Position: turret.Position.Add(facing.Mul(64)), Rotation: turret.Rotation},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: 3, Y: 3}},
			Color:    color.NRGBA{R: 245, G: 220, B: 130, A: 255},
		},
		shot{},
	); err != nil {
		return fmt.Errorf("create shot: %w", err)
	}
	return nil
}
```

Register the firing system in `main.go` after `camera.follow`:

```go
	if err := g.AddSystem(core.PhaseFixed, "shot.fire", fireShot(player), castrum.After("camera.follow")); err != nil {
		return fmt.Errorf("register firing: %w", err)
	}
```

Run the game and click the left mouse button. A small circle appears at the end of the turret. It doesn't travel yet, so each shot stays where it was created.

The fire system uses `Pressed`, a tick-stable press, so a quick click between fixed ticks can still fire. The `shot` component marks the new entity; we'll use that marker to find it next.

## Move the shots

Add `math` to the imports in `shots.go`, then append this function:

```go
func moveShots(world *core.World, player *tank) core.System {
	shots := core.NewQuery(world).With(core.Transform{}, shot{})
	return core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("move shots: hull has no transform")
		}
		var remove []core.EntityID
		for entry := range shots.Execute() {
			entry.Update(func(t *core.Transform) {
				facing := geom.Vector2{Y: -1}.Rotate(t.Rotation)
				t.Position = t.Position.Add(facing.Mul(800 * ctx.DeltaTime.Seconds()))
			})
			transform, _ := entry.Component[core.Transform]()
			offset := transform.Position.Sub(hull.Position)
			if math.Hypot(offset.X, offset.Y) > 1200 {
				remove = append(remove, entry.ID())
			}
		}
		for _, id := range remove {
			if err := ctx.World.DestroyEntity(core.NewEntity(id)); err != nil {
				return fmt.Errorf("remove shot: %w", err)
			}
		}
		return nil
	})
}
```

Register it in `main.go` after `shot.fire`:

```go
	if err := g.AddSystem(core.PhaseFixed, "shot.move", moveShots(g.World(), player), castrum.After("shot.fire")); err != nil {
		return fmt.Errorf("register shot movement: %w", err)
	}
```

Run the game and click the left mouse button. A shot should fly out of the turret. Aim elsewhere and fire again: each shot keeps its original direction.

Firing runs after the turret has aimed, and shot movement runs after firing, so a newly created shot can move on the same tick. The `shot` component marks the entities that `moveShots` should find. Its query selects shots with transforms, leaving the tank and tile alone. We remove distant shots after the query finishes; destroying entities during iteration would invalidate the query.

The shots don't hit anything yet. Next, in [time for enemies](time-for-enemies.md), we'll give them something to hit.
