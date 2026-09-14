// Package atlas re-exports atlas types from the internal atlas package.
// The Builder is only obtainable through internal/atlas.Service.NewBuilder,
// which ensures every Builder is wired to a store that the texture provider
// can read from. There is no standalone NewBuilder function here.
package atlas

import (
	internalatlas "github.com/leonard-atorough/castrum/internal/atlas"
)

type Atlas = internalatlas.Atlas
type AtlasRegion = internalatlas.AtlasRegion
type ID = internalatlas.ID
type Builder = internalatlas.Builder
type AtlasError = internalatlas.AtlasError
type AtlasErrorList = internalatlas.AtlasErrorList

// NewTextureAtlas creates an Atlas with the given regions map.
// In most cases you should use internal/atlas.Service.NewBuilder instead,
// which ensures the atlas is registered with a store.
func NewTextureAtlas(id, assetID string, texW, texH int, regions map[string]AtlasRegion) *Atlas {
	return internalatlas.NewTextureAtlas(id, assetID, texW, texH, regions)
}
