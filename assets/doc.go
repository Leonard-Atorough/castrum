// Package assets provides loading, saving, and caching of game assets such as
// textures, atlas metadata, and entity blueprints.
//
// [NewAssets] is the entry point. It wires a [Loader] and [Saver] to a shared
// internal service so that saving an asset invalidates the loader's cache
// entry for the same ID.
//
// The [Loader] decodes assets from an [fs.FS] using format-specific decoders
// registered at construction time (PNG, JPEG, YAML, JSON). Custom decoders
// can be added via [Loader.RegisterDecoder]. Loaded assets are cached by ID
// and type; the cache can be invalidated per-ID or cleared entirely.
//
// The [Saver] encodes assets to the host filesystem or an [io.Writer] using
// registered encoders. By default, file saves are atomic (write to temp, then
// rename) and create missing directories.
//
// Blueprints are YAML documents describing an entity by name and component
// properties. [CreateFromBlueprint] instantiates an entity from a
// [Blueprint] using a [ComponentRegistry] that maps component type names to
// construction factories.
package assets
