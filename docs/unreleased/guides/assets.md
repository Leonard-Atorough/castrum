# Assets

Assets are files or other data that your game identifies by a path and loads through an `asset.Loader`. An `asset.Saver` encodes typed values through a store or writer. This guide covers path resolution, codec configuration, caching, streaming, and explicit saving.

## Where assets live

By default, asset paths resolve against the process working directory. If your game starts in a directory containing `assets/sprites/hull.png`, load it with the path `assets/sprites/hull.png`.

Give the game an explicit `fs.FS` when you want to control the asset root. An embedded filesystem is useful for a single-binary release:

```go
import "embed"

//go:embed assets
var files embed.FS

g, err := castrum.New(
	castrum.WithTitle("My game"),
	castrum.WithFilesystem(files),
)
if err != nil {
	return err
}
```

The asset paths do not change when you switch from a directory to `embed.FS`; only the filesystem that resolves them changes. Keep every referenced file under the embedded directory. See [Publishing your game](publishing-your-game.md) for the complete release pattern.

## Built-in assets

The [`builtin`](../../../builtin) package ships the Go Regular font at `builtin.Font`. `castrum.New` makes built-in assets available automatically, alongside the default working-directory filesystem or any filesystem you provide:

```go
import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/builtin"
)

g, err := castrum.New(
	castrum.WithFilesystem(files),
)
if err != nil {
	return err
}

if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
	return err
}
```

The font is embedded in binaries that use Castrum and is otherwise loaded lazily. `builtin.Register` preloads and caches it, returning errors during setup instead of on the first rendered frame. Paths under `builtin/` are reserved for engine-shipped assets.

## Loading assets

### Supported asset formats

The loader chooses a decoder from the lowercased file extension. The built-in loader decodes PNG, JPG, and JPEG files into `asset.TextureData`. It also decodes the JSON sidecars used by atlases into `asset.AtlasMeta` if available.

The default runner, `ebitrun`, registers WAV, MP3, and OGG decoders for `asset.AudioData` when you create it. Construct the runner before explicitly preloading audio. A format constant such as `asset.FormatYAML` is only vocabulary; it does not mean that a decoder is available.

The built-in format constants are `FormatJSON`, `FormatYAML`, `FormatYML`, `FormatPNG`, `FormatJPG`, `FormatJPEG`, `FormatWAV`, `FormatMP3`, and `FormatOGG`.

### Cached and non-cached loading

Use `Load[T]` for an asset that should be identified by a path and reused. The loader infers the format from the extension, decodes the value as `T`, and caches it. Repeated loads return the cached value, and concurrent loads of the same path, type, and format share one in-flight decode.

Preload assets during setup when a missing file or decode failure should stop the game from starting. You also avoid paying the file and decode cost on the first frame:

```go
if _, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData]("assets/sprites/hull.png"); err != nil {
	return err
}
```

Use `LoadReader` when the data comes from somewhere other than the loader's filesystem. A reader has no extension, so pass the format explicitly. This path does not cache or deduplicate the decode, and it does not close the reader:

```go
dialogue, err := loader.LoadReader[Dialogue](reader, asset.FormatJSON)
if err != nil {
	return err
}
```

Use `WithID` when you deliberately want a stable logical cache identity, such as for generated data:

```go
data, err := loader.Load[Dialogue](
	"generated/dialogue.json",
	asset.WithID("dialogue:intro"),
)
```

An overridden ID is part of correctness, not just an optimization. Two different files with the same ID, target type, and format share the first cached value.

#### Loading audio

Load decoded audio as `asset.AudioData` after `ebitrun.New` has registered the audio codecs:

```go
if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData]("assets/audio/confirm.ogg"); err != nil {
	return err
}
```

For normal playback, prefer the audio component's `Load: audio.LoadEager` for short effects that play often. The decoded PCM is cached and can be shared by multiple players. Use `audio.LoadStream` for long music; the [audio guide](audio.md) covers that choice and the playback component.

#### Loading images

Load images as `asset.TextureData`. The asset package decodes them into backend-free Go image values; the runner converts them to a renderable texture when it needs to draw them:

```go
texture, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData]("assets/sprites/hull.png")
if err != nil {
	return err
}
_ = texture.Image
```

### Streaming assets

Use `Open` when a consumer needs the raw file instead of a cached decoded value. `Open` returns the file and the format inferred from its extension. You own the returned file and must close it:

