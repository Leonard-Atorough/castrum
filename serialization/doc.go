// Package serialization joins data and entities: a registry of the component
// and resource types that participate in data serialization and saves. Types
// are registered explicitly - registered means participating, and data
// referencing an unknown key fails loud with the key named.
//
// The registry resolves the game's asset codecs from the world lazily, so
// codec configuration installed after the registry is created participates
// from that point on. Registering a component also installs JSON for it on
// the game's codecs, making the type loadable and saveable through the
// asset system wherever the codecs are used.
package serialization
