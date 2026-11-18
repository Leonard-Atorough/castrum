package castrum

import (
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/serialization"
)

// Save keys identify engine components on disk. Changing a key breaks
// existing save files.
const (
	TransformKey = "core.Transform"
	ColliderKey  = "collision.Collider"
)

// registerEngine registers the engine's own components with reg.
func registerEngine(reg *serialization.Registry) error {
	if err := reg.Component[core.Transform](TransformKey); err != nil {
		return err
	}
	return reg.Component[collision.Collider](ColliderKey)
}
