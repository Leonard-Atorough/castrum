# Camera

A Castrum camera combines two components:

- `render.Camera` describes the framing policy: which camera is primary,
  where its position maps in the viewport, and which world rectangle limits
  the view.
- `core.Transform` supplies the camera's position, rotation, and zoom. Zoom
  is the transform's uniform scale.

The renderer interpolates the transform, applies the camera policy, and
publishes the resulting `*render.CameraView` for the frame. Game code updates
the transform and reads the published view; it does not construct the
rendered view itself.

The [camera example](../../examples/camera/main.go) demonstrates panning,
rotation, zoom, and bounds:

```text
go run ./examples/camera
```

## Create a camera

The engine always provides a default camera at the world origin, with zoom 1
and a centered anchor. Spawn a user camera when the game needs to control the
framing:

```go
bounds := geom.RectFromMinMax(
	geom.Vector2{},
	geom.Vector2{X: 1600, Y: 900},
)
camera, err := render.SpawnCamera(
	g.World(),
	render.Camera{Primary: true, Bounds: bounds},
	bounds.Center(),
	1,
)
if err != nil {
	return err
}
```

`SpawnCamera` creates the entity, stores the starting position in its
`core.Transform`, and uses the zoom to initialize the transform's uniform
scale. A zoom of zero means 1. A user camera must set `Primary: true` to
render; otherwise the engine camera remains active. If multiple user cameras
are primary, the collector uses the first one in its deterministic query
order.

Setup validation happens during `SpawnCamera`. An anchor outside the
`[-0.5, 0.5]` range or a negative zoom returns an error immediately.

## Move and zoom

A camera moves like any other entity. Update its `core.Transform` from a
system, usually in the fixed phase, and let the renderer interpolate the
transform between ticks:

```go
func cameraSystem(camera, target *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		targetTransform, _ := target.Component[core.Transform](ctx.World)
		return camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = targetTransform.Position
		})
	})
}
```

Zoom is the transform's uniform scale. A value of 1 is the default view;
values above 1 zoom in, and values between 0 and 1 zoom out. Update both
scale components together:

```go
if err := camera.Update(ctx.World, func(t *core.Transform) {
	zoom := math.Max(zoomMin, math.Min(zoomMax, t.Scale.X+rate*dt))
	t.Scale = geom.Vector2{X: zoom, Y: zoom}
}); err != nil {
	return err
}
```

The renderer requires a positive, uniform scale when it resolves a camera.
Negative or non-uniform scale is invalid framing and produces an error during
collection. Zero scale components are treated as zoom 1. A camera view
cannot be stretched.

## Choose an anchor

`Camera.Anchor` determines where the camera position appears in the viewport:

- `(0, 0)` is the viewport center.
- `(-0.5, -0.5)` is the top-left corner.
- The valid range from `-0.5` to `0.5` spans the viewport.

For example, a camera at the world origin with a top-left anchor maps world
coordinates to screen coordinates without adding a manual half-screen offset:

```go
camera, err := render.SpawnCamera(
	g.World(),
	render.Camera{
		Primary: true,
		Anchor:  geom.Vector2{X: -0.5, Y: -0.5},
	},
	geom.Vector2{},
	1,
)
```

The anchor affects projection and culling together. `WorldToScreen`,
`ScreenToWorld`, and the collector's culling rectangle all use the same
anchored view.

## Rotate the view

Camera rotation is `core.Transform.Rotation`, measured in radians. Positive
values rotate the camera counter-clockwise in world space, which appears
clockwise on a Y-down screen. Rotation is interpolated numerically, not along
the shortest arc, so a change that crosses pi may take the longer path.

A sprite rotated by the same angle as the camera appears unrotated on screen.
Use this cancellation rule when deciding how to combine camera and sprite
rotation.

Rotated views have two practical consequences:

- Culling uses the circumscribed rectangle of the rotated view. The rectangle
  may be larger than the visible area, but visible objects are not culled.
- `WorldToScreen` rounds after applying the complete transform, including
  rotation. See [the coordinate system](../reference/coordinate-system.md)
  for the projection details and pixel-snapping rules.

## Camera bounds

`Camera.Bounds` is an optional world-space rectangle. A zero or empty
rectangle leaves the view unbounded. Otherwise, the renderer clamps the
resolved view after interpolation and before projection. This does not
change the camera entity's transform.

For relative movement, use `Camera.MovePosition`. It constrains the starting
position to the camera's movement window, applies a world-space displacement,
and constrains the destination. Movement therefore starts from the boundary
when the stored position is outside the window, rather than accumulating
off-screen travel. The method returns a position without modifying the
supplied components.

With `cam` and `transform` read from the camera entity's `render.Camera` and
`core.Transform` components, apply the returned position explicitly:

```go
position, err := cam.MovePosition(transform, displacement, ctx.LogicalWidth, ctx.LogicalHeight)
if err != nil {
	return err
}
transform.Position = position
return camera.SetComponent(ctx.World, transform)
```

The caller controls speed and elapsed-time scaling. Rotate screen-relative
movement into world space before passing it, and apply zoom and rotation
changes to the transform before calling the method. A zero displacement still
constrains the position, so the same call handles changes to framing without
pan input. Invalid anchors, zoom, non-finite movement values, and nonpositive
logical viewport dimensions return an error at the call.

The movement window uses the renderer's bounds calculation, accounting for
zoom, rotation, anchor, and logical viewport size. The view's half-extents
shrink the region its center can move through. Rotated views use conservative
circumscribed extents; if the view cannot fit on an axis, its center pins to
the bounds center on that axis. Empty bounds leave relative movement unbounded.

`MovePosition` is opt-in: assigning an absolute position or following a target
does not invoke it. It leaves `PrevTransform` and interpolation unchanged, so
an already out-of-bounds previous snapshot can still pin the view during the
transition. See the [camera example](../../examples/camera/main.go) for
panning, rotation, and zoom using this operation.

## Read the rendered view

The collector publishes the resolved framing as the world's
`*render.CameraView` resource. It contains the interpolated position and
rotation, zoom, and anchor for the most recently drawn frame. A reader in a
frame or fixed phase therefore sees the previous frame's view, and the
resource does not exist until the first draw.

Handle that first-frame case explicitly. Do not use `MustResource`, which is
for resources registered before the game loop starts:

```go
view, err := ctx.World.Resource[*render.CameraView]()
if errors.Is(err, core.ErrResourceNotRegistered) {
	return nil
}
if err != nil {
	return err
}
```

For cursor picking, convert the screen position through the published view:

```go
cursor := view.ScreenToWorld(
	ctx.Input.Cursor(),
	ctx.LogicalWidth,
	ctx.LogicalHeight,
)
```

Do not construct a `CameraView` by hand. A hand-built view uses component
state rather than the interpolated view that was actually drawn, which can
make picking drift between ticks.

## Where to go next

- [Rendering](rendering.md) - drawables, layers, ordering, and overlays.
- [Input](input.md) - actions, snapshots, and frame-versus-tick reads.
- [The coordinate system](../reference/coordinate-system.md) - projection
  math, conventions, and pixel snapping.
- [Your first game](../getting-started/your-first-game.md) - a follow camera
  and cursor aiming in a complete program.
