package assets

import (
	"github.com/leonard-atorough/castrum/internal/blueprint"
	"github.com/leonard-atorough/castrum/internal/texture"
)

type Assets struct {
	Textures   *texture.Store
	Blueprints *blueprint.Store
}

func NewAssets() *Assets {
	return &Assets{
		Textures:   texture.NewStore(),
		Blueprints: blueprint.NewStore(),
	}
}

var GlobalAssets = NewAssets()
