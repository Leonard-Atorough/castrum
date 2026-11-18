# Load game assets

Your game refers to an asset by a slash-separated path relative to its asset filesystem. The game's loader reads and decodes that path; rendering and other consumers use the decoded data when they need it.

This guide introduces loading shipped assets and choosing where they come from. It uses an image as a concrete example and assumes a game `g` created by `castrum.New`. To put the image on an entity, see [Draw and position an image](../rendering/images.md). For saving game data, custom codecs, streams, and other asset workflows not yet covered here, use the existing [assets guide](../../../unreleased/guides/assets.md).

## Load a required image during setup

Put `player.png` under an `assets` directory in the process working directory. In your error-returning setup function, use the loader the game provides:

```go
loader := g.World().MustResource[*asset.Loader]()
if _, err := loader.Load[asset.TextureData]("assets/player.png"); err != nil {
	return fmt.Errorf("load player image: %w", err)
}
```

Import `fmt` and `github.com/Leonard-Atorough/castrum/asset`. Run your game from the directory containing `assets`. The load opens and decodes the image during setup, so a missing file or invalid image is reported before you start the loop. The default image decoders support PNG and JPEG.

The `TextureData` is a decoded Go image, not a GPU texture. Use the same asset path in a `render.TextureSource`; the runner resolves it for drawing. See [Draw and position an image](../rendering/images.md).

## Choose the filesystem root

By default, the game's filesystem reads from the process working directory, not from the source file's location. Keep asset paths slash-separated even on Windows.

To package the `assets` directory with a Go program, place it next to the file containing this declaration and construct the game with the embedded filesystem:

```go
import "embed"

//go:embed assets
var files embed.FS
```

```go
g, err := castrum.New(castrum.WithFilesystem(files))
if err != nil {
	return fmt.Errorf("create game: %w", err)
}
```

These are excerpts: use the declaration at package scope, and use the game construction in an error-returning function with `fmt` and `castrum` imported. The path remains `assets/player.png`; embedding changes where the filesystem gets its bytes, not the asset name. Include all files the game needs in the embedded tree. Castrum also supplies its built-in assets under `builtin/`.

## Decide when to load

Use explicit setup loads for files the game cannot run without. A `TextureSource` can instead resolve an image when the renderer collects it; this keeps setup shorter but moves file and decode failures into drawing. The default loader reuses successfully decoded values for the same cleaned path, requested type, and format. Preloading through the game's loader therefore also makes that decoded image available to later reuse loads.

Loading an asset does not create an entity or draw it. For how paths, codecs, caching, and lazy consumers fit together, see [How assets reach your game](../../concepts/assets-and-loading.md). For concrete checks when loading fails, see [Diagnose an asset or rendering failure](troubleshooting.md).
