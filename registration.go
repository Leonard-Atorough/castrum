package castrum

import (
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/serialization"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// Save keys identify engine components on disk. Changing a key breaks
// existing save files.
const (
	TransformKey = "core.Transform"
	ColliderKey  = "collision.Collider"
	SpriteKey    = "sprite.Sprite"
	TextKey      = "sprite.Text"
	PrimitiveKey = "sprite.Primitive"
)

// registerEngine registers the engine's own components with reg.
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
