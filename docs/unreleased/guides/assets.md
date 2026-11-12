# Assets

The `asset` package converts files and streams into typed Go values and back again. A loader reads from an `fs.FS` and caches decoded assets; a saver encodes values to a write-capable store or a caller-owned writer. Codecs define the conversion for each Go type and format. Loading and saving are separate: sharing codecs does not connect their storage or caches.

## Load your first asset

`castrum.New` provides an `*asset.Loader` on the world. By default, it reads game assets from the process working directory. If that directory contains `assets/sprites/hull.png`, preload the image during setup:

```go
import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
)

g, err := castrum.New()
if err != nil {
	return err
}

loader := g.World().MustResource[*asset.Loader]()
if _, err := loader.Load[asset.TextureData]("assets/sprites/hull.png"); err != nil {
	return err
}
```

`Load[T]` chooses a decoder for the requested Go type and the file's lowercased extension. By default, it opens the file, decodes it, closes it, and caches the result. Preloading moves missing-file, decode, and close errors into setup instead of the first frame that needs the asset.

`asset.TextureData` contains a backend-free Go image, not a GPU texture. The runner creates the renderable texture when needed. See [Rendering](rendering.md) for using images in sprites.

## Choose the asset root

Asset names are slash-separated paths relative to the configured filesystem, including on Windows. The loader cleans paths before using them. Keep these names stable rather than constructing absolute paths in game logic.

Pass `castrum.WithFilesystem` to control the root. An embedded filesystem is useful for a single-binary release:

```go
import "embed"

//go:embed assets
var files embed.FS

g, err := castrum.New(
	castrum.WithFilesystem(files),
)
if err != nil {
	return err
}
```

Here, the same `assets/sprites/hull.png` name works with either the working-directory filesystem or `embed.FS`; only the filesystem resolving it changes. Keep referenced game files under the embedded directory. See [Publishing your game](publishing-your-game.md) for the complete release pattern.

### Built-in assets

The [`builtin`](../../../builtin) package ships the Go Regular font at `builtin.Font`. `castrum.New` overlays built-in files onto the default or supplied filesystem automatically. Paths under `builtin/` are reserved for engine-shipped assets.

The font is embedded but loaded lazily. To preload it and report errors during setup, import `github.com/Leonard-Atorough/castrum/builtin` and call:

```go
if err := builtin.Register(loader); err != nil {
	return err
}
```

Standalone loaders do not receive this overlay automatically; give them `builtin.WithFilesystem(files)` when they need engine-shipped files.

## Choose a decoded type and format

The default codecs support these combinations:

| Go type | Formats | Availability |
| --- | --- | --- |
| `asset.TextureData` | PNG, JPG, JPEG | Default loader |
| `asset.FontData` | TTF, OTF | Default loader |
| `asset.AtlasMeta` | JSON | Default loader and saver |
| `asset.AudioData` | WAV, MP3, OGG | Game loader after `ebitrun.New` |
| Game-defined types | JSON or custom formats | Explicit codec registration |

Textures and fonts have default decoders, not default encoders. `AtlasMeta` has both JSON directions. A constant such as `asset.FormatYAML` names a format; it does not install a codec. Registrations match the exact Go type: a codec for `Profile` is not a codec for `*Profile` or every JSON document.

### Cache identity

With the default `CacheReuse` policy, repeated `Load[T]` calls reuse the decoded value. Concurrent calls with the same cleaned path, type, and format share one in-flight read, decode, and close. Keep the loader rather than constructing one per load.

Use `asset.WithFormat` to override extension inference:

```go
texture, err := loader.Load[asset.TextureData](
	"assets/sprites/hull.data",
	asset.WithFormat(asset.FormatPNG),
)
if err != nil {
	return err
}
_ = texture.Image
```

The cache key combines the cleaned source path, exact Go type, and format within one loader. Equivalent paths such as `./assets/sprites/hull.png` and `assets/sprites/hull.png` share a cached value; different paths or loaders do not. `WithID` cache aliases are no longer supported.

## Configure game-defined data

Use `asset.Codecs` to configure conversion once and reuse it across loaders and savers. Start with `asset.NewCodecs` to retain the default image, font, and atlas codecs, then register your own types before constructing the game:

