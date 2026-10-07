# The asset pipeline

An asset ID is a slash-separated path resolved against the `fs.FS` owned by the world's `asset.Server`. The default filesystem is the process working directory; `WithFilesystem` can supply `embed.FS` or another filesystem. Paths are normalized before loading.

## Typed loading and cache identity

`Server.Load[T]` opens the path, selects a decoder from the format, decodes a value of type `T`, and caches it. The cache identity combines the asset ID, target type, and format. The same file can therefore have multiple typed representations, while two different files loaded with the same explicit ID, type, and format collide by design.

Repeated loads return the cached value. Concurrent loads of the same identity share one in-flight decode. `LoadReader` uses an explicitly supplied format but does not cache or deduplicate the result; it also does not close the caller's reader.

`Open` is the uncached path. It returns the raw file and inferred format, and the caller owns the file and must close it. A streaming consumer must also verify that the file supports seeking when looping or resetting a position; directory files and `embed.FS` do.

## Decoders

Decoders are registered per target Go type and format. The format normally comes from the lowercased file extension. `RegisterDecoder` adds a decoder and rejects duplicates unless `override` is true. Format constants name supported vocabulary; they do not guarantee that a decoder is registered.

The asset package provides backend-free image and atlas decoding. The default `ebitrun` runner registers WAV, MP3, and OGG audio decoders when it is constructed, and binds decoded audio to its sample rate. Explicit audio loads must therefore happen after `ebitrun.New`.

A missing file, missing decoder, or decoder failure returns an `AssetError` describing the operation and asset. Lazy loads surface these errors at first use; preload required assets during setup when failure should prevent launch.

## Atlases

The server owns one atlas store. `RegisterGridAtlas` loads a texture, divides it into row-major tiles, and registers names such as `prefix_0`. `RegisterAtlasFromSidecar` loads irregular regions from a JSON sidecar. Registration validates the source and region bounds before storing the atlas.

A sprite refers to an atlas and region by ID. The collector resolves that pair when it collects a drawable, so an unknown atlas or region can fail on the first rendered frame even if the sprite component itself passed validation. Register atlases during setup to move predictable failures earlier.

## Audio paths

Eager audio drains the file into cached `asset.AudioData`, allowing players to share decoded samples. Streaming opens a source for each play and decodes it as it runs, avoiding one complete PCM buffer in the cache but requiring an open, seekable source for looping. The [audio guide](../guides/audio.md) covers the game-facing `audio.Source` choices.

## Where to go next

- [Assets](../guides/assets.md) - the game-facing loading workflow.
- [Publishing your game](../guides/publishing-your-game.md) - embedding the filesystem in a release.
- [Runner separation](runner-separation.md) - why backend decoders arrive at runner construction.
