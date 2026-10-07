# Rendering

Rendering is state-driven. Attach a `Transform` and a `Sprite` to an entity, give the sprite a drawable, and the runner turns the world into pixels. You do not issue world draw calls from gameplay systems.

The rendering path is:

```text
Transform + Sprite -> collect -> resolve -> cull -> sort -> draw
```

The [primitive example](../../examples/primitive) shows the shape drawables, while [examples/animate](../../examples/animate) shows an atlas-backed sprite. The [assets guide](assets.md) covers loading textures and registering atlases.

## Draw a sprite

Every rendered entity needs a `Transform` and a `Sprite`. A `Transform` supplies the world position, rotation, and scale. A `Sprite` supplies the picture and its visual style:

```go
_, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 100}},
	core.Sprite{Drawable: core.TextureSource{Texture: "sprites/hull.png"}},
)
if err != nil {
	return err
}
```

The engine adds and maintains the previous-transform state needed for interpolation when a `Transform` enters the world. You do not create or update `core.PrevTransform` yourself. An entity without a `Transform` does not enter the renderer's sprite query.

The zero `Sprite` is valid but invisible: a nil `Drawable` declares style without a picture. This is useful when another system will attach the drawable later.

## Choose a drawable

`Sprite.Drawable` is a sealed set of five current choices:

- `AtlasSource` draws one named region from a registered atlas.
- `TextureSource` draws a standalone texture at its full size.
- `RectShape` draws a rectangle in world units.
- `CircleShape` draws a circle or ellipse in world units.
- `LineShape` draws a segment between two points relative to the entity position.

### Atlas regions

Register an atlas before creating sprites that depend on it, then point the sprite at a named region:

```go
server := g.AssetServer()
if err := server.RegisterGridAtlas("characters", "char.png", 16, 16, "char"); err != nil {
	return err
}

_, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 100}},
	core.Sprite{Drawable: core.AtlasSource{
		Atlas:  "characters",
		Region: "char_0",
	}},
)
if err != nil {
	return err
}
```

An `AtlasSource` stores names, not image data. The collector resolves the atlas and region when it builds a frame. An unknown atlas or region therefore reports an error at the first rendered frame, not when the `Sprite` component is validated. Register atlases during setup when you want those failures near the declaration. See [Assets](assets.md) for grid and sidecar atlases.

### Standalone textures

Use `TextureSource` when the entity should draw the entire image rather than a named atlas region:

```go
core.Sprite{Drawable: core.TextureSource{Texture: "sprites/player.png"}}
```

The texture is loaded through the asset server when the collector resolves it. Preload it during setup if a missing or malformed image should prevent the game from starting:

```go
if _, err := g.AssetServer().Load[asset.TextureData]("sprites/player.png"); err != nil {
	return err
}
```

### Shapes

Shapes do not use assets:

```go
core.Sprite{
	Drawable: core.RectShape{Size: geom.Vector2{X: 120, Y: 60}},
	Color:    color.RGBA{G: 200, A: 255},
}
```

`RectShape.Size` is centered on the entity position. `CircleShape.Radii` describes the radii; unequal radii make an ellipse. `LineShape.From` and `LineShape.To` are offsets from the entity position, and the endpoints must differ. Shapes are measured in world units and are affected by the transform's scale and rotation.

Shapes can be filled or outlined. Set `Outline: true` and provide a positive `StrokeWidth`. A line always draws as a stroke and uses `StrokeWidth`. A nil `Color` defaults shapes to black; for texture sources, nil leaves the source's pixels uncolored and opaque.

## Transform sprites

`Transform` controls the spatial part of every drawable:

- `Position` places the drawable in world space.
- `Rotation` rotates it in radians.
- `Scale` scales it on each axis. A zero axis means unscaled, so the usual zero value is useful.

The renderer centers texture sprites, rectangles, and circles on `Position`. A line's position is its declared endpoint origin plus its `From` and `To` offsets. Texture sprites and shapes can be scaled independently on each axis; only texture sources respond to `FlipH` and `FlipV`.

