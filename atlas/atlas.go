package atlas

import (
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
