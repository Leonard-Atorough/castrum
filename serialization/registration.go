package serialization

import (
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
)

// NOTE: Call! This function registers the built-in File codec for the engine.
func RegisterBuiltins(codecs *asset.Codecs) error {
	return codecs.SetJSON[File]()
}

const (
	TransformKey = "core.Transform"
	ColliderKey  = "collision.Collider"
)

// NOTE: Move to engine-specific registration file
func RegisterEngine(reg *Registry) error {
	if err := reg.Component[core.Transform](TransformKey); err != nil {
		return err
	}
	if err := reg.Component[collision.Collider](ColliderKey); err != nil {
		return err
	}
	return nil
}
