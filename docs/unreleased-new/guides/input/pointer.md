# Use the mouse

Use the raw input snapshot for pointer data, such as a cursor position or scroll offset. For a named button action that can also have a keyboard or controller binding, use [player actions](actions.md) instead.

## Pick a position in the world

The default runner reports the cursor in logical-screen coordinates. Convert it through the last rendered camera view when you want to aim at or select something in the world:

```go
func cursorInWorld(ctx *core.Context) (geom.Vector2, bool, error) {
	view, err := ctx.World.Resource[*render.CameraView]()
	if errors.Is(err, core.ErrResourceNotRegistered) {
		return geom.Vector2{}, false, nil
	}
	if err != nil {
		return geom.Vector2{}, false, fmt.Errorf("get camera view: %w", err)
	}
	return view.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight), true, nil
}
```

The helper uses `errors`, `fmt`, and the `core`, `geom`, and `render` packages. Call it from a system, handle the error, and use the position only when the boolean is true. Before the renderer has produced its first view, the resource may not exist; skipping that update is normal.

The view represents the most recently rendered camera, including its framing, so the picked position matches what the player saw. It can differ from a camera transform changed since that draw. [2D space and the camera](../../concepts/2d-space.md) explains the coordinate conversion; the [raycast example](../../../../examples/raycast) uses the result to aim a line through the world.

## Read the wheel once per frame

In a frame-phase system, `ctx.Input.Wheel()` gives the current runner update's signed scroll offset. Use the offset as a displacement, not as a held state or a rate to multiply by `DeltaTime`; it can be fractional and resets when there is no scrolling. The [input example](../../../../examples/input) uses it to zoom.

Read wheel movement in a frame phase rather than applying it once per fixed tick. Catch-up ticks share the same frame snapshot, while an update with no fixed tick can be replaced before a tick reads it. `ctx.Input` is runner-owned and read-only; snapshot reads do not consume the value.
