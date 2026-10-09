# Publishing your game

Publishing a Castrum game is a Go release with an `ebitrun` target. The usual release shape is:

1. Keep the game's assets under a directory in the command package.
2. Embed that directory with Go's `embed` package.
3. Pass the embedded filesystem to `castrum.New` with `WithFilesystem`.
4. Construct `ebitrun` and hand the game to it with `g.Run`.
5. Build once for each target platform and test the resulting binary outside the repository.

This keeps asset lookup independent of the process working directory. Castrum does not provide a separate packaging format; the Go binary, the embedded files, and the runner's platform integration are the release.

## Embed the asset tree

Give the command package a stable asset root:

```text
mygame/
  main.go
  assets/
    sprites/
      player.png
    audio/
      music.ogg
```

With `//go:embed assets`, the embedded filesystem contains the `assets` directory, so the IDs remain `assets/sprites/player.png` and `assets/audio/music.ogg`. The same IDs resolve from the working directory during development and from `embed.FS` in a release; only the filesystem changes:

```go
import "embed"

//go:embed assets
var files embed.FS
```

Keep every referenced file under that directory. The [assets guide](assets.md) covers path resolution and loading; the [wander example](../../examples/wander) shows this pattern in a runnable game.

## Use the embedded filesystem

Pass the filesystem when constructing the game, before registering assets or creating entities that depend on them:

```go
func run() error {
	g, err := castrum.New(
		castrum.WithTitle("My Game"),
		castrum.WithFilesystem(files),
	)
	if err != nil {
		return err
	}

	if err := g.World().MustResource[*asset.Server]().RegisterGridAtlas(
		"characters", "assets/sprites/characters.png", 16, 16, "char",
	); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	return g.Run(runner)
}
```

`WithFilesystem` accepts any `fs.FS`. The default is the process working directory, which is convenient for development but fragile for a published executable. `embed.FS` is read-only: use it for shipped assets, not save games, downloaded content, screenshots, or other user data. Store writable data separately using the platform's normal application-data location.

## Fail before the window opens

Put required setup and validation before `g.Run`:

- Register atlases and custom decoders.
- Preload assets whose absence should prevent launch.
- Spawn initial entities and check every returned error.
- Construct `ebitrun` and validate its options.
- Construct the runner before explicitly preloading `asset.AudioData`; it registers the runner's audio decoders.

The runner resolves eager resources and runs startup systems before opening its platform loop. A startup failure returns from `g.Run` without opening a window. Lazy loads and streaming playback can still fail later, so keep the asset IDs and errors visible when wrapping them. See [Audio](audio.md) for eager and streaming playback behavior.

## Build for a target

Build from the module root. Embedded files are compiled into the binary, so the output does not need the repository's `assets/` directory at runtime:

```sh
GOOS=windows GOARCH=amd64 go build -o dist/mygame-windows-amd64.exe .
GOOS=darwin  GOARCH=arm64 go build -o dist/mygame-darwin-arm64 .
GOOS=linux   GOARCH=amd64 go build -o dist/mygame-linux-amd64 .
```

In PowerShell, set the environment variables for each command instead:

```powershell
$env:GOOS = "windows"; $env:GOARCH = "amd64"; go build -o dist/mygame-windows-amd64.exe .
```

Use a separate output name for each `GOOS` and `GOARCH` pair. Cross-compiling Ebitengine can also require platform-specific native build support; consult the current [Ebitengine platform documentation](https://ebitengine.org) for mobile, console, graphics, windowing, and distribution requirements.

## Test the release artifact

Run the built executable from a directory unrelated to the source tree. This catches accidental dependence on the process working directory:

```sh
cd /tmp
/path/to/mygame-linux-amd64
```

Exercise the first use of every lazy asset and startup path. Embedding proves that files are present in the binary; it does not prove that an asset ID, decoder, atlas region, or runtime platform configuration is correct.

## Where to go next

- [examples/wander](../../examples/wander) - a runnable embedded-filesystem pattern.
- [Assets](assets.md) - filesystem roots, typed loads, atlases, and streaming.
- [Audio](audio.md) - eager and streaming playback, including embedded files.
- [Ebitengine's platform docs](https://ebitengine.org) - the maintained platform matrix and distribution guidance.
