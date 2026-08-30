package render

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Sprite struct {
	Transform
	Texture *Texture
	Visible bool
}

type Texture struct {
	Path          string
	Image         *ebiten.Image
	Height, Width int
}
