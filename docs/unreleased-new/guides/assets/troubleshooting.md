# Diagnose an asset or rendering failure

If an image fails to load or does not appear, start with the error and the point at which it occurs. Setup loads can fail before the game starts; a sprite can also defer a file or region failure until the renderer collects it.

## The file cannot be opened

1. Check the asset path, spelling, case, and extension in the reported error. Asset paths use `/`, including on Windows.
2. With the default filesystem, check the process working directory. It must contain the path you supplied; it need not be the directory containing the Go source.
3. If you passed `castrum.WithFilesystem`, check the path relative to that filesystem's root. For `embed.FS`, check that the file is in the embedded tree.
4. Load the required image during setup as shown in [Load game assets](loading.md) to expose the failure before drawing.

## The image cannot be decoded

The requested Go type and file format select the decoder. The default game loader can decode PNG and JPEG as `asset.TextureData`; a renamed file still has to contain valid image data. If you supply custom codec configuration, check which decoders it provides. Wrap and return the original error so the operation and underlying cause remain visible.

## A spritesheet fails

Check the sheet document and its texture path separately. Both paths are relative to the filesystem root; the texture path inside the document is not relative to the document's directory.

The default sheet decoder checks metadata when the JSON is loaded. A valid document can still name a missing image, refer to a missing region, or contain a rectangle outside the image. Resolve the sheet, load its texture, call `ValidateBounds`, and check each animation frame name during setup as shown in [Draw and animate a spritesheet](../rendering/spritesheets.md).

## Nothing is visible

1. Confirm the entity has both a `core.Transform` and a `render.Sprite`, and that the sprite has a drawable and is not hidden.
2. Check the camera's position, zoom, anchor, and bounds. An entity can exist outside the camera view. Start with [Cameras](../rendering/cameras.md).
3. Check the transform's scale and the sprite's color and alpha. A nil drawable draws nothing; transparent color can make a drawable invisible.
4. Confirm the game is running with a runner that renders the world, and return errors from setup and the loop rather than ignoring them.

For an asset-free comparison, run the [primitive example](../../../../examples/primitive/main.go) from `examples/primitive` in the [development workspace](../../../../CONTRIBUTING.md#develop-examples-against-the-checkout). It displays shapes without loading images.
