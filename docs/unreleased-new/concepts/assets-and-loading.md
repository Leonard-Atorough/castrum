# How assets reach your game

An asset path names content in a filesystem. A loader resolves the name, selects a decoder for a Go type and format, and returns decoded data. A consumer such as the renderer can request that data when it needs it.

## Paths and storage

The game's loader reads from the filesystem selected at game creation. By default that is the process working directory; an embedded or other filesystem can replace it. Paths use forward slashes and are relative to that root, not to the Go source or to another asset document.

The filesystem and the codecs answer different questions: the filesystem supplies bytes; a decoder interprets them. Selecting a format does not itself install a decoder. [Load game assets](../guides/assets/loading.md) shows how to select a root and preload an image.

## When loading happens

Constructing a game or a file reference does not read every asset. An explicit load reads and decodes at the call site. A consumer can instead resolve a path later, so a missing file or decode error may surface during collection or playback. Preloading a required image during setup moves its filesystem and decode errors earlier.

Spritesheets have an additional boundary: the default JSON decoder validates sheet metadata without loading the image. Checking whether all rectangles fit requires the texture's actual bounds. [Draw and animate a spritesheet](../guides/rendering/spritesheets.md) shows how to check both the bounds and the frame names before play.

## Reuse and ownership

The default loader caches successful decoded values by cleaned path, exact requested Go type, and format. A later reuse load with the same identity uses that value. A failed load is not cached; a later load can retry. A bypass load reads independently and neither refreshes nor invalidates an existing cached value.

Treat cached sheet data and the region maps retained by memory references as read-only while they are shared. Saving a document does not invalidate loader or renderer caches. Use an appropriate fresh-read policy for mutable documents rather than assuming a saved file replaces a previously cached value.

For failure checks, see [Diagnose an asset or rendering failure](../guides/assets/troubleshooting.md). Exact loader and sheet contracts live with the [loader API](../../../asset/loader.go) and [sheet API](../../../asset/sheet.go).