```go
file, format, err := loader.Open("assets/audio/music.ogg")
if err != nil {
	return err
}
defer file.Close()
_ = format
```

Streaming avoids decoding and caching the entire asset, but the consumer must perform the format-specific decoding. A streaming source also needs to support seeking when playback loops or resets its position. Normal directory files and `embed.FS` files meet that requirement; verify custom filesystem implementations before using them for streaming.

#### Streaming audio

The `ebitrun` audio provider performs the streaming path for you when an `audio.Source` uses `audio.LoadStream`. It opens the file, selects the WAV, MP3, or OGG decoder, and keeps the source open for the lifetime of that play. When the play ends or is removed, the provider closes the source.

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

Use streaming for long tracks that should not occupy memory as one decoded PCM buffer. Streaming is still lazy: missing files, unsupported formats, and decoder errors appear when the provider creates the play.

## Shared codec configuration

Use `asset.Codecs` to configure conversion once and reuse it across loaders and savers. `asset.NewCodecs` retains the default image, font, and atlas decoders. Configure game-defined decoders before construction:

```go
codecs := asset.NewCodecs()
if err := codecs.SetDecoder(asset.FormatJSON, asset.Decoder[Dialogue](decodeDialogue)); err != nil {
	return err
}
g, err := castrum.New(
	castrum.WithFilesystem(files),
	castrum.WithAssetCodecs(codecs),
)
if err != nil {
	return err
}
```

For data needed before game construction, use `asset.NewLoader(files, asset.LoaderOptions{Codecs: codecs})`, which returns a loader and an error. `LoaderOptions{}` selects defaults; a nil filesystem uses the working directory. Standalone loaders that need engine-shipped files use `builtin.WithFilesystem(files)`.

Each loader snapshots registrations at construction. Changing `codecs` later affects only new loaders. Sharing configuration does not share caches or storage. Snapshots copy functions, not captured closure state, so custom decoders must support concurrent calls if used concurrently.

The runner registers audio decoders only on its game's loader at the effective audio-context sample rate. It does not modify the source configuration or standalone loaders; conflicting registrations return errors.

## Custom asset formats

Register a typed decoder when your game needs a value that Castrum does not decode by default. A decoder receives an `io.Reader` and should parse and validate its value, returning an error for invalid input:

```go
type Dialogue struct {
	Lines []string `json:"lines"`
}

func decodeDialogue(reader io.Reader) (Dialogue, error) {
	var dialogue Dialogue
	err := json.NewDecoder(reader).Decode(&dialogue)
	return dialogue, err
}

loader := g.World().MustResource[*asset.Loader]()
if err := loader.RegisterDecoder(
	asset.FormatJSON,
	asset.Decoder[Dialogue](decodeDialogue),
); err != nil {
	return err
}

dialogue, err := loader.Load[Dialogue]("assets/dialogue/intro.json")
if err != nil {
	return err
}
```

Local duplicate registrations fail. To install or replace a decoder in configuration, use `Codecs.SetDecoder` before constructing the loader. There is no live replacement behind cached values. Decoders must not close or retain their reader.

For a custom extension, define a format and override inference when loading:

```go
type LevelData struct {
	Name string
}

levelFormat := asset.Format("level")
if err := loader.RegisterDecoder(
	levelFormat,
	asset.Decoder[LevelData](decodeLevel),
); err != nil {
	return err
}

level, err := loader.Load[LevelData](
	"assets/world.bin",
	asset.WithFormat(levelFormat),
)
```

## Save game-defined data

Use a long-lived saver for preference values or game-defined documents, not `settings.Group`, runtime providers, or a live world. Applying loaded values and reconstructing gameplay state remain explicit game code.

An encoder decides how a typed value becomes bytes. A `SaveStore` decides where those bytes go and when they become visible. An `fs.FS` alone does not provide write capability. Castrum does not yet provide a directory-backed store or automatically construct a saver.

### Configure an encoder

Install an encoder before constructing the saver:

```go
import (
	"encoding/json"
	"io"
)

type Profile struct {
	Name string `json:"name"`
}

codecs := asset.NewCodecs()
if err := codecs.SetEncoder(asset.FormatJSON, func(writer io.Writer, value Profile) error {
	return json.NewEncoder(writer).Encode(value)
}); err != nil {
	return err
}
```

`SetEncoder` installs or replaces that direction on the configuration. An existing saver's `RegisterEncoder` only adds entries; duplicate registrations fail. Encoders must not close their writer.

