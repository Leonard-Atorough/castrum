package atlas

import (
	"maps"

	"github.com/leonard-atorough/castrum/geom"
)

type AtlasRegion struct {
	Name       string
	X, Y, W, H int
}

func (a *AtlasRegion) Bounds() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: float64(a.X), Y: float64(a.Y)},
		Max: geom.Vector2{X: float64(a.X + a.W), Y: float64(a.Y + a.H)},
	}
}

type ID string

type Atlas struct {
	id         ID
	assetID    string
	texW, texH int
	regions    map[string]AtlasRegion
}

func NewTextureAtlas(id, assetID string, texW, texH int, regions map[string]AtlasRegion) *Atlas {
	return &Atlas{
		id:      ID(id),
		assetID: assetID,
		texW:    texW,
		texH:    texH,
		regions: regions,
	}
}

func (a *Atlas) ID() ID {
	return a.id
}

func (a *Atlas) AssetID() string {
	return a.assetID
}

func (a *Atlas) Dimensions() (int, int) {
	return a.texW, a.texH
}

func (a *Atlas) Region(name string) (AtlasRegion, bool) {
	region, exists := a.regions[name]
	return region, exists
}

func (a *Atlas) Regions() map[string]AtlasRegion {
	copied := make(map[string]AtlasRegion, len(a.regions))
	maps.Copy(copied, a.regions)
	return copied
}