```go
type Profile struct {
	Name string `json:"name"`
}

codecs := asset.NewCodecs()
if err := codecs.SetJSON[Profile](); err != nil {
	return err
}

g, err := castrum.New(
	castrum.WithFilesystem(files),
	castrum.WithAssetCodecs(codecs),
)
if err != nil {
	return err
}

loader := g.World().MustResource[*asset.Loader]()
profile, err := loader.Load[Profile]("profiles/player.json")
if err != nil {
	return err
}
```

This example expects `profiles/player.json` in `files`; include that directory separately if using the earlier `//go:embed assets` declaration.

`SetJSON[T]` installs or replaces JSON encoding and decoding together. It uses standard `encoding/json` behavior, including custom marshal/unmarshal methods. Decoding requires one complete JSON value with only whitespace after it. Missing fields remain zero-valued and unknown fields follow standard JSON behavior. Your game owns validation, defaults, document versions, migration, and applying the loaded values.

### Configuration lifetime

Each loader and saver snapshots registrations at construction. Changing `codecs` later affects only subsequently constructed loaders and savers. Sharing configuration does not share decoded caches or select a storage source.

On `Codecs`, `SetDecoder` and `SetEncoder` install or replace their direction independently. On a constructed loader or saver, `RegisterDecoder` and `RegisterEncoder` only add entries; duplicate registrations fail. Configure replacements before construction rather than trying to replace a codec behind cached values.

Snapshots copy functions, not captured closure state. Custom codecs must support concurrent calls if used concurrently.

### Load before creating a game

A standalone loader is useful when configuration is needed before game or runner construction:

```go
loader, err := asset.NewLoader(files, asset.LoaderOptions{Codecs: codecs})
if err != nil {
	return err
}
profile, err := loader.Load[Profile]("profiles/player.json")
if err != nil {
	return err
}
```

`LoaderOptions{}` selects the default codecs; a nil filesystem uses the working directory. This loader remains independent of the world's loader. In particular, constructing a runner does not register audio decoders on standalone loaders.

## Read changed files without disturbing cached content

Use `CacheBypass` to read mutable documents through a long-lived loader:

```go
profile, err := loader.Load[Profile](
	"profiles/player.json",
	asset.WithCachePolicy(asset.CacheBypass),
)
if err != nil {
	return err
}
```

Bypass opens and decodes the source independently on every call. It does not consume, populate, replace, or invalidate cached values, and does not wait for an in-flight cached read. Concurrent bypass calls do not share their decode through the loader. A custom decoder can still return shared objects; bypass does not deep-copy them.

Choose a default when constructing a loader used primarily for mutable data:

```go
loader, err := asset.NewLoader(files, asset.LoaderOptions{
	Codecs:      codecs,
	CachePolicy: asset.CacheBypass,
})
if err != nil {
	return err
}
```

| Policy | Read/populate decoded cache | Share in-flight reads |
| --- | --- | --- |
| `CacheReuse` (zero/default) | Yes | Yes |
| `CacheBypass` | No | No |

`WithCachePolicy` overrides either default for one load. Invalid policies and nil options return errors before opening a file. Both policies close files they open; failed decoding or closing returns no value and is never cached. If decoding and closing both fail, the returned error preserves both causes.

Bypass is not cache refresh or hot reload. A previously started cached read can still finish and populate the reused cache with its own result. GPU textures and parsed runner fonts are not invalidated.

## Register sprite atlases

Register atlases during setup, before spawning entities that refer to them. Registration loads the image through the normal typed cache, validates the regions, and stores the atlas for sprites to resolve later.

For a regular grid, provide tile dimensions and a non-empty prefix. The image dimensions must be evenly divisible by the tile dimensions. Regions are named in row-major order as `prefix_0`, `prefix_1`, and so on:

```go
if err := loader.RegisterGridAtlas(
	"characters", "assets/sprites/characters.png", 16, 16, "char",
); err != nil {
	return err
}

sprite := render.Sprite{
	Drawable: render.AtlasSource{Atlas: "characters", Region: "char_0"},
}
```

For irregular regions, put pixel rectangles in a JSON sidecar:

```json
{
  "regions": [
    { "name": "hero_idle_0", "x": 0, "y": 0, "w": 32, "h": 32 },
    { "name": "hero_idle_1", "x": 32, "y": 0, "w": 32, "h": 32 }
  ]
}
```

Register the image and sidecar together:

```go
if err := loader.RegisterAtlasFromSidecar(
	"characters",
	"assets/sprites/characters.png",
	"assets/sprites/characters.atlas.json",
); err != nil {
	return err
}
```

