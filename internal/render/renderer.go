package render

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
)

type renderItem struct {
	entityID   core.EntityID
	renderable components.Renderable
	transform  components.Transform
}

// TextureLoader is the texture-loading behavior Renderer depends on. Defined
// here (the consumer) rather than in the assets package, so Renderer only
// couples to the behavior it needs, not the full Assets struct.
type TextureLoader interface {
	Load(path string) (*assets.Texture, error)
}

type Renderer struct {
	textures  TextureLoader
	Primitive *PrimitiveRenderer
}

func New(textures TextureLoader) *Renderer {
	return &Renderer{
		textures:  textures,
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
	renderItems := make([]renderItem, 0)
	// Get the camera's visible world-space bounds for frustum culling
	viewportBounds := camera.ViewportBounds()

	for entry := range world.NewQuery().WithRequiredComponents(components.Renderable{}, components.Transform{}).Execute() {
		if !entry.Entity.IsAlive() {
			continue
		}
		renderable := entry.Get[components.Renderable]()
		transform := entry.Get[components.Transform]()

		entityBounds := geom.NewRect(
			geom.Vector2{X: transform.Position.X - transform.Scale.X, Y: transform.Position.Y - transform.Scale.Y},
			geom.Vector2{X: transform.Position.X + transform.Scale.X, Y: transform.Position.Y + transform.Scale.Y},
		)
		if !viewportBounds.Intersects(entityBounds) {
			continue
		}

		renderItems = append(renderItems, renderItem{
			entityID:   entry.EntityID,
			renderable: renderable,
			transform:  transform,
		})
	}

	// Sort by layer > render depth > Y position (entityId too unstable)
	slices.SortStableFunc(renderItems, func(a, b renderItem) int {
		if a.renderable.Layer != b.renderable.Layer {
			return int(a.renderable.Layer) - int(b.renderable.Layer)
		}
		// render depth comparison
		if a.renderable.Depth != b.renderable.Depth {
			return int(a.renderable.Depth) - int(b.renderable.Depth)
		}
		// can't use direct subtraction for float comparison, so we use conditional checks
		if a.transform.Position.Y != b.transform.Position.Y {
			if a.transform.Position.Y < b.transform.Position.Y {
				return -1
			}
			return 1
		}
		return 0
	})

	// Render
	for _, item := range renderItems {
		if item.renderable.TexturePath != "" {
			r.drawSprite(screen, camera, item.transform, item.renderable)
		} else {
			r.Primitive.Draw(screen, camera, item.transform, item.renderable)
		}
	}
}

func (r *Renderer) DrawDebugInfo(screen *ebiten.Image, camera *Camera, world *core.World) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.1f\nTPS: %0.1f\nCamera Position: %v\n", ebiten.ActualFPS(), ebiten.ActualTPS(), camera.Position))
}

func (r *Renderer) drawSprite(screen *ebiten.Image, camera *Camera, transform components.Transform, renderable components.Renderable) {
	tx, err := r.textures.Load(renderable.TexturePath)
	if err != nil {
		return // silently skip entities with missing textures
	}

	frameW, frameH := tx.Width, tx.Height
	screenPos := camera.WorldToScreen(transform.Position)

	op := &ebiten.DrawImageOptions{}
	// first we set the position of the sprite on the screen by updating the DrawImageOptions
	op.GeoM.Translate(-float64(frameW)/2, -float64(frameH)/2)
	// Apply scaling (including camera zoom), rotation, and other transforms from Transform
	scaleX := transform.Scale.X * camera.Zoom
	scaleY := transform.Scale.Y * camera.Zoom
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	cr, cg, cb, ca := colorOrDefault(transform.Color).RGBA()
	op.ColorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	// Finally, draw the sprite's texture onto the screen using the options
	screen.DrawImage(tx.Image, op)
}
