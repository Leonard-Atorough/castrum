package core

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/asset"
)

// Sprite represents the shared visual properties of a sprite.
//
// The zero value is the shown, opaque sprite: a declared sprite is
// drawn unless told otherwise. Hide one with Hidden; see through it
// with Transparency.
type Sprite struct {
	Layer     uint8 // 0-31
	SortOrder int8  // within layer, higher on top; Y-fallback below that
	// Hidden skips collection entirely. false — the zero value —
	// means shown.
	Hidden bool
	// FlipH and FlipV mirror the sprite horizontally and vertically.
	FlipH, FlipV bool
	// Transparency fades the sprite: 0.0 — the zero value — is fully
	// opaque, 1.0 is fully see-through. A faded-out sprite still
	// collects and blits; use Hidden to stop showing it.
	Transparency float32
	Tint         color.Color
}

func (s Sprite) Validate() error {
	if s.Transparency < 0 || s.Transparency > 1 {
		return fmt.Errorf("transparency must be between 0 and 1")
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
