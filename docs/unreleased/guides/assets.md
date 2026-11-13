# Assets

Use `asset` to load shipped content and persist game-defined documents without coupling either task to a renderer or a live world. Choose the storage and cache behavior first; codecs then determine how each Go type becomes bytes or a decoded value.

| What you need | Setup |
| --- | --- |
| Shared images, fonts, and other shipped content | The game's loader, with the default `CacheReuse` policy |
| Preferences, save snapshots, or other mutable documents | A `Directory`, a bypass-default loader, and a saver |
| Bytes from a network, buffer, or other stream | `LoadReader` or `SaveWriter` with an explicit format |

A loader reads from an `fs.FS`; a saver writes through a `SaveStore`. `Directory` implements both interfaces for a host directory. `Codecs` configures conversion, not storage: sharing codecs does not connect two destinations or share their caches. Keep these objects for the lifetime of the work they serve rather than reconstructing them for each operation.

## Load shipped content

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

The pipeline supports these type and format combinations:

| Go type | Formats | Availability |
| --- | --- | --- |
| `asset.TextureData` | PNG, JPG, JPEG | Default loader |
| `asset.FontData` | TTF, OTF | Default loader |
| `asset.Sheet` | JSON | Default loader and saver |
| `asset.AudioData` | WAV, MP3, OGG | Game loader after `ebitrun.New` |
| Game-defined types | JSON or custom formats | Explicit codec registration |

Textures and fonts have default decoders, not default encoders. Sheets have a default JSON pair; game-defined JSON types require registration. A constant such as `asset.FormatYAML` names a format; it does not install a codec. Registrations match the exact Go type: a codec for `Profile` is not a codec for `*Profile` or every JSON document.

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

The cache key combines the cleaned source path, exact Go type, and format within one loader. Equivalent paths such as `./assets/sprites/hull.png` and `assets/sprites/hull.png` share a cached value; different paths or loaders do not.

## Draw regions from a spritesheet

An `asset.Sheet` describes one texture and its named pixel rectangles. Its default JSON decoder validates metadata before caching; actual image bounds require the texture. Loading and saving use the value type `asset.Sheet`, and file or memory references let sprites and clips share it without a registry.

See [Spritesheets](spritesheets.md) for the complete authoring, saving, loading, drawing, and animation workflow, including [setup-time validation](spritesheets.md#choose-when-validation-happens). Generic storage and cache choices remain in this guide.

## Configure game-defined data

Use `asset.Codecs` to configure conversion once and reuse it across loaders and savers. Start with `castrum.NewAssetCodecs` to retain image/font decoders, the sheet JSON pair, and core preference codecs, then register your own types before constructing the game. Use `ebitrun.NewAssetCodecs` when you also need display preferences; `asset.NewCodecs` remains the engine-independent alternative.

```go
type Profile struct {
	Name string `json:"name"`
}

codecs := castrum.NewAssetCodecs()
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

## Save and reload a document

For writable desktop storage, choose an application-owned directory and open it with `asset.NewDirectory`. The constructor opens an existing directory; your setup code decides where it belongs and creates it if needed. Castrum does not infer a preference or save location from the working directory, game title, or settings group.

This complete operation writes a supplied snapshot and reads it back through the same directory. It uses the `Profile` type above:

```go
import (
	"errors"
	"os"

	"github.com/Leonard-Atorough/castrum/asset"
)

func saveAndReloadProfile(root string, next Profile) (profile Profile, err error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return profile, err
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		return profile, err
	}
	defer func() {
		err = errors.Join(err, directory.Close())
	}()

	codecs := asset.NewCodecs()
	if err := codecs.SetJSON[Profile](); err != nil {
		return profile, err
	}
	dataLoader, err := asset.NewLoader(directory, asset.LoaderOptions{
		Codecs:      codecs,
		CachePolicy: asset.CacheBypass,
	})
	if err != nil {
		return profile, err
	}
	saver, err := asset.NewSaver(directory, asset.SaverOptions{Codecs: codecs})
	if err != nil {
		return profile, err
	}
	if err := saver.Save("profiles/player.json", next); err != nil {
		return profile, err
	}
	return dataLoader.Load[Profile]("profiles/player.json")
}
```

The loader and saver are connected by the same `Directory`, not merely the same codecs. `Save` selects the `Profile` encoder by its static Go type and the `.json` extension; the saver gives storage a callback already bound to that value and encoder. Storage supplies the writer, checks completion, and publishes the file. The loader independently selects the corresponding decoder.

For a running game, keep the directory and facades open rather than calling this setup function for every save. Close the directory only after all users and open files have finished. Neither the loader nor saver closes borrowed storage. Handle the close error when ending its lifetime, as the named return does above.

A standalone loader can restore configuration before game or runner construction. `LoaderOptions{}` selects default codecs; a nil filesystem reads from the working directory. Standalone loaders do not inherit the game's built-in overlay or runner-installed audio decoders.

### Restore before applying or saving

The example deliberately saves a new snapshot. For startup restoration, load first instead. Only `errors.Is(err, fs.ErrNotExist)` should select first-run defaults. Malformed JSON, unsupported versions, validation failures, and other I/O errors require an explicit recovery decision; do not silently replace a bad document with defaults.

Persist preference values or game-defined snapshots, not `settings.Group`, runtime providers, or a live world. Your game owns migration, validation, applying restored values, and reconstructing gameplay. [Settings groups](settings.md) provide a copying and validation boundary, not a save-location policy.

### Directory guarantees and limits

`Directory.Write` creates missing parent directories and a uniquely named staging file in the destination's directory. It invokes the encoding callback synchronously, checks encoding and close errors, then replaces the destination using the platform's rename operation.

- An encoding, staging-write, or close failure before publication leaves the previous destination untouched. Cleanup is attempted, and cleanup errors remain visible alongside the primary cause. Newly created parent directories may remain after a failed save.
- Staging files request permissions `0o600`; new parent directories request `0o700`. Platform permissions and ACL rules still apply. Existing parent permissions are not changed.
- Replacement errors are returned. The store never deletes the destination as a fallback and never truncates it before encoding.
- This is staged replacement, not a promise of universal atomicity or power-loss durability. Go does not guarantee atomic rename on non-Unix platforms, including Windows. A publication error does not establish an unconditional rollback guarantee.
- Rooted operations prevent paths and symlinks from escaping the supplied root on supported desktop platforms. This host-directory backend is not browser or cloud persistence.

Asset names remain slash-separated relative paths on every platform, including Windows. `Saver.Save` cleans names before storage; direct `Directory.Write` calls require valid `fs.ValidPath` names other than `"."`, with no backslashes. Absolute, escaping, empty, and non-canonical direct-write names are rejected. The root passed to `NewDirectory` is instead a native OS directory path.

Concurrent operations are supported, but writes to the same destination are not ordered by invocation time. Capture an owned snapshot before handing data to a worker and serialize saves when the newest snapshot must win. Do not close the directory while operations are active.

## Choose cached content or fresh reads

Saving does not invalidate decoded assets, GPU textures, or parsed runner fonts. Keep `CacheReuse` for shared content and use `CacheBypass` for mutable data, as in the directory example.

| Policy | Read/populate decoded cache | Share in-flight reads |
| --- | --- | --- |
| `CacheReuse` (zero/default) | Yes | Yes |
| `CacheBypass` | No | No |

`LoaderOptions.CachePolicy` chooses a default. Override either default for one call with `WithCachePolicy`:

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

Invalid policies and nil options return errors before opening a file. Both policies close files they open; failed decoding or closing returns no value and is never cached. If decoding and closing both fail, the returned error preserves both causes.

Bypass is not cache refresh or hot reload. A previously started cached read can still finish and populate the reused cache with its own result.

## Read streams without caching

Use `LoadReader[T]` for bytes from somewhere other than the loader's filesystem. For fresh filesystem reads, prefer `Load` with `CacheBypass`. Supply the stream format explicitly because a reader has no extension:

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

## Write to an existing stream

For a caller-owned writer, construct a stream-only saver with a nil store and call `SaveWriter`:

```go
import "bytes"

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