Move entities in fixed-phase systems. The engine captures the previous position before user fixed systems run, then interpolates from that position to the current one for display. This keeps motion smooth when the display rate and fixed simulation rate differ:

```go
if err := g.AddSystem(core.PhaseFixed, "player.move", core.SystemFunc(func(ctx *core.Context) error {
	return player.Update(ctx.World, func(t *core.Transform) {
		t.Position.X += 2
	})
})); err != nil {
	return err
}
```

## Style and draw order

The `Sprite` fields control presentation without changing the drawable:

- `Layer` is the broad draw band, from 0 through 31. Higher layers draw later.
- `SortOrder` orders sprites within a layer. Higher values draw later.
- World Y is the final ordering fallback within equal layer and sort order.
- `Hidden` skips collection without destroying or changing the entity.
- `FlipH` and `FlipV` mirror texture sources; they do not affect shapes.
- `Color` multiplies texture pixels or supplies the shape's fill and stroke color. Its alpha controls opacity.
- `Outline` and `StrokeWidth` apply to shapes.

The sort keys are applied in that order: layer, sort order, then interpolated world Y. Use layers for stable bands such as background, actors, and foreground. Use sort order for explicit ordering inside one band, and let world Y provide the natural 2D fallback when objects share both.

## The camera

Every game starts with an engine-owned primary camera at the world origin with zoom 1. Access it with `g.MainCamera()`:

```go
camera := g.MainCamera()
if err := camera.Update(g.World(), func(t *core.Transform) {
	t.Position = geom.Vector2{X: 320, Y: 180}
}); err != nil {
	return err
}
if err := camera.Update(g.World(), func(c *core.Camera) {
	c.Zoom = 2
}); err != nil {
	return err
}
```

`Camera.Zoom` must be positive. Zoom 1 leaves world scale unchanged; larger values zoom in and smaller positive values zoom out. The camera's `Transform` moves the view through the world.

When camera state belongs to a game entity, spawn another entity with `Camera` and `Transform`, and mark the camera `Primary`:

```go
_, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 320, Y: 180}},
	core.Camera{Primary: true, Zoom: 1.5},
)
if err != nil {
	return err
}
```

A user-spawned primary takes precedence over the engine camera. Keep one user primary active so camera selection stays intentional. When game code needs to connect screen-space input to world-space entities, build a `core.CameraView` from the camera entity's current `Transform.Position` and `Camera.Zoom`, then use its `WorldToScreen` or `ScreenToWorld` method. The [input example](../../examples/input) shows this pattern.

## Overlays

Use a runner draw callback for HUDs, debug text, and other screen-space content that should not live in the world. Overlays are the imperative exception to the state-driven world:

```go
runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
	ebitenutil.DebugPrint(screen, "score: 12")
	return nil
})
```

The runner draws the world first, then runs user draw callbacks once per display frame in registration order. Later callbacks draw on top of earlier ones. The callback receives the backend screen, so it is runner-specific and not portable across rendering backends. Return an error so `Run` can surface it.

## Culling and render failures

The collector culls drawable bounds against the active camera viewport before sorting. Off-screen sprites still cost entity matching and collection, but they are not sent to the backend for drawing. Culling uses the drawable's bounds and does not expand them for a rotated sprite's full swept area, so a rotating object can appear at the edge as its unrotated bounds enter the view.

If no primary camera exists, the world draw list is empty and overlays still run. The engine camera normally prevents that case.

Rendering errors usually mean a reference could not be resolved: a texture could not load, or an atlas or region was not registered. Return errors from setup when you preload or register assets, and keep the first-frame resolution behavior in mind for assets you load lazily.

## Where to go next

- [Assets](assets.md) - filesystem roots, texture loads, and atlas registration.
- [Animation](animation.md) - change atlas regions over time with animation clips.
- [Input](input.md) - screen-space input and frame snapshots.
- [The coordinate system](../reference/coordinate-system.md) - world, screen, zoom, and interpolation details.
- [examples/primitive](../../examples/primitive) - a runnable shape-rendering example.
