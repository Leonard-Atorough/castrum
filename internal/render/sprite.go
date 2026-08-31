package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/components"
)

type Sprite struct {
	Transform components.Transform
	Texture   *Texture
	Visible   bool
}

type Texture struct {
	Path          string
	Image         *ebiten.Image
	Height, Width int
}
