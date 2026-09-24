package asset

import (
	"encoding/json"
	"io"
)

// AtlasMeta is the JSON sidecar describing the named regions of one
// atlas image.
type AtlasMeta struct {
	Regions []AtlasRegionMeta `json:"regions"`
}

// AtlasRegionMeta describes one named region of an atlas image in pixel
// coordinates, as written in the sidecar file.
type AtlasRegionMeta struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

// decodeAtlasMeta decodes the JSON sidecar. Registered by default; see
// defaults.go.
func decodeAtlasMeta(reader io.Reader) (AtlasMeta, error) {
	var meta AtlasMeta
	err := json.NewDecoder(reader).Decode(&meta)
	return meta, err
}

type AtlasID string

type AtlasRegion struct {
	X, Y, W, H int
}

type Atlas struct {
	id                 AtlasID
	texturePath        ID
	regions            map[string]AtlasRegion
	textureW, textureH int
}

type AtlasStore struct {
	atlases map[AtlasID]*Atlas
}

func NewStore() *AtlasStore {
	return &AtlasStore{
		atlases: make(map[AtlasID]*Atlas),
	}
}

func (s *AtlasStore) Atlas(id AtlasID) (*Atlas, bool) {
	atlas, ok := s.atlases[id]
	return atlas, ok
}

func (s *AtlasStore) Region(atlasID AtlasID, regionName string) (*AtlasRegion, bool) {
	atlas, ok := s.atlases[atlasID]
	if !ok {
		return nil, false
	}
	region, ok := atlas.regions[regionName]
	if !ok {
		return nil, false
	}
	return &region, true
}
