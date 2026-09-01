package texture

import "github.com/hajimehoshi/ebiten/v2"

type Texture struct {
	Path          string
	Image         *ebiten.Image
	Height, Width int
}
