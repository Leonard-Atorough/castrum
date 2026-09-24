package core

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/asset"
)

// Sprite represents the shared visual properties of a sprite.
type Sprite struct {
	Layer        uint8 // 0-31
	SortOrder    int8  // within layer, higher on top; Y-fallback below that
	Visible      bool
	FlipH, FlipV bool
	Opacity      float32 // 0..1
	Tint         color.Color
}

func (s Sprite) Validate() error {
	if s.Opacity < 0 || s.Opacity > 1 {
		return fmt.Errorf("opacity must be between 0 and 1")
	}
	if s.Layer > 31 {
		return fmt.Errorf("layer must be between 0 and 31")
	}
	return nil
}

// AtlasSprite represents a sprite sourced from an atlas.
type AtlasSprite struct {
	Atlas  asset.AtlasID
	Region string
}

func (a AtlasSprite) Validate() error {
	if a.Atlas == "" {
		return fmt.Errorf("atlas ID cannot be empty")
	}
	if a.Region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	return nil
}

// TextureSprite represents a sprite sourced from a standalone texture.
type TextureSprite struct {
	Texture asset.ID // the whole image
}

func (t TextureSprite) Validate() error {
	if t.Texture == "" {
		return fmt.Errorf("texture ID cannot be empty")
	}
	return nil
}
