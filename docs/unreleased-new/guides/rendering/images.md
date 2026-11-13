# Draw and position an image

You can put an image in the world by giving an entity a position and a sprite. The sprite describes what to draw; the engine's renderer does the drawing.

This guide uses the development API and the default Ebitengine runner. Start with a Go project that uses the Castrum development checkout and contains a PNG image at `assets/player.png`. Run the program from the project directory so the default asset filesystem can find that path. For a released module, use the [matching documentation](../../../README.md) instead.

## Create the game and sprite

Put this complete program in `main.go`. It loads the image during setup, creates an entity at the world origin, and starts the runner:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/render"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("My game"))
	if err != nil {
		return fmt.Errorf("create game: %w", err)
	}

	loader := g.World().MustResource[*asset.Loader]()
	if _, err := loader.Load[asset.TextureData]("assets/player.png"); err != nil {
		return fmt.Errorf("load player image: %w", err)
	}

	if _, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{Drawable: render.TextureSource{Texture: "assets/player.png"}},
	); err != nil {
		return fmt.Errorf("create player: %w", err)
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}
	return g.Run(runner)
}
```

Run `go run .` from the project directory. The image should appear centered in the window. Close the window to stop the program. The default camera looks at the world origin, which is also the zero-value transform's position.

Loading during setup makes a missing or invalid PNG fail before the game loop starts. `TextureSource` itself stores a path; without the explicit load, the renderer can report a load error when it first collects the sprite. See [Load game assets](../assets/loading.md) for other filesystem roots and the [failure checklist](../assets/troubleshooting.md) if the image is not visible.

## Place and size it

Change the entity's transform to place the image 100 world units to the right of the origin and draw it twice as large:

```go
core.Transform{
	Position: geom.Vector2{X: 100},
	Scale:    geom.Vector2{X: 2, Y: 2},
}
```

Add `"github.com/Leonard-Atorough/castrum/geom"` to the program's imports and replace `core.Transform{}` in the entity. The image moves right from the window center and doubles in size. Its position is its center, not its upper-left corner. Zero scale components mean a scale of one; rotation is in radians.

Move an entity during play by updating its transform in a system. The [wander example](../../../../examples/wander/main.go) shows fixed-tick movement and a camera together; it uses an embedded sheet region instead of a standalone image. Its module follows the development checkout, so set up the [development workspace](../../../../CONTRIBUTING.md#develop-examples-against-the-checkout) and run `go run .` from `examples/wander`.

## Choose what the camera sees

The world is not limited to the window. A camera frames part of it; moving an image away from the origin may take it out of view. To follow or frame a different area, see [Cameras](cameras.md). For world coordinates, logical pixels, and interpolation, see [2D space and the camera](../../concepts/2d-space.md).
