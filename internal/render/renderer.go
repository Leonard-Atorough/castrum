package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
)

type Renderer struct {
	Assets    *assets.Assets
	Primitive *PrimitiveRenderer
}

func NewRenderer(assets *assets.Assets) *Renderer {
	return &Renderer{
		Assets:    assets,
		Primitive: NewPrimitiveRenderer(),
	}
}

func (r *Renderer) Clear(screen *ebiten.Image, c color.Color) {
	screen.Fill(c)
}

// DrawScene renders every entity with a Renderable+Transform. A Renderable
// with a TexturePath is drawn as a sprite; otherwise it's drawn as a
// primitive shape - callers never need to say which.
func (r *Renderer) DrawScene(screen *ebiten.Image, camera *Camera, world *core.World) {
	entities := world.Query(core.Types(components.Renderable{}, components.Transform{})...)

	for _, entityID := range entities {
		renderable, err := core.GetComponent[components.Renderable](world, entityID)
		if err != nil || !renderable.Visible {
			continue
		}
		transform, err := core.GetComponent[components.Transform](world, entityID)
		if err != nil {
			continue
		}

		if renderable.TexturePath != "" {
			r.drawSprite(screen, camera, transform, renderable)
		} else {
			r.Primitive.Draw(screen, camera, transform, renderable)
		}
	}
}

func (r *Renderer) drawSprite(screen *ebiten.Image, camera *Camera, transform components.Transform, renderable components.Renderable) {
	tx, err := r.Assets.Textures.GetTexture(renderable.TexturePath)
	if err != nil {
		return // silently skip entities with missing textures
	}

	frameW, frameH := tx.Width, tx.Height
	screenPos := camera.WorldToScreen(transform.Position)

	op := &ebiten.DrawImageOptions{}
	// first we set the position of the sprite on the screen by updating the DrawImageOptions
	op.GeoM.Translate(-float64(frameW)/2, -float64(frameH)/2)
	// Apply scaling, rotation, and other transforms from Transform
	op.GeoM.Scale(transform.Scale.X, transform.Scale.Y)
	op.GeoM.Rotate(transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	cr, cg, cb, ca := colorOrDefault(transform.Color).RGBA()
	op.ColorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	// Finally, draw the sprite's texture onto the screen using the options
	screen.DrawImage(tx.Image, op)
}
