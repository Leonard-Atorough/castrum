# The coordinate system

Castrum uses a 2D world whose positive X direction points right and positive Y points down. A drawable is centered on its transform position. A position of `(640, 360)` therefore places the center of a sprite at that world point, not its upper-left corner.

## Transform state

`Transform` stores position, rotation in radians, scale, and offset. A zero scale axis is treated as one during collection. Sprite and shape dimensions are expressed in world units, then affected by transform scale and camera zoom.

When a transform enters the world, Castrum adds a matching `PrevTransform` unless one was supplied. The engine captures the current transform state - position, rotation, and scale - at the start of each fixed tick. Game code should not write or remove `PrevTransform`; it is the renderer's interpolation state.

## Camera projection

A camera pairs `render.Camera` framing policy with a `Transform`. Its uniform `Transform.Scale` is zoom: zero means 1, values above 1 zoom in, and values between 0 and 1 zoom out. Camera scale must resolve to positive and uniform; negative or non-uniform scale returns a named error during collection.

The renderer resolves and publishes a `CameraView`; game code reads that resource rather than constructing one. With the centered zero anchor and no rotation, it maps a world point to logical-screen coordinates using the camera's world position, zoom, and logical dimensions:

```text
screen = round((world - camera.position) * zoom + logicalSize / 2)
```

The zero camera anchor places the camera position at the logical-screen center. `Camera.Anchor` offsets that point by viewport fractions: `(-0.5, -0.5)` places it at the top-left, removing the need to offset the camera position by half the screen size. At zoom 1, one world unit maps to one logical pixel. `ScreenToWorld` reverses the projection without the final rounding, so the result can contain fractional world coordinates.

The engine camera is created by `castrum.New` at the world origin with a centered anchor and zoom 1. A user camera marked primary takes precedence; if multiple cameras claim primary, the collector uses the first matching camera in its deterministic query order.

The logical size belongs to the runner and is published through `Context.LogicalWidth` and `LogicalHeight`. The window can be larger or smaller; the runner scales the logical canvas, while world projection and culling continue to use the logical dimensions.

Camera rotation is `Transform.Rotation`. Positive radians are counter-clockwise in world space and visually clockwise on the Y-down screen. It interpolates numerically, not along the shortest arc. The view applies the inverse camera rotation, so a sprite with the same rotation as the camera appears unrotated.

`Camera.Bounds` is an optional world `geom.Rect`. An empty rectangle leaves the view unbounded; otherwise the resolved view clamps after interpolation and before projection. If the view cannot fit on an axis, its center pins to that bounds center on that axis.

## Pixel snapping

`WorldToScreen` rounds after the complete transform, including anchor and rotation. The renderer then builds the sprite or shape around that snapped center, so its corners and line endpoints remain rigid rather than being rounded independently. Snapping keeps pixel art stable, but it also quantizes sub-pixel motion.

Rotation is applied around the drawable's centered position. Culling uses the interpolated position and the drawable's bounds; rotation is not included in the culling bounds, so a long rotated drawable can be culled while part of its rotated image would otherwise reach the view.

## Interpolation

The collector interpolates transform position, rotation, and scale between `PrevTransform` and `Transform` using `Context.Alpha`. Rotation interpolates as a plain numeric lerp, so a rotation crossing pi or -pi spins the long way round. Offset uses the current transform. This makes fixed-step movement appear smooth without running gameplay once per display frame.

The previous-transform capture must run before gameplay movement. The [scheduler](the-scheduler.md) documents that ordering; the [rendering guide](../guides/rendering.md) covers the game-facing camera and drawable APIs.

## Where to go next

- [Rendering](../guides/rendering.md) - cameras, drawables, culling, and ordering.
- [The scheduler](the-scheduler.md) - fixed ticks and alpha.
- [Runner separation](runner-separation.md) - logical and window sizes at the platform boundary.
