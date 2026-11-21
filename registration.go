package castrum

import (
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/serialization"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// Save keys identify engine components on disk. Changing a key breaks
// existing save files.
const (
	// TransformKey identifies [core.Transform] in save data.
	TransformKey = "core.Transform"
	// ColliderKey identifies [collision.Collider] in save data.
	ColliderKey = "collision.Collider"
	// SpriteKey identifies [sprite.Sprite] in save data.
	SpriteKey = "sprite.Sprite"
	// TextKey identifies [sprite.Text] in save data.
	TextKey = "sprite.Text"
	// PrimitiveKey identifies [sprite.Primitive] in save data.
	PrimitiveKey = "sprite.Primitive"
)

// Asset keys identify engine asset types in data files. Changing a key
// breaks existing data files.
const (
	// TextureAssetKey identifies [asset.TextureData] in asset declarations.
	TextureAssetKey = "asset.TextureData"
	// FontAssetKey identifies [asset.FontData] in asset declarations.
	FontAssetKey = "asset.FontData"
	// SheetAssetKey identifies [asset.Sheet] in asset declarations.
	SheetAssetKey = "asset.Sheet"
	// AudioAssetKey identifies [asset.AudioData] in asset declarations.
	AudioAssetKey = "asset.AudioData"
)

func registerEngine(reg *serialization.Registry) error {
	if err := reg.Component[core.Transform](TransformKey); err != nil {
		return err
	}
	if err := reg.Component[collision.Collider](ColliderKey); err != nil {
		return err
	}
	if err := reg.Component[sprite.Sprite](SpriteKey); err != nil {
		return err
	}
	if err := reg.Component[sprite.Text](TextKey); err != nil {
		return err
	}
	return reg.Component[sprite.Primitive](PrimitiveKey)
}

func registerAssets(reg *serialization.Registry) error {
	if err := reg.Asset[asset.TextureData](TextureAssetKey); err != nil {
		return err
	}
	if err := reg.Asset[asset.FontData](FontAssetKey); err != nil {
		return err
	}
	if err := reg.Asset[asset.Sheet](SheetAssetKey); err != nil {
		return err
	}
	return reg.Asset[asset.AudioData](AudioAssetKey)
}
