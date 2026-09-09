package atlas

import (
	"encoding/json"
	"fmt"
	"image"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
)

// Builder constructs a TextureAtlas from a texture image.
// It supports grid-based slicing or metadata-based slicing.
type Builder struct {
	id      string
	texture *ebiten.Image
	regions map[string]*SubTexture
}

// NewBuilder creates a new atlas builder for the given texture.
func NewBuilder(id string, texture *ebiten.Image) *Builder {
	return &Builder{
		id:      id,
		texture: texture,
		regions: make(map[string]*SubTexture),
	}
}

// AddRegion manually adds a named region to the atlas.
func (b *Builder) AddRegion(name string, x, y, w, h int) *Builder {
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

// GridSlice creates regions by dividing the texture into a grid.
// cols, rows: number of columns and rows in the grid
// frameW, frameH: width and height of each frame
// Optional nameFunc: customize region names (default: "0", "1", "2", ...)
func (b *Builder) GridSlice(cols, rows, frameW, frameH int, nameFunc func(idx int) string) *Builder {
	if nameFunc == nil {
		nameFunc = func(idx int) string { return fmt.Sprintf("%d", idx) }
	}

	idx := 0
	for row := range rows {
		for col := range cols {
			x := col * frameW
			y := row * frameH
			name := nameFunc(idx)
			b.AddRegion(name, x, y, frameW, frameH)
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
		b.AddRegion(name, f.X, f.Y, f.W, f.H)
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

	return &TextureAtlas{
		ID:      b.id,
		Texture: b.texture,
		TexW:    b.texture.Bounds().Dx(),
		TexH:    b.texture.Bounds().Dy(),
		Regions: b.regions,
	}, nil
}
