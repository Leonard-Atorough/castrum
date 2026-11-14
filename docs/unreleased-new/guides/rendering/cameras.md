# Cameras

Think of the camera as a window into your game world. You can move it to follow a character, zoom out to show more of a level, or keep it inside the edges of a map. None of that moves the objects in the world; it changes how you see them.

Every Castrum game starts with a camera at the world origin. Its position, rotation, and zoom live in a `core.Transform`, just like the transform of a visible entity. The `render.Camera` component holds the framing choices that are specific to cameras.

The examples below use a game `g` created by `castrum.New`. They belong in a function that returns an error. If you have created a sprite as shown in [Draw and position an image](images.md), use its returned entity handle to try the follow example.

## Move and zoom the default camera

Let's move the camera 100 world units to the right and zoom in:

```go
if err := g.MainCamera().Update(g.World(), func(t *core.Transform) {
	t.Position = geom.Vector2{X: 100}
	t.Scale = geom.Vector2{X: 2, Y: 2}
}); err != nil {
	return fmt.Errorf("frame camera: %w", err)
}
```

Add the `fmt` and `github.com/Leonard-Atorough/castrum/geom` imports if they are not already present. If you also moved the image from the earlier guide to `X: 100`, it stays in the center of the window as the camera moves. At zoom two, it appears twice as large as it did before you changed the camera.

A zoom above one brings things closer; a zoom between zero and one shows more of the world. Set both scale axes to the same positive value. A zero axis is treated as one, but negative, non-finite, or unequal effective axes cause camera resolution to fail when the renderer collects the frame.

## Follow a player

A camera can move during the game just like another entity. This system reads a player's position and puts the camera there:

```go
func followPlayer(camera, player *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		target, ok := player.Component[core.Transform](ctx.World)
		if !ok {
			return fmt.Errorf("follow player: player has no transform")
		}
		return camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = target.Position
		})
	})
}
```

Register it after the system that moves the player, in the same fixed phase. This example assumes that system has already been registered as `"player.move"`:

```go
if err := g.AddSystem(core.PhaseFixed, "camera.follow", followPlayer(g.MainCamera(), player), castrum.After("player.move")); err != nil {
	return fmt.Errorf("register camera follow: %w", err)
}
```

`castrum.After` inserts the follow system immediately after the named, already registered user system; it is not a permanent dependency. If you later insert another system after `"player.move"`, that new system goes closest to it. Here `player` is the handle returned when you created the player's entity. The renderer interpolates camera movement between fixed ticks, so the view can move smoothly even when the display draws more often than the simulation updates. If you use camera bounds, the displayed position can be clamped even though the system continues to copy the player's position.

## Choose the position's anchor

The camera's position normally sits at the center of the screen. For some games, it is more convenient to place the world origin at the top-left instead. Change the camera's anchor to do that:

```go
if err := g.MainCamera().Update(g.World(), func(camera *render.Camera) {
	camera.Anchor = geom.Vector2{X: -0.5, Y: -0.5}
}); err != nil {
	return fmt.Errorf("set camera anchor: %w", err)
}
```

With an unrotated camera at the world origin and zoom one, the origin is now at the top-left instead of the center. If you tried the earlier movement example, reset the camera's position to `geom.Vector2{}` and its scale to `geom.Vector2{}` (zero scale means one) to see this mapping. Anchor coordinates are fractions of the logical view relative to its center; each axis must be between `-0.5` and `0.5`.

## Keep the view within a level

Suppose your level occupies a rectangle from `(0, 0)` to `(1600, 900)`. You can stop the view from showing space beyond it:

```go
if err := g.MainCamera().Update(g.World(), func(camera *render.Camera) {
	camera.Bounds = geom.RectFromMinMax(
		geom.Vector2{},
		geom.Vector2{X: 1600, Y: 900},
	)
}); err != nil {
	return fmt.Errorf("set camera bounds: %w", err)
}
```

The renderer clamps the resolved view after interpolation. If the view is larger than the level along an axis, it centers that axis instead. This changes the displayed view, not necessarily the camera entity's stored position.

If your movement controls also need to respect bounds, use `Camera.MovePosition` with the current transform, world-space displacement, and logical width and height. It returns a constrained position or an error; it does not modify the transform. The [camera example](../../../../examples/camera/main.go) demonstrates this with a moving camera. From `examples/camera` in the [development workspace](../../../../CONTRIBUTING.md#develop-examples-against-the-checkout), run `go run .`: arrow keys pan, A/D rotate, and W/S change zoom.

## Logical size and window size

The default runner renders to a logical canvas and scales it to the window. Both default to 1280 by 720 pixels, but can be configured independently with `ebitrun.WithLogicalSize` and `ebitrun.WithWindowSize` when constructing the runner. Camera framing uses the logical size, not the physical window size.

For the coordinate model and how to convert positions using the last rendered view, see [2D space and the camera](../../concepts/2d-space.md).
