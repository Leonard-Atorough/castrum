# Adding a simple HUD

The game now counts destroyed enemies, but the player can't see that count. Let's add a small heads-up display, or **HUD**, that shows the score.

## Create the score

Castrum can draw text as a sprite. Create `hud.go`:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

const targetScore = 3

func addHUD(g *castrum.Game, player *tank, progress *round) error {
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return fmt.Errorf("load font: %w", err)
	}
	label, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{Y: 100}},
		render.Sprite{Drawable: render.TextSource{Font: builtin.Font, Text: "score 0 / 3", Size: 20}},
	)
	if err != nil {
		return fmt.Errorf("create score label: %w", err)
	}

	if err := g.AddSystem(core.PhaseFixed, "hud.score", core.SystemFunc(func(ctx *core.Context) error {
		hull, ok := player.hull.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("position score: hull has no transform")
		}
		if err := label.Update(ctx.World, func(t *core.Transform) {
			t.Position = hull.Position.Add(geom.Vector2{Y: 100})
		}); err != nil {
			return fmt.Errorf("position score: %w", err)
		}
		return label.Update(ctx.World, func(s *render.Sprite) {
			s.Drawable = render.TextSource{
				Font: builtin.Font,
				Text: fmt.Sprintf("score %d / %d", progress.score, targetScore),
				Size: 20,
			}
		})
	})); err != nil {
		return fmt.Errorf("register score display: %w", err)
	}
	return nil
}
```

In `main.go`, call it after the `shot.move` registration and before creating the runner:

```go
	if err := addHUD(g, player, progress); err != nil {
		return err
	}
```

Run `go run .` and shoot an enemy. The score below the tank should change from `0 / 3` to `1 / 3`.

`builtin.Register` makes Castrum's built-in font available to the loader. The label is an ordinary world-space sprite: its system puts it below the hull every tick, so it travels with the player and camera. This is a simple way to show feedback, not a screen-fixed UI. See the existing [rendering guide](../../../unreleased/guides/rendering.md) for more about text and overlays.

The score gives the player a goal, but reaching it doesn't end the game yet. Next: [show victory or defeat](victory-or-defeat.md).
