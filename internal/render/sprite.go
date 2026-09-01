package render

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/texture"
)

type Sprite struct {
	Transform components.Transform
	Texture   *texture.Texture
	Visible   bool
}