### Supply a store

The saver selects the encoder by exact static Go type and inferred lowercased extension. It calls `store.Write(name, encode)` with a callback capturing the encoder and value. The store supplies a staging writer to `encode`, handles errors, and publishes completed output. It does not look up a codec or receive a `Profile`.

This minimal store stages bytes in memory:

```go
import "bytes"

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

Use it with the configured encoder:

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

The map contains JSON bytes, not a `Profile`. This store is in-memory only and not safe for concurrent access. A file-backed store must preserve existing output on encoding failure rather than truncate the destination first.

`Write` calls `encode` synchronously at most once and must call it before returning success. It must not retain the callback or publish incomplete output on an encoding failure. The store owns its writer and reports write, close, and publication failures. Replacement, directory creation, and durability guarantees belong to storage.

`Save` cleans slash-separated names relative to the store. Empty names, absolute paths, backslashes, and paths still escaping the root after cleaning are rejected. Missing encoders and invalid names fail before storage is invoked. `WithFormat` overrides extension inference.

### Write to an existing stream

Use a nil store for stream-only saving:

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

`SaverOptions{}` selects defaults, which do not include general-purpose encoders. `SaveWriter` never closes the writer and cannot roll back bytes already written. `Save` on a stream-only saver returns an error. Both constructors snapshot configuration without sharing caches or selecting storage.

Saving does not invalidate loader caches. To reread changed bytes, use `Open` followed by `LoadReader` on a filesystem that can read the saved destination. Sharing codecs alone does not make saved bytes available to a loader. Coordinate same-destination saves, use concurrency-safe encoders and storage for concurrent calls, and copy mutable values before handing them to a worker.

## Atlases

Treat an atlas as setup for image loading: registration loads the image through the normal typed cache, builds the named regions, validates their bounds, and stores the result for sprites to resolve later.

For a regular grid, register the image with tile dimensions and a prefix. Regions are named in row-major order as `prefix_0`, `prefix_1`, and so on:

```go
if err := g.World().MustResource[*asset.Loader]().RegisterGridAtlas(
	"characters", "assets/sprites/characters.png", 16, 16, "char",
); err != nil {
	return err
}

sprite := render.Sprite{
	Drawable: render.AtlasSource{Atlas: "characters", Region: "char_0"},
}
```

For irregular regions, put the pixel rectangles in a JSON sidecar and register the image and sidecar together:

```json
{
  "regions": [
    { "name": "hero_idle_0", "x": 0, "y": 0, "w": 32, "h": 32 },
    { "name": "hero_idle_1", "x": 32, "y": 0, "w": 32, "h": 32 }
  ]
}
```

```go
if err := loader.RegisterAtlasFromSidecar(
	"characters",
	"assets/sprites/characters.png",
	"assets/sprites/characters.atlas.json",
); err != nil {
	return err
}
```

Register atlases during setup, before spawning entities that refer to them. A malformed sheet or sidecar then fails near its declaration; an unknown atlas or region still fails when the renderer resolves the sprite.

## Errors and loading strategy

`Load` returns an `asset.AssetError` for filesystem and decoding failures. Its `Op` is `Load` or `Decode`, and the underlying error remains available through `errors.Is` and `errors.As`. `Open` reports `Op: "Open"`.

Choose eager or lazy loading according to when failure should surface:

- Preload required images, effects, configuration, and atlases during setup.
- Load optional or infrequently used assets when they are needed, accepting the first-use cost.
- Use eager audio for short repeated effects and streaming audio for long tracks.
- Keep error context when wrapping an asset error so the path and operation remain visible.

## Best practices

- Keep asset paths stable and relative to the configured `fs.FS`; do not build paths from the process's current directory in game logic.
- Use typed loads so the intended decoded value is visible at the call site.
- Register custom decoders and atlases before startup or before the first system that depends on them.
- Choose explicit cache IDs only when you can maintain their uniqueness and meaning.
- Close every file returned by `Open`; `LoadReader` does not close its input.
- Use `embed.FS` for releases that should not depend on a user's working directory.

## Where to go next

- [Rendering](rendering.md) - the drawables that consume textures and atlas regions.
- [Audio](audio.md) - eager and streaming playback, buses, and the mixer.
- [Publishing your game](publishing-your-game.md) - embedding assets in a single binary.
- [The asset pipeline](../reference/asset-pipeline.md) - the implementation-level loading and caching model.