Region names must be unique and non-empty, dimensions positive, and rectangles within the image bounds. Duplicate atlas IDs are rejected. A malformed image or sidecar fails during registration; an unknown atlas or region referenced by a sprite fails when the renderer resolves it. See [Rendering](rendering.md) for atlas drawables.

## Read streams without caching

Use `LoadReader[T]` for bytes from somewhere other than the loader's filesystem, or when you need a fresh decode rather than a cached value. Supply the format explicitly because a reader has no extension:

```go
profile, err := loader.LoadReader[Profile](reader, asset.FormatJSON)
if err != nil {
	return err
}
```

This requires the `Profile` codec configured above. `LoadReader` does not cache, deduplicate concurrent decodes, or close the reader. You own the reader's lifetime.

Use `Open` when a consumer needs the raw file. It returns the file and inferred format without decoding or caching:

```go
file, format, err := loader.Open("profiles/player.json")
if err != nil {
	return err
}
defer file.Close()

profile, err := loader.LoadReader[Profile](file, format)
if err != nil {
	return err
}
```

You must close every file returned by `Open`. The consumer owns format-specific decoding and must verify seeking support if it needs to loop or reset its position.

## Choose eager or streaming audio

The default runner registers WAV, MP3, and OGG decoders on the game's loader during construction. Create the runner before explicitly preloading `asset.AudioData`:

```go
runner, err := ebitrun.New(g)
if err != nil {
	return err
}

if _, err := loader.Load[asset.AudioData]("assets/audio/confirm.ogg"); err != nil {
	return err
}
```

These decoders use the runner's audio-context sample rate. They do not modify the source `Codecs` configuration or other loaders, and conflicting registrations fail rather than being replaced.

For short repeated effects, use `audio.LoadEager`, the default on `audio.Source`. It caches complete decoded PCM that multiple players can share. For long music, use `audio.LoadStream`:

```go
source := audio.NewSource()
source.Load = audio.LoadStream
source.Loop = audio.LoopForever
clip := audio.Clip{Audio: "assets/audio/music.ogg"}
_, err := g.World().NewEntity(source, clip)
if err != nil {
	return err
}
```

The `ebitrun` provider opens and decodes a source for each streamed play instead of caching one complete PCM buffer. It keeps the file open until the play is released, then closes it. Its streaming path requires a seekable file, even for non-looping playback. Normal directory files and `embed.FS` files satisfy that requirement; verify custom filesystems.

Streaming remains lazy: missing files, unsupported formats, decoder failures, and non-seekable sources appear when the provider creates the play. See [Audio](audio.md) for playback ownership and controls.

## Add a custom codec

Register a typed decoder for formats the defaults do not cover. A decoder receives an `io.Reader`, must not close or retain it, and should return errors for invalid input. An encoder receives a writer and a typed value and must not close the writer.

For example, a plain-text dialogue decoder can use the `.txt` extension directly:

```go
import "io"

type Dialogue string

textFormat := asset.Format("txt")
if err := loader.RegisterDecoder(
	textFormat,
	asset.Decoder[Dialogue](func(reader io.Reader) (Dialogue, error) {
		data, err := io.ReadAll(reader)
		return Dialogue(data), err
	}),
); err != nil {
	return err
}

dialogue, err := loader.Load[Dialogue]("assets/dialogue/intro.txt")
if err != nil {
	return err
}
```

Use `codecs.SetDecoder` instead when multiple loaders need the registration, or when replacing a default before construction. Use `codecs.SetEncoder` for the reverse direction, or `saver.RegisterEncoder` to add it to one existing saver. For extensionless or differently named files, pass `asset.WithFormat(textFormat)` to `Load` or `Save`.

## Save game-defined data

Use a long-lived saver when your game needs to persist ordinary values. A `SaveStore` provides write capability; an `fs.FS` alone does not. Castrum does not yet provide a directory-backed save store, and the game does not automatically construct a saver.

Save preference values or game-defined documents, not `settings.Group`, runtime providers, or a live world. Applying loaded values and reconstructing gameplay state remain explicit game code.

### How encoders and stores cooperate

The encoder decides **how a value becomes bytes**. The store decides **where those bytes go and when they become visible**. The saver connects them:

1. Configure an `Encoder[Profile]` with `Codecs.SetEncoder`, or use `SetJSON[Profile]` to install both JSON directions.
2. Construct a saver with that configuration and a store.
3. Call `saver.Save("profiles/player.json", profile)`. The saver validates the name and selects the encoder by the static Go type `Profile` and format `json`.
4. The saver calls `store.Write(name, encode)`. It supplies the callback, bound to the selected encoder and value; the store does not construct it or look up a codec.
5. The store prepares a private writer and calls `encode(writer)` synchronously. If encoding fails, it discards the staged output and returns the error.
6. Otherwise, the store finishes writing and closing as needed, then publishes the completed document.

