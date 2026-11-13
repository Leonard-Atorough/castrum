# Defining the player

Our window is empty because the world doesn't contain anything to draw yet. Let's put a tank on screen. Keep the project from [setting up](setting-up.md), and continue running commands from its directory.

The tank has two pieces: a hull and a turret. Making them separate entities will let the hull drive one way while the turret aims another.

## Create the tank

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
	if _, err := addTank(g); err != nil {
		return err
	}
```

Run `go run .`. You should see the tank centered on a white tile. If an image cannot be found, check that the `sprites/` directory is in your project and that you are running the command from that directory.

## What makes the tank visible?

Each call to `NewEntity` gives the world an entity with components. A `Transform` describes a position, rotation, and scale. A `Sprite` describes what to draw there. The hull and turret start at the world origin, and the default camera looks at that origin, so both appear in the middle of the window.

The turret uses a higher drawing layer to stay above the hull. The white tile is another entity, drawn underneath the tank. It gives you a fixed landmark: once the camera follows the tank, you can see the tile move away.

Before creating the entities, we load the images by their paths. Loading during setup means a missing or invalid file fails before play, rather than when the renderer first needs it. See [Load game assets](../../guides/assets/loading.md) for more about paths and loading, and [Draw and position an image](../../guides/rendering/images.md) for more about sprites.

The `tank` value keeps handles to the hull and turret together. We haven't used those handles yet, but we'll need them to move the two pieces independently.

Next: [control the player](controlling-the-player.md).
