package atlas

import (
	"maps"

	"github.com/leonard-atorough/castrum/geom"
)

// AtlasRegion describes a named rectangular area within a texture atlas,
// in pixel coordinates relative to the texture's top-left corner.
type AtlasRegion struct {
	Name       string
	X, Y, W, H int
}

// Bounds returns the region as a [geom.Rect].
func (a *AtlasRegion) Bounds() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: float64(a.X), Y: float64(a.Y)},
		Max: geom.Vector2{X: float64(a.X + a.W), Y: float64(a.Y + a.H)},
	}
}

// ID identifies an atlas within the atlas store.
type ID string

// Atlas is a texture atlas: a single texture image divided into named
// regions. Atlases are created via [Builder.Build] or [NewTextureAtlas].
type Atlas struct {
	id         ID
	assetID    string
	texW, texH int
	regions    map[string]AtlasRegion
}

// NewTextureAtlas creates an Atlas with a defensive copy of the given
// regions map. The atlas is not registered with any store.
func NewTextureAtlas(id, assetID string, texW, texH int, regions map[string]AtlasRegion) *Atlas {
	copied := make(map[string]AtlasRegion, len(regions))
	maps.Copy(copied, regions)
	return &Atlas{
		id:      ID(id),
		assetID: assetID,
		texW:    texW,
		texH:    texH,
		regions: copied,
	}
}

// ID returns the atlas identifier.
func (a *Atlas) ID() ID {
	return a.id
}

// AssetID returns the texture asset path the atlas is built from.
func (a *Atlas) AssetID() string {
	return a.assetID
}

// Dimensions returns the texture width and height in pixels.
func (a *Atlas) Dimensions() (int, int) {
	return a.texW, a.texH
}

// Region returns the named region, or ok=false if no such region exists.
func (a *Atlas) Region(name string) (AtlasRegion, bool) {
	region, exists := a.regions[name]
	return region, exists
}

// Regions returns a copy of all regions. Mutating the returned map does
// not affect the atlas.
func (a *Atlas) Regions() map[string]AtlasRegion {
	copied := make(map[string]AtlasRegion, len(a.regions))
	maps.Copy(copied, a.regions)
	return copied
}
