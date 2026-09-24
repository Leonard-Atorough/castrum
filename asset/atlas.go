package asset

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"sync"
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

// AtlasID identifies a registered atlas within the [Server]'s store.
type AtlasID string

// AtlasRegion is a named rectangular region of an atlas texture, in
// pixels.
type AtlasRegion struct {
	X, Y, W, H int
}

// Rect returns the region as a pixel rectangle — the value a runner
// feeds to its sub-image call.
func (r AtlasRegion) Rect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

// Atlas is a texture subdivided into named pixel regions, validated
// against the dimensions of the texture it names. Immutable after
// construction; the pairing of regions to texture is enforceable because
// the registration verbs derive the region set and the dimensions from
// the same loaded texture.
type Atlas struct {
	id                 AtlasID
	texturePath        ID
	regions            map[string]AtlasRegion
	textureW, textureH int
}

// ID returns the atlas's identifier.
func (a *Atlas) ID() AtlasID {
	return a.id
}

// TexturePath returns the texture the atlas's regions index into.
func (a *Atlas) TexturePath() ID {
	return a.texturePath
}

// TextureW returns the atlas texture's width in pixels.
func (a *Atlas) TextureW() int {
	return a.textureW
}

// TextureH returns the atlas texture's height in pixels.
func (a *Atlas) TextureH() int {
	return a.textureH
}

// Region resolves a named region. It errors, naming the region and the
// atlas, if no such region exists.
func (a *Atlas) Region(name string) (AtlasRegion, error) {
	region, ok := a.regions[name]
	if !ok {
		return AtlasRegion{}, fmt.Errorf("region %q not found in atlas %q", name, a.id)
	}
	return region, nil
}

// NewAtlas builds an Atlas from explicit fields, validating every region
// against the texture dimensions and copying the regions map. It is the
// construction seam for tests, tooling, and hand-authored atlases; games
// register through [Server.RegisterAtlasFromSidecar] and
// [Server.RegisterGridAtlas] instead, where the texture path and the
// validated dimensions come from the same loaded texture.
func NewAtlas(id AtlasID, texturePath ID, texW, texH int, regions map[string]AtlasRegion) (*Atlas, error) {
	if id == "" {
		return nil, fmt.Errorf("atlas id must not be empty")
	}
	if texturePath == "" {
		return nil, fmt.Errorf("texture path for atlas %q must not be empty", id)
	}
	if texW <= 0 || texH <= 0 {
		return nil, fmt.Errorf("texture dimensions %dx%d for atlas %q are not positive", texW, texH, id)
	}
	copied := make(map[string]AtlasRegion, len(regions))
	for name, region := range regions {
		if name == "" {
			return nil, fmt.Errorf("region name in atlas %q must not be empty", id)
		}
		if region.W <= 0 || region.H <= 0 {
			return nil, fmt.Errorf("region %q in atlas %q has dimensions %dx%d, want positive",
				name, id, region.W, region.H)
		}
		if region.X < 0 || region.Y < 0 || region.X+region.W > texW || region.Y+region.H > texH {
			return nil, fmt.Errorf("region %q in atlas %q (%d,%d %dx%d) falls outside texture bounds %dx%d",
				name, id, region.X, region.Y, region.W, region.H, texW, texH)
		}
		copied[name] = region
	}
	return &Atlas{
		id:          id,
		texturePath: texturePath,
		regions:     copied,
		textureW:    texW,
		textureH:    texH,
	}, nil
}

// AtlasStore resolves atlas handles. Exactly one store exists per
// [Asset]: there is no constructor outside the package, and atlases
// enter only through [Server.RegisterAtlasFromSidecar],
// [Server.RegisterGridAtlas], or [AtlasStore.Register].
type AtlasStore struct {
	mu      sync.RWMutex
	atlases map[AtlasID]*Atlas
}

func newStore() *AtlasStore {
	return &AtlasStore{atlases: make(map[AtlasID]*Atlas)}
}

