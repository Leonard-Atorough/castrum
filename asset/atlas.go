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
	// Regions lists the named regions in the atlas image.
	Regions []AtlasRegionMeta `json:"regions"`
}

// AtlasRegionMeta describes one named region of an atlas image in pixel
// coordinates, as written in the sidecar file.
type AtlasRegionMeta struct {
	// Name is the region's unique name within the atlas.
	Name string `json:"name"`
	// X is the left coordinate in pixels.
	X int `json:"x"`
	// Y is the top coordinate in pixels.
	Y int `json:"y"`
	// W is the region's width in pixels.
	W int `json:"w"`
	// H is the region's height in pixels.
	H int `json:"h"`
}

func decodeAtlasMeta(reader io.Reader) (AtlasMeta, error) {
	var meta AtlasMeta
	err := json.NewDecoder(reader).Decode(&meta)
	return meta, err
}

// AtlasID identifies a registered atlas within the [Server].
type AtlasID string

// AtlasRegion is a named rectangular region of an atlas texture, in pixels.
type AtlasRegion struct {
	// X is the region's left coordinate.
	X int
	// Y is the region's top coordinate.
	Y int
	// W is the region's width.
	W int
	// H is the region's height.
	H int
}

// Rect returns the region's pixel bounds as an [image.Rectangle].
func (r AtlasRegion) Rect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

// Atlas associates named pixel regions with a texture. It is immutable
// after construction, and each region is validated against the texture
// bounds, so an Atlas is safe to share across goroutines.
type Atlas struct {
	id          AtlasID
	texturePath ID
	regions     map[string]AtlasRegion
}

// ID returns the atlas's identifier.
func (a *Atlas) ID() AtlasID {
	return a.id
}

// TexturePath returns the texture the atlas's regions index into.
func (a *Atlas) TexturePath() ID {
	return a.texturePath
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

// NewAtlas creates an atlas from explicit texture dimensions and regions. It
// rejects empty IDs, empty region maps, empty region names, non-positive
// texture or region dimensions, and regions outside the texture bounds. It
// copies regions so later changes to the input map do not affect the atlas.
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
	if len(regions) == 0 {
		return nil, fmt.Errorf("atlas %q must have at least one region", id)
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
	}, nil
}

// atlasRegistry holds the server's registered atlases. It is safe for
// concurrent use. Registration rejects duplicates, so a registered atlas's
// regions never change.
type atlasRegistry struct {
	mu      sync.RWMutex
	atlases map[AtlasID]*Atlas
}

func newAtlasRegistry() *atlasRegistry {
	return &atlasRegistry{atlases: make(map[AtlasID]*Atlas)}
}

func (r *atlasRegistry) register(atlas *Atlas) error {
	if atlas == nil {
		return fmt.Errorf("cannot register a nil atlas")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := atlas.ID()
	if _, exists := r.atlases[id]; exists {
		return fmt.Errorf("atlas %q is already registered", id)
	}
	r.atlases[id] = atlas
	return nil
}

func (r *atlasRegistry) atlas(id AtlasID) (*Atlas, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	atlas, ok := r.atlases[id]
	if !ok {
		return nil, fmt.Errorf("atlas %q is not registered", id)
	}
	return atlas, nil
}

// Atlas returns the atlas registered under id. It returns an error if id is
// not registered. Lookups are safe for concurrent use.
func (s *Server) Atlas(id AtlasID) (*Atlas, error) {
	return s.atlases.atlas(id)
}

// RegisterAtlas adds a programmatically-built atlas - one constructed with
// [NewAtlas] rather than the Register* constructors. It returns an error if
// atlas is nil or its ID is already registered.
func (s *Server) RegisterAtlas(atlas *Atlas) error {
	return s.atlases.register(atlas)
}

// RegisterAtlasFromSidecar loads the texture and JSON sidecar, validates the
// regions against the texture dimensions, and registers the atlas. If any
// step fails, the atlas is not registered.
func (s *Server) RegisterAtlasFromSidecar(id AtlasID, texturePath, metaPath string) error {
	texture, err := s.Load[TextureData](texturePath)
	if err != nil {
		return err
	}
	meta, err := s.Load[AtlasMeta](metaPath)
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
	return s.atlases.register(atlas)
}

// RegisterGridAtlas divides the texture into equally sized tiles named
// prefix_0, prefix_1, and so on in row-major order, then registers the atlas.
// Tile dimensions must be positive, prefix must be non-empty, and the texture
// dimensions must be evenly divisible by the tile dimensions.
func (s *Server) RegisterGridAtlas(id AtlasID, texturePath string, tileW, tileH int, prefix string) error {
	if tileW <= 0 || tileH <= 0 {
		return fmt.Errorf("tile dimensions %dx%d for atlas %q are not positive", tileW, tileH, id)
	}
	if prefix == "" {
		return fmt.Errorf("tile prefix for atlas %q must not be empty", id)
	}
	texture, err := s.Load[TextureData](texturePath)
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
	return s.atlases.register(atlas)
}
