# Drive and aim

Our window is empty because the world doesn't contain anything to draw yet. Let's create a tank, then teach it to move. Keep the `main.go` from [setup](setup.md) and run each command from the same project directory.

## Put the tank on screen

The tank has two pieces: a hull that drives and a turret that can aim independently. Each is an **entity**: an identity with components attached. A `Transform` says where a piece is; a `Sprite` says what it looks like.

Create `tank.go` next to `main.go`:

```go
package main

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

const (
	hullImage   = "sprites/T-34/ww2_top_view_hull4.png"
	turretImage = "sprites/T-34/ww2_top_view_turret4.png"
)

type tank struct {
	hull, turret *core.Entity
	finished     bool
}

func addTank(g *castrum.Game) (*tank, error) {
	loader := g.World().MustResource[*asset.Loader]()
	for _, path := range []string{hullImage, turretImage} {
		if _, err := loader.Load[asset.TextureData](path); err != nil {
			return nil, fmt.Errorf("load %s: %w", path, err)
		}
	}

	if _, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{
			Drawable:  render.RectShape{Size: geom.Vector2{X: 160, Y: 160}},
			Color:     color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			SortOrder: -1,
		},
	); err != nil {
		return nil, fmt.Errorf("create starting tile: %w", err)
	}

	hull, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{Drawable: render.TextureSource{Texture: hullImage}},
	)
	if err != nil {
		return nil, fmt.Errorf("create hull: %w", err)
	}
	turret, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{Drawable: render.TextureSource{Texture: turretImage}, Layer: 1},
	)
	if err != nil {
		return nil, fmt.Errorf("create turret: %w", err)
	}
	return &tank{hull: hull, turret: turret}, nil
}
```

In `main.go`, replace `// Game setup goes here.` with:

```go
	player, err := addTank(g)
	if err != nil {
		return err
	}
	_ = player
```

Run `go run .`. You should see the tank centered on a white tile. The tile gives you a fixed landmark: later, when the camera follows the tank, it makes the movement visible.

Both tank pieces start at the world origin. The default camera looks at that origin, so they appear in the middle of the window. The turret uses a higher drawing layer to stay above the hull. We load the images during setup so a wrong path fails before play, rather than when the renderer first needs the image.

The `tank` value keeps the two entity handles together. For now `_ = player` tells Go we're keeping the handle for the next step even though we haven't used it yet. `finished` will let the combat rules stop the tank at the end of the round.

## Move, aim, and follow

A **system** is a piece of behavior the game runs on a schedule. We'll register three fixed-tick systems: one moves the hull, one keeps the turret on the hull and points it at the mouse, and one follows the hull with the camera.

Create `controls.go`:

```go
package main

import (
	"errors"
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

func addControls(g *castrum.Game, player *tank) error {
	if err := g.AddSystem(core.PhaseFixed, "tank.move", core.SystemFunc(func(ctx *core.Context) error {
		if player.finished {
			return nil
		}
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
	})); err != nil {
		return fmt.Errorf("register movement: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, "tank.aim", core.SystemFunc(func(ctx *core.Context) error {
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
	})); err != nil {
		return fmt.Errorf("register aiming: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, "camera.follow", core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("follow: hull has no transform")
		}
		return g.MainCamera().Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position
		})
	})); err != nil {
		return fmt.Errorf("register camera follow: %w", err)
	}
	return nil
}
```

In `main.go`, replace `_ = player` with:

```go
	if err := addControls(g, player); err != nil {
		return err
	}
```

Run `go run .`. Use **W/S** to drive forward and backward, **A/D** to turn, and move the mouse to aim the turret. The camera follows the hull; the tile stays behind as you drive away.

The movement system reads named actions and changes the hull's transform. It multiplies speed by `DeltaTime` so the value is measured per second rather than per tick. The systems are registered in order: aiming and camera follow see the position movement just wrote.

The mouse is in screen coordinates, while the hull lives in the world. The aiming system uses the last rendered camera view to turn the cursor into a world position. Before the first frame there may not be a view yet, so it waits for one rather than treating that normal startup condition as an error. For a closer look at these spaces, see [2D space and the camera](../../concepts/2d-space.md).

The tank can move, but it doesn't have a goal yet. Next: [add shots, enemies, and a way to win or lose](combat.md).
