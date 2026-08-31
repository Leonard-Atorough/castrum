package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Renderer struct {
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Clear(screen *ebiten.Image, c color.Color) {
	screen.Fill(c)
}

func (r *Renderer) DrawSprite(screen *ebiten.Image, sprite *Sprite, camera *Camera) {
	if !sprite.Visible {
		return
	}
	op := &ebiten.DrawImageOptions{}

	frameW := sprite.Texture.Width
	frameH := sprite.Texture.Height

	screenPos := camera.WorldToScreen(sprite.Transform.Position)

	// first we set the position of the sprite on the screen by updating the DrawImageOptions
	op.GeoM.Translate(-float64(frameW)/2, -float64(frameH)/2)
	// Apply scaling, rotation, and other transforms from Sprite
	op.GeoM.Scale(sprite.Transform.Scale.X, sprite.Transform.Scale.Y)
	op.GeoM.Rotate(sprite.Transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	op.ColorScale.Scale(float32(sprite.Transform.Color.(color.RGBA).R)/255, float32(sprite.Transform.Color.(color.RGBA).G)/255, float32(sprite.Transform.Color.(color.RGBA).B)/255, float32(sprite.Transform.Color.(color.RGBA).A)/255)

	// Finally, draw the sprite's texture onto the screen using the options
	screen.DrawImage(sprite.Texture.Image, op)
}