// Register adds an atlas to the store. An atlas whose ID is already
// registered is an error.
func (s *AtlasStore) Register(atlas *Atlas) error {
	if atlas == nil {
		return fmt.Errorf("cannot register a nil atlas")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := atlas.ID()
	if _, exists := s.atlases[id]; exists {
		return fmt.Errorf("atlas %q is already registered", id)
	}
	s.atlases[id] = atlas
	return nil
}

// Atlas resolves a registered atlas. It errors, naming the ID, if no
// atlas is registered under it.
func (s *AtlasStore) Atlas(id AtlasID) (*Atlas, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	atlas, ok := s.atlases[id]
	if !ok {
		return nil, fmt.Errorf("atlas %q is not registered", id)
	}
	return atlas, nil
}

// Region resolves a named region of a registered atlas in one lookup.
func (s *AtlasStore) Region(atlasID AtlasID, regionName string) (AtlasRegion, error) {
	atlas, err := s.Atlas(atlasID)
	if err != nil {
		return AtlasRegion{}, err
	}
	return atlas.Region(regionName)
}

// RegisterAtlasFromSidecar loads the texture and its JSON sidecar through
// the unified load flow, validates every region against the loaded
// texture's dimensions, and registers the atlas. The texture path and
// the validated dimensions come from the same load, so region-to-texture
// pairing holds by construction. Failure at any step errors and leaves
// the store untouched.
func (a *Server) RegisterAtlasFromSidecar(id AtlasID, texturePath, metaPath string) error {
	texture, err := a.Load[TextureData](texturePath)
	if err != nil {
		return err
	}
	meta, err := a.Load[AtlasMeta](metaPath)
	if err != nil {
		return err
	}

	regions := make(map[string]AtlasRegion, len(meta.Regions))
	for _, r := range meta.Regions {
		if _, exists := regions[r.Name]; exists {
			return fmt.Errorf("region %q is defined twice in %s", r.Name, metaPath)
		}
		regions[r.Name] = AtlasRegion{X: r.X, Y: r.Y, W: r.W, H: r.H}
	}

	atlas, err := NewAtlas(id, ID(texturePath), texture.Width, texture.Height, regions)
	if err != nil {
		return err
	}
	return a.atlasStore.Register(atlas)
}

// RegisterGridAtlas loads the texture and divides it into equally sized
// tiles named prefix_0.. in row-major order, then registers the atlas.
// The texture dimensions must be evenly divisible by the tile
// dimensions.
func (a *Server) RegisterGridAtlas(id AtlasID, texturePath string, tileW, tileH int, prefix string) error {
	if tileW <= 0 || tileH <= 0 {
		return fmt.Errorf("tile dimensions %dx%d for atlas %q are not positive", tileW, tileH, id)
	}
	if prefix == "" {
		return fmt.Errorf("tile prefix for atlas %q must not be empty", id)
	}
	texture, err := a.Load[TextureData](texturePath)
	if err != nil {
		return err
	}
	if texture.Width%tileW != 0 || texture.Height%tileH != 0 {
		return fmt.Errorf("texture dimensions %dx%d for atlas %q are not evenly divisible by tile dimensions %dx%d",
			texture.Width, texture.Height, id, tileW, tileH)
	}

	rows := texture.Height / tileH
	cols := texture.Width / tileW
	regions := make(map[string]AtlasRegion, rows*cols)
	idx := 0
	for y := range rows {
		for x := range cols {
			regions[fmt.Sprintf("%s_%d", prefix, idx)] = AtlasRegion{
				X: x * tileW, Y: y * tileH, W: tileW, H: tileH,
			}
			idx++
		}
	}

	atlas, err := NewAtlas(id, ID(texturePath), texture.Width, texture.Height, regions)
	if err != nil {
		return err
	}
	return a.atlasStore.Register(atlas)
}
