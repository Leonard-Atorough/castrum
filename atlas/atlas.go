// Package atlas provides texture atlas types for sprite-sheet rendering.
//
// The primary way to build an atlas is through [castrum.Game.AtlasBuilder],
// which returns a [Builder] wired to the engine's internal atlas store.
// Calling [Builder.Build] registers the atlas so the texture provider can
// look up regions by name at render time.
//
// [NewTextureAtlas] constructs an [Atlas] directly without registering it
// with any store. It is useful when you need a standalone atlas for purposes
// that don't involve the texture provider, such as wiring animation clips.
package atlas

import (
	internalatlas "github.com/leonard-atorough/castrum/internal/atlas"
)

// Atlas is a texture atlas: a single texture image divided into named
// regions. Atlases are created via [Builder.Build] or [NewTextureAtlas].
type Atlas = internalatlas.Atlas

// AtlasRegion describes a rectangular area within a texture atlas in pixel
// coordinates.
type AtlasRegion = internalatlas.AtlasRegion

// ID identifies an atlas within the engine's atlas store.
type ID = internalatlas.ID

// Builder constructs an [Atlas] by slicing regions from a texture image.
// A Builder is obtained from [castrum.Game.AtlasBuilder]; there is no
// standalone constructor, because every Builder must be wired to a store
// that the texture provider can read from.
type Builder = internalatlas.Builder

// AtlasError is the error type returned by [Builder] methods.
type AtlasError = internalatlas.AtlasError

// AtlasErrorList is a collection of errors returned by [Builder.GridSlice]
// when multiple regions fail during a grid slice operation.
type AtlasErrorList = internalatlas.AtlasErrorList

// NewTextureAtlas creates an Atlas with the given regions map. The atlas is
// not registered with any store; use [castrum.Game.AtlasBuilder] when the
// atlas must be visible to the texture provider for rendering.
func NewTextureAtlas(id, assetID string, texW, texH int, regions map[string]AtlasRegion) *Atlas {
	return internalatlas.NewTextureAtlas(id, assetID, texW, texH, regions)
}
