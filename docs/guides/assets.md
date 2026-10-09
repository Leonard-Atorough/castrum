# Assets

Assets are files or other data that your game identifies by a path and loads through an `asset.Server`. This guide covers where those paths resolve, how typed loading and caching work, and when to stream instead of decoding an entire asset into memory.

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

## Loading assets

### Supported asset formats

The server chooses a decoder from the lowercased file extension. The built-in server decodes PNG, JPG, and JPEG files into `asset.TextureData`. It also decodes the JSON sidecars used by atlases into `asset.AtlasMeta` if available.

The default runner, `ebitrun`, registers WAV, MP3, and OGG decoders for `asset.AudioData` when you create it. Construct the runner before explicitly preloading audio. A format constant such as `asset.FormatYAML` is only vocabulary; it does not mean that a decoder is available.

The built-in format constants are `FormatJSON`, `FormatYAML`, `FormatYML`, `FormatPNG`, `FormatJPG`, `FormatJPEG`, `FormatWAV`, `FormatMP3`, and `FormatOGG`.

### Cached and non-cached loading

Use `Load[T]` for an asset that should be identified by a path and reused. The server infers the format from the extension, decodes the value as `T`, and caches it. Repeated loads return the cached value, and concurrent loads of the same path, type, and format share one in-flight decode.

Preload assets during setup when a missing file or decode failure should stop the game from starting. You also avoid paying the file and decode cost on the first frame:

```go
if _, err := g.World().MustResource[*asset.Server]().Load[asset.TextureData]("assets/sprites/hull.png"); err != nil {
	return err
}
```

Use `LoadReader` when the data comes from somewhere other than the server's filesystem. A reader has no extension, so pass the format explicitly. This path does not cache or deduplicate the decode, and it does not close the reader:

```go
dialogue, err := server.LoadReader[Dialogue](reader, asset.FormatJSON)
if err != nil {
	return err
}
```

Use `WithID` when you deliberately want a stable logical cache identity, such as for generated data:

```go
data, err := server.Load[Dialogue](
	"generated/dialogue.json",
	asset.WithID("dialogue:intro"),
)
```

An overridden ID is part of correctness, not just an optimization. Two different files with the same ID, target type, and format share the first cached value.

#### Loading audio

Load decoded audio as `asset.AudioData` after `ebitrun.New` has registered the audio codecs:

```go
if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData]("assets/audio/confirm.ogg"); err != nil {
	return err
}
```

For normal playback, prefer the audio component's `Load: audio.LoadEager` for short effects that play often. The decoded PCM is cached and can be shared by multiple players. Use `audio.LoadStream` for long music; the [audio guide](audio.md) covers that choice and the playback component.

#### Loading images

Load images as `asset.TextureData`. The asset package decodes them into backend-free Go image values; the runner converts them to a renderable texture when it needs to draw them:

```go
texture, err := g.World().MustResource[*asset.Server]().Load[asset.TextureData]("assets/sprites/hull.png")
if err != nil {
	return err
}
_ = texture.Image
```

### Streaming assets

Use `Open` when a consumer needs the raw file instead of a cached decoded value. `Open` returns the file and the format inferred from its extension. You own the returned file and must close it:

```go
file, format, err := server.Open("assets/audio/music.ogg")
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
source := audio.NewSource("assets/audio/music.ogg")
source.Load = audio.LoadStream
source.Loop = audio.LoopForever
```

Use streaming for long tracks that should not occupy memory as one decoded PCM buffer. Streaming is still lazy: missing files, unsupported formats, and decoder errors appear when the provider creates the play.

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

server := g.World().MustResource[*asset.Server]()
if err := server.RegisterDecoder(
	asset.FormatJSON,
	asset.Decoder[Dialogue](decodeDialogue),
	false,
); err != nil {
	return err
}

dialogue, err := server.Load[Dialogue]("assets/dialogue/intro.json")
if err != nil {
	return err
}
```

The `override` argument defaults to `false`, so duplicate registrations fail. Set it to `true` only when deliberately replacing the decoder for the same target type and format.

For a custom extension, define a format and override inference when loading:

```go
type LevelData struct {
	Name string
}

levelFormat := asset.Format("level")
if err := server.RegisterDecoder(
	levelFormat,
	asset.Decoder[LevelData](decodeLevel),
	false,
); err != nil {
	return err
}

level, err := server.Load[LevelData](
	"assets/world.bin",
	asset.WithFormat(levelFormat),
)
```

## Atlases

Treat an atlas as setup for image loading: registration loads the image through the normal typed cache, builds the named regions, validates their bounds, and stores the result for sprites to resolve later.

For a regular grid, register the image with tile dimensions and a prefix. Regions are named in row-major order as `prefix_0`, `prefix_1`, and so on:

```go
if err := g.World().MustResource[*asset.Server]().RegisterGridAtlas(
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
if err := server.RegisterAtlasFromSidecar(
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