## Supply another storage backend

Implement `SaveStore` when documents belong somewhere other than a host directory:

```go
type SaveStore interface {
	Write(name string, encode func(io.Writer) error) error
}
```

The store receives a name and a byte-producing callback, not a codec registry or a typed value. It must invoke the callback synchronously at most once, invoke it before returning success, and never retain it. The store owns its writer and must not publish incomplete output on encoding failure. It reports write, close, and publication errors; its replacement and durability guarantees must be explicit.

For example, an in-memory implementation can encode into a private `bytes.Buffer` and replace its map entry only after success. An object-store adapter can stage an upload according to its own publication semantics. Neither needs to understand JSON or the Go type being saved. Read capability is separate: implement `fs.FS` too if the same destination should be usable by a loader.

Missing encoders and invalid save names fail before storage is invoked. `WithFormat` overrides extension inference for `Save`, just as it does for `Load`. Custom encoders and storage must be safe for concurrent use if called concurrently.

## Handle failures at the right time

Path and stream load/save operations return errors containing `*asset.AssetError`; underlying causes remain available through `errors.Is` and `errors.As`. `Load` reports `Load`, `Decode`, or `Close`; `Open` reports `Open`; path saves report `Save`, with encoding failures retaining their `Encode` cause. `LoadReader` and `SaveWriter` report `Decode` and `Encode` without inventing a file path. Constructors and registration methods can return ordinary errors instead.

Choose when predictable failures should surface:

- Preload required images, fonts, configuration, and sheets during setup; validate sheet bounds when every region must be checked.
- Construct the runner before preloading required audio effects.
- Load optional assets on demand when the first-use cost and failure timing are acceptable.
- Stream long audio when retaining complete decoded PCM would be wasteful, while accepting lazy playback errors.
- Keep error context when wrapping failures so the path and operation remain visible.

## Where to go next

- [Spritesheets](spritesheets.md) - author regions, save and load sheets, and animate sprites.
- [Rendering](rendering.md) - drawables that consume textures and sheet regions.
- [Audio](audio.md) - eager and streaming playback, buses, and the mixer.
- [Publishing your game](publishing-your-game.md) - embedding assets in a single binary.
- [The asset pipeline](../reference/asset-pipeline.md) - the implementation-level loading and caching model.
