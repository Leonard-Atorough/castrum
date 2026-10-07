# The coordinate system

Castrum uses a 2D world whose positive X direction points right and positive Y points down. A drawable is centered on its transform position. A position of `(640, 360)` therefore places the center of a sprite at that world point, not its upper-left corner.

## Transform state

`Transform` stores position, rotation in radians, scale, and offset. A zero scale axis is treated as one during collection. Sprite and shape dimensions are expressed in world units, then affected by transform scale and camera zoom.

When a transform enters the world, Castrum adds a matching `PrevTransform` unless one was supplied. The engine captures the current position at the start of each fixed tick. Game code should not write or remove `PrevTransform`; it is the renderer's interpolation state.

## Camera projection

A `CameraView` maps a world point to logical-screen coordinates using the camera's world position, zoom, and logical dimensions:

```text
screen = round((world - camera.position) * zoom + logicalSize / 2)
```

The camera position is the world point at the center of the logical screen. At zoom 1, one world unit maps to one logical pixel. `ScreenToWorld` reverses the projection without the final rounding, so the result can contain fractional world coordinates.

The engine camera is created by `castrum.New` and is centered on half the logical size at zoom 1. A user camera marked primary takes precedence; if multiple cameras claim primary, the collector uses the first matching camera in its deterministic query order.

The logical size belongs to the runner and is published through `Context.LogicalWidth` and `LogicalHeight`. The window can be larger or smaller; the runner scales the logical canvas, while world projection and culling continue to use the logical dimensions.

## Pixel snapping

Projection rounds the screen-space center to whole pixels. The renderer then builds the sprite or shape around that snapped center, so its corners and line endpoints remain rigid rather than being rounded independently. Snapping keeps pixel art stable, but it also quantizes sub-pixel motion.

Rotation is applied around the drawable's centered position. Culling uses the interpolated position and the drawable's bounds; rotation is not included in the culling bounds, so a long rotated drawable can be culled while part of its rotated image would otherwise reach the view.

## Interpolation

The collector interpolates transform position between `PrevTransform` and `Transform` using `Context.Alpha`. Rotation, scale, and offset use the current transform. This makes fixed-step movement appear smooth without running gameplay once per display frame.

The previous-transform capture must run before gameplay movement. The [scheduler](the-scheduler.md) documents that ordering; the [rendering guide](../guides/rendering.md) covers the game-facing camera and drawable APIs.

## Where to go next

- [Rendering](../guides/rendering.md) - cameras, drawables, culling, and ordering.
- [The scheduler](the-scheduler.md) - fixed ticks and alpha.
- [Runner separation](runner-separation.md) - logical and window sizes at the platform boundary.
