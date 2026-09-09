package atlas

import (
	"encoding/json"
	"fmt"
	"image"
	"io/fs"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// SubTexture represents a named region within a TextureAtlas.
type SubTexture struct {
	Name          string
	Image         *ebiten.Image
	Width, Height int
}

// TextureAtlas is a pure data struct holding a texture and its named regions.
// It does not handle loading or I/O; use Builder or Manager for that.
type TextureAtlas struct {
	// Identifier for this atlas (e.g., filename or custom ID)
	ID string

	// The full texture image
	Texture *ebiten.Image
	TexW    int // texture width
	TexH    int // texture height

	// Named regions within the texture
	Regions map[string]*SubTexture
}

// NewTextureAtlas creates a TextureAtlas from a pre-loaded ebiten.Image.
// Regions must be populated separately or via Builder.
func NewTextureAtlas(id string, texture *ebiten.Image) *TextureAtlas {
	return &TextureAtlas{
		ID:      id,
		Texture: texture,
		TexW:    texture.Bounds().Dx(),
		TexH:    texture.Bounds().Dy(),
		Regions: make(map[string]*SubTexture),
	}
}

type atlasStorer interface {
	store(id string, atlas *TextureAtlas)
}

// Builder constructs a TextureAtlas from a texture image.
// It supports grid-based slicing or metadata-based slicing.
type Builder struct {
	id      string
	texture *ebiten.Image
	regions map[string]*SubTexture
	// The store responsible for managing this atlas
	store atlasStorer
}

// NewBuilder creates a new atlas builder for the given texture.
func NewBuilder(id string, texture *ebiten.Image, store atlasStorer) *Builder {
	return &Builder{
		id:      id,
		texture: texture,
		regions: make(map[string]*SubTexture),
		store:   store,
	}
}

// SliceRegion manually adds a named region to the atlas.
func (b *Builder) SliceRegion(name string, x, y, w, h int) *Builder {
	rect := image.Rect(x, y, x+w, y+h)
	if !rect.In(b.texture.Bounds()) {
		// Silently skip out-of-bounds regions in builder (or return error in Build())
		return b
	}
	b.regions[name] = &SubTexture{
		Name:   name,
		Image:  b.texture.SubImage(rect).(*ebiten.Image),
		Width:  w,
		Height: h,
	}
	return b
}

// AutoSlice creates regions by dividing the texture into a grid.
// cols, rows: number of columns and rows in the grid
// frameW, frameH: width and height of each frame
// Optional nameFunc: customize region names (default: "0", "1", "2", ...)
func (b *Builder) AutoSlice(cols, rows, frameW, frameH int, nameFunc func(idx int) string) *Builder {
	if nameFunc == nil {
		nameFunc = func(idx int) string { return fmt.Sprintf("%d", idx) }
	}

	idx := 0
	for row := range rows {
		for col := range cols {
			x := col * frameW
			y := row * frameH
			name := nameFunc(idx)
			b.SliceRegion(name, x, y, frameW, frameH)
			idx++
		}
	}
	return b
}

// FromMetadata loads regions from a JSON metadata file (Aseprite/TexturePacker format).
// The JSON should have the structure: { "frames": { "name": { "frame": { "x", "y", "w", "h" } } } }
func (b *Builder) FromMetadata(filesystem fs.FS, metaPath string) error {
	file, err := filesystem.Open(metaPath)
	if err != nil {
		return fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	var def struct {
		Frames map[string]struct {
			Frame struct {
				X int `json:"x"`
				Y int `json:"y"`
				W int `json:"w"`
				H int `json:"h"`
			} `json:"frame"`
		} `json:"frames"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&def); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	for name, frameData := range def.Frames {
		f := frameData.Frame
		b.SliceRegion(name, f.X, f.Y, f.W, f.H)
	}
	return nil
}

// Build validates and returns the constructed TextureAtlas.
func (b *Builder) Build() (*TextureAtlas, error) {
	if b.texture == nil {
		return nil, fmt.Errorf("atlas %q: texture is required", b.id)
	}
	if len(b.regions) == 0 {
		return nil, fmt.Errorf("atlas %q: at least one region is required", b.id)
	}

	atlas := &TextureAtlas{
		ID:      b.id,
		Texture: b.texture,
		TexW:    b.texture.Bounds().Dx(),
		TexH:    b.texture.Bounds().Dy(),
		Regions: b.regions,
	}

	b.store.store(b.id, atlas)
	return atlas, nil
}

// AtlasStore orchestrates atlas creation and storage.
// It is lightweight: it stores atlases by ID and provides lookup.
type AtlasStore struct {
	mu      sync.RWMutex
	atlases map[string]*TextureAtlas
}

// NewAtlasStore creates a new AtlasStore.
func NewAtlasStore() *AtlasStore {
	return &AtlasStore{
		atlases: make(map[string]*TextureAtlas),
	}
}

// NewAtlas creates a new Builder for constructing an atlas with the given ID.
func (m *AtlasStore) NewAtlas(id string) *Builder {
	return &Builder{
		id:      id,
		store:   m,
		regions: make(map[string]*SubTexture),
	}
}

// Get retrieves an atlas by ID, or nil if not found.
func (m *AtlasStore) Get(id string) *TextureAtlas {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.atlases[id]
}

// Has checks if an atlas with the given ID exists.
func (m *AtlasStore) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.atlases[id]
	return ok
}

// Remove unregisters an atlas by ID.
func (m *AtlasStore) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.atlases, id)
}

func (m *AtlasStore) store(id string, atlas *TextureAtlas) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.atlases[id] = atlas
}