The store only handles a name and a byte-producing callback. With suitable encoders, the same store can save JSON, images, or custom formats without knowing their Go types. Conversely, an encoder can write through different stores or directly to a caller-owned writer.

### A minimal in-memory store

This store stages bytes in a buffer and replaces the named document only after encoding succeeds:

```go
import (
	"bytes"
	"io"
)

type memoryStore map[string][]byte

func (store memoryStore) Write(name string, encode func(io.Writer) error) error {
	var staged bytes.Buffer
	if err := encode(&staged); err != nil {
		return err
	}
	store[name] = bytes.Clone(staged.Bytes())
	return nil
}
```

With the earlier `Profile` codecs:

```go
store := make(memoryStore)
saver, err := asset.NewSaver(store, asset.SaverOptions{Codecs: codecs})
if err != nil {
	return err
}
if err := saver.Save("profiles/player.json", Profile{Name: "Player"}); err != nil {
	return err
}
```

The map now contains JSON bytes at `store["profiles/player.json"]`, not a `Profile` value. This example is in-memory only and is not safe for concurrent access. A file-backed store would instead encode to a staging file, check write and close errors, and publish it according to its platform's replacement guarantees. Opening and truncating the destination before encoding would not preserve the old document on failure.

`Save` selects an encoder by the value's static type and lowercased extension. `asset.WithFormat` overrides the format for extensionless or custom-named files. Names are cleaned slash-separated paths relative to the store; empty names, absolute paths, backslashes, and paths that still escape the root after cleaning are rejected. Missing encoders and invalid names fail before storage is invoked.

The store's `Write(name, encode)` method must call `encode` synchronously at most once, must call it before returning success, and must not retain it. The store owns its writer and must not publish incomplete output on an encoding failure. It reports write, close, and publication failures. Replacement, directory creation, and durability guarantees belong to the store.

### Write to an existing stream

For a caller-owned writer, construct a stream-only saver with a nil store and call `SaveWriter`:

```go
saver, err := asset.NewSaver(nil, asset.SaverOptions{Codecs: codecs})
if err != nil {
	return err
}
var output bytes.Buffer
if err := saver.SaveWriter(&output, Profile{Name: "Player"}, asset.FormatJSON); err != nil {
	return err
}
```

`SaverOptions{}` selects the default codecs. `SaveWriter` never closes the writer and does not use storage. It cannot provide staged publication or roll back bytes already written if encoding fails. `Save` on a stream-only saver returns an error.

### Reload and coordinate mutable data

Saving does not invalidate a loader's decoded cache. To reread changed bytes, use `Load` with `CacheBypass` and a filesystem that can read the saved destination. Sharing codecs alone does not make saved bytes available to a loader.

Coordinate concurrent saves to the same destination so an older snapshot cannot finish after a newer one. Encoders and storage must be concurrency-safe when used concurrently. Copy mutable data before handing it to a worker; [settings groups](settings.md) already provide a policy-based copying boundary.

## Handle failures at the right time

Path and stream load/save operations return errors containing `*asset.AssetError`; underlying causes remain available through `errors.Is` and `errors.As`. `Load` reports `Load`, `Decode`, or `Close`; `Open` reports `Open`; path saves report `Save`, with encoding failures retaining their `Encode` cause. `LoadReader` and `SaveWriter` report `Decode` and `Encode` without inventing a file path. Constructors and registration methods can return ordinary errors instead.

Choose when predictable failures should surface:

- Preload required images, fonts, configuration, and atlases during setup.
- Construct the runner before preloading required audio effects.
- Load optional assets on demand when the first-use cost and failure timing are acceptable.
- Stream long audio when retaining complete decoded PCM would be wasteful, while accepting lazy playback errors.
- Keep error context when wrapping failures so the path and operation remain visible.

## Where to go next

- [Rendering](rendering.md) - drawables that consume textures and atlas regions.
- [Audio](audio.md) - eager and streaming playback, buses, and the mixer.
- [Publishing your game](publishing-your-game.md) - embedding assets in a single binary.
- [The asset pipeline](../reference/asset-pipeline.md) - the implementation-level loading and caching model.
