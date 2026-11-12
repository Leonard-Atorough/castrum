# The asset pipeline

An asset ID is a slash-separated path resolved against the `fs.FS` owned by the world's `asset.Loader`. The default filesystem is the process working directory; `WithFilesystem` can supply `embed.FS` or another filesystem. Paths are normalized before loading.

## Typed loading and cache identity

`loader.Load[T]` selects a typed decoder, opens the path, decodes a value of type `T`, and caches it. The cache identity combines the asset ID, target type, and format. The same file can therefore have multiple typed representations, while two different files loaded with the same explicit ID, type, and format collide by design.

Repeated loads return the cached value. Concurrent loads of the same identity share one in-flight decode. `LoadReader` uses an explicitly supplied format but does not cache or deduplicate the result; it also does not close the caller's reader.

`Open` is the uncached path. It returns the raw file and inferred format, and the caller owns the file and must close it. A streaming consumer must also verify that the file supports seeking when looping or resetting a position; directory files and `embed.FS` do.

## Codec configuration and scopes

`asset.Codecs` configures decoders and encoders independently per exact target Go type and format. Formats normally come from lowercased file extensions. `SetDecoder` and `SetEncoder` install or replace their respective direction on the configuration. Format constants name vocabulary; they do not guarantee that a codec is configured.

`NewCodecs` supplies backend-free image/font decoders and a JSON pair for `AtlasMeta`. `SetJSON[T]` installs or replaces both directions atomically and uses standard JSON behavior, including custom marshal/unmarshal methods. A decode must contain exactly one JSON value followed by whitespace or EOF. Missing fields are zero-valued, not filled from game defaults. Validation and schema evolution remain outside the codec registry.

`NewLoader` and `NewSaver` take concrete `LoaderOptions` and `SaverOptions` values. Their `Codecs` field selects configuration to snapshot; nil selects the defaults. Subsequent configuration changes do not affect them. Their private scopes permit new local decoder/encoder registrations but not replacements; `RegisterDecoder` and `RegisterEncoder` reject duplicates. Snapshotting copies functions, not captured closure state. Constructors reject dependencies containing nil pointers, not valid nil-map or nil-slice implementations.

`castrum.WithAssetCodecs` configures the game's content loader while preserving the built-in filesystem overlay. Standalone facades can exist before game construction without replacing the world's content loader.

The default `ebitrun` runner registers WAV, MP3, and OGG audio decoders only on its game's loader, bound to the effective audio-context sample rate. Conflicting registrations return an error rather than being overwritten. Explicit audio loads must happen after `ebitrun.New`; runner registration does not modify the source configuration or standalone loaders.

A missing file, missing codec, or codec failure returns an `AssetError` describing the operation and path when known. Stream failures have no invented file identity. Underlying causes remain available through `errors.Is` and `errors.As`. Lazy loads surface errors at first use; preload required assets during setup when failure should prevent launch.

## Saving

`saver.Save` resolves an encoder before calling the configured `SaveStore`. It passes a synchronous encoding callback to storage, which owns its writer and publication semantics. A filesystem's read capability does not imply write capability; there is no built-in directory backend yet.

`SaveWriter` performs encoding only. Like `LoadReader`, it neither closes its caller-owned stream nor accesses storage. A saver with no store supports `SaveWriter` but returns an error for path saves.

The registry stores typed codec functions, recovering the exact `Decoder[T]` or `Encoder[T]` at lookup. Stream conversion keeps values typed throughout; only cached and in-flight load results need a typed value box. This preserves the static requested type even for nil interfaces and typed nil pointers. A decoder or encoder for one type cannot be selected solely because another type uses the same format.

Saves do not update or invalidate loader, GPU, or font caches. Re-reading mutable data currently uses `Open` and `LoadReader`. Games own snapshot copying, same-destination write ordering, versions, migration, validation, and application of loaded data.

## Atlases

The loader owns one atlas store. `RegisterGridAtlas` loads a texture, divides it into row-major tiles, and registers names such as `prefix_0`. `RegisterAtlasFromSidecar` loads irregular regions from a JSON sidecar. Registration validates the source and region bounds before storing the atlas.

A sprite refers to an atlas and region by ID. The collector resolves that pair when it collects a drawable, so an unknown atlas or region can fail on the first rendered frame even if the sprite component itself passed validation. Register atlases during setup to move predictable failures earlier.

## Audio paths

Eager audio drains the file into cached `asset.AudioData`, allowing players to share decoded samples. Streaming opens a source for each play and decodes it as it runs, avoiding one complete PCM buffer in the cache but requiring an open, seekable source for looping. The [audio guide](../guides/audio.md) covers the game-facing `audio.Source` choices.

## Where to go next

- [Assets](../guides/assets.md) - the game-facing loading workflow.
- [Publishing your game](../guides/publishing-your-game.md) - embedding the filesystem in a release.
- [Runner separation](runner-separation.md) - why backend decoders arrive at runner construction.
