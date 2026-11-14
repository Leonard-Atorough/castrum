# Draw and position an image

You can put an image in the world by giving an entity a position and a sprite. The sprite describes what to draw; the engine's renderer does the drawing.

This guide assumes an existing game `g` created with `castrum.New`, a runner that displays it, and a PNG at `assets/player.png`. With the default asset filesystem, run the game from the directory containing `assets`. For a released module, use the [matching documentation](../../../README.md) instead.

## Add an image to the world

Load the image during setup, then create an entity with a transform and a sprite. This helper returns the entity so your movement and camera systems can use it:

```go
func newPlayer(g *castrum.Game) (*core.Entity, error) {
	loader := g.World().MustResource[*asset.Loader]()
	if _, err := loader.Load[asset.TextureData]("assets/player.png"); err != nil {
		return nil, fmt.Errorf("load player image: %w", err)
	}

	player, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{Drawable: render.TextureSource{Texture: "assets/player.png"}},
	)
	if err != nil {
		return nil, fmt.Errorf("create player: %w", err)
	}
	return player, nil
}
```

The helper uses `fmt` and the `castrum`, `asset`, `core`, and `render` packages. Call it during game setup, handle its error, and retain the returned `player` handle if you want to move or follow the entity. The image appears centered in the window when the game runs: the default camera looks at the world origin, which is also the zero-value transform's position.

Loading during setup makes a missing or invalid PNG fail before the game loop starts. `TextureSource` itself stores a path; without the explicit load, the renderer can report a load error when it first collects the sprite. See [Load game assets](../assets/loading.md) for other filesystem roots and the [failure checklist](../assets/troubleshooting.md) if the image is not visible.

## Place and size it

Change the entity's transform to place the image 100 world units to the right of the origin and draw it twice as large:

```go
core.Transform{
	Position: geom.Vector2{X: 100},
	Scale:    geom.Vector2{X: 2, Y: 2},
}
```

Import `github.com/Leonard-Atorough/castrum/geom` and replace `core.Transform{}` in the entity. The image moves right from the window center and doubles in size. Its position is its center, not its upper-left corner. Zero scale components mean a scale of one; rotation is in radians.

Move an entity during play by updating its transform in a system. The [wander example](../../../../examples/wander/main.go) shows fixed-tick movement and a camera together; it uses an embedded sheet region instead of a standalone image. Its module follows the development checkout, so set up the [development workspace](../../../../CONTRIBUTING.md#develop-examples-against-the-checkout) and run `go run .` from `examples/wander`.

## Choose what the camera sees

The world is not limited to the window. A camera frames part of it; moving an image away from the origin may take it out of view. To follow or frame a different area, see [Cameras](cameras.md). For world coordinates, logical pixels, and interpolation, see [2D space and the camera](../../concepts/2d-space.md).
