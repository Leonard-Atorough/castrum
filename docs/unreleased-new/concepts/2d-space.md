# 2D space and the camera

Castrum keeps a distinction between the world you simulate and the logical canvas the runner draws. A transform locates an entity in the world; the camera determines which part of that world reaches the canvas.

## World positions and transforms

World X increases to the right and Y increases downward. A sprite's transform component holds its position, rotation, scale, and offset. Texture and rectangle drawables are centered on their position; a line instead describes endpoints relative to its position. Scale changes the drawable, and rotation is expressed in radians. A zero scale component is treated as one.

An entity does not need to be on screen to exist or be updated by systems. The camera determines what is visible. See [Draw and position an image](../guides/rendering/images.md) for a working example.

## From world to logical screen

The default camera is at the world origin, has zoom one, and maps its position to the center of the logical canvas. With no camera rotation, a point one world unit to the right of the camera appears one logical pixel to the right of that center at zoom one. Zoom two makes that distance two logical pixels.

The camera anchor changes where its position maps on the canvas. The zero anchor is the center; `(-0.5, -0.5)` is the top-left. Rotation changes the view's orientation, and bounds can clamp the resolved camera view. See [Cameras](../guides/rendering/cameras.md) to configure these choices.

The runner's logical canvas and physical window can have different sizes. World projection and camera bounds use the logical dimensions; the runner scales the logical canvas for display. A window resize does not by itself change the world's coordinate system.

## Frame and tick positions

Gameplay commonly changes transforms on fixed ticks while the runner draws at the display frame rate. The engine captures the previous transform before fixed-tick movement and the renderer interpolates between previous and current positions, rotations, and scales. This smooths display between ticks without changing the simulated position.

The world adds `PrevTransform` when a `Transform` is created. Treat that snapshot as engine-owned; do not write or remove it. Rotation interpolation is numeric, not a shortest-path angle interpolation. Offset uses the current transform.

## The last rendered view

After successful collection, the renderer publishes a `*render.CameraView` resource describing that collected frame. It includes interpolation and bounds, so it can differ from the camera entity's current transform. `WorldToScreen` projects a world point to logical pixels and rounds it; `ScreenToWorld` performs the inverse without rounding.

The resource is available only after a successful collection with a primary camera. If collection fails or no primary camera is available, a previously published view remains unchanged. Systems therefore see the most recently collected view, not necessarily the camera state they just wrote. Use the logical dimensions with these conversions.
