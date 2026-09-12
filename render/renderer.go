package render

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

type renderItem struct {
	entityID   ecs.EntityID
	renderable components.Sprite
	transform  components.Transform
	animation  *components.Animation
}

// TextureLoader is the interface for loading individual textures.
// Defined here (the consumer) rather than in the assets package, so Renderer
// only couples to the behavior it needs.
type TextureLoader interface {
	Load(path string) (*assets.Texture, error)
}

type Renderer struct {
	textureLoader TextureLoader
	clipStore     *animation.AnimationClipStore
	Primitive     *PrimitiveRenderer
	cameraQuery   *ecs.Query
	renderItems   []renderItem // Reusable buffer for hot path
}

func New(textureLoader TextureLoader) *Renderer {
	return &Renderer{
		textureLoader: textureLoader,
		Primitive:     NewPrimitiveRenderer(),
	}
}

func (r *Renderer) Clear(screen *ebiten.Image, c color.Color) {
	screen.Fill(c)
}

// DrawScene renders every entity with a Renderable+Transform. A Renderable
// with a TexturePath is drawn as a sprite; otherwise it's drawn as a
// primitive shape - callers never need to say which.
// The primary camera is queried from the world.
func (r *Renderer) DrawScene(screen *ebiten.Image, world *ecs.World) {
	// Query for the primary camera
	var primaryCamera components.Camera
	var cameraFound bool

	if r.cameraQuery == nil {
		r.cameraQuery = world.NewQuery().WithRequiredComponents(components.Camera{})
	}
	for result := range r.cameraQuery.Execute() {
		cameraID := result.EntityID
		cam, err := world.GetComponent[components.Camera](cameraID)
		if err != nil {
			continue
		}

		if cam.Primary {
			primaryCamera = cam
			cameraFound = true
			break
		}
	}

	if !cameraFound {
		return // No primary camera, nothing to render
	}

	// Reuse buffer to avoid allocations in hot path
	r.renderItems = r.renderItems[:0]
	// Get the camera's visible world-space bounds for frustum culling
	viewportBounds := primaryCamera.ViewportBounds()

	for entry := range world.NewQuery().WithRequiredComponents(components.Sprite{}, components.Transform{}).Execute() {
		if !entry.Entity.IsAlive() {
			continue
		}
		renderable, _ := entry.Get[components.Sprite]()
		transform, _ := entry.Get[components.Transform]()

		entityBounds := geom.Rect{
			Min: geom.Vector2{X: transform.Position.X - transform.Scale.X, Y: transform.Position.Y - transform.Scale.Y},
			Max: geom.Vector2{X: transform.Position.X + transform.Scale.X, Y: transform.Position.Y + transform.Scale.Y},
		}
		if !viewportBounds.Intersects(entityBounds) {
			continue
		}

		// Optionally fetch Animation component if it exists
		var anim *components.Animation
		if animation, err := entry.Get[components.Animation](); err == nil {
			anim = &animation
		}

		r.renderItems = append(r.renderItems, renderItem{
			entityID:   entry.EntityID,
			renderable: renderable,
			transform:  transform,
			animation:  anim,
		})
	}

	// Sort by layer > render depth > Y position (entityId too unstable)
	slices.SortStableFunc(r.renderItems, func(a, b renderItem) int {
		if a.renderable.Layer != b.renderable.Layer {
			return int(a.renderable.Layer) - int(b.renderable.Layer)
		}
		// render depth comparison
		if a.renderable.SortOrder != b.renderable.SortOrder {
			return int(a.renderable.SortOrder) - int(b.renderable.SortOrder)
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
	for _, item := range r.renderItems {
		if item.renderable.TexturePath != "" {
			if r.clipStore == nil {
				// first time, initialize the clip store from world resources
				clipStore, ok := world.GetResource[*animation.AnimationClipStore]()
				if !ok {
					// failed to get the clip store, skip rendering this sprite
					continue
				}
				r.clipStore = clipStore
			}
			r.drawSprite(screen, primaryCamera, item.transform, item.renderable, item.animation)
		} else {
			r.Primitive.Draw(screen, primaryCamera, item.transform, item.renderable)
		}
	}
}

func (r *Renderer) DrawDebugInfo(screen *ebiten.Image, world *ecs.World) {
	// Query for the primary camera
	var primaryCamera components.Camera
	var cameraFound bool

	if r.cameraQuery == nil {
		r.cameraQuery = world.NewQuery().WithRequiredComponents(components.Camera{})
	}
	for result := range r.cameraQuery.Execute() {
		cameraID := result.EntityID
		cam, err := world.GetComponent[components.Camera](cameraID)
		if err != nil {
			continue
		}

		if cam.Primary {
			primaryCamera = cam
			cameraFound = true
			break
		}
	}

	if !cameraFound {
		return
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.1f\nTPS: %0.1f\nCamera Position: %v\n", ebiten.ActualFPS(), ebiten.ActualTPS(), primaryCamera.Position))
}

func (r *Renderer) drawSprite(screen *ebiten.Image, cam components.Camera, transform components.Transform, renderable components.Sprite, anim *components.Animation) {
	var frameW, frameH int
	var frameImage *ebiten.Image

	// If animation is present, resolve the frame texture from the clip
	if anim != nil && anim.ClipPath != "" {
		clip := r.clipStore.Get(anim.ClipPath)
		if clip == nil {
			// Fall back to static texture if clip not found
			r.drawStaticTexture(screen, cam, transform, renderable)
			return
		}

		// Ensure frame index is in bounds
		if anim.FrameIndex >= len(clip.Frames) {
			return // frame index out of bounds, skip
		}

		// Get the region name from the clip's frame list
		regionName := clip.Frames[anim.FrameIndex]

		// Get the subimage from the atlas
		subTex, ok := clip.Atlas.Regions[regionName]
		if !ok {
			return // region not found in atlas
		}

		frameImage = subTex.Image
		frameW = subTex.Width
		frameH = subTex.Height
	} else {
		// No animation, load static texture
		tx, err := r.textureLoader.Load(renderable.TexturePath)
		if err != nil {
			return // silently skip entities with missing textures
		}

		frameImage = tx.Image
		frameW = tx.Width
		frameH = tx.Height
	}

	screenPos := cam.WorldToScreen(transform.Position)

	op := &ebiten.DrawImageOptions{}
	// first we set the position of the sprite on the screen by updating the DrawImageOptions
	op.GeoM.Translate(-float64(frameW)/2, -float64(frameH)/2)
	// Apply scaling (including camera zoom), rotation, and other transforms from Transform
	scaleX := transform.Scale.X * cam.Zoom
	scaleY := transform.Scale.Y * cam.Zoom
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	cr, cg, cb, ca := colorOrDefault(transform.Color).RGBA()
	op.ColorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	// Finally, draw the sprite's texture onto the screen using the options
	screen.DrawImage(frameImage, op)
}

func (r *Renderer) drawStaticTexture(screen *ebiten.Image, cam components.Camera, transform components.Transform, renderable components.Sprite) {
	tx, err := r.textureLoader.Load(renderable.TexturePath)
	if err != nil {
		return // silently skip entities with missing textures
	}

	frameW, frameH := tx.Width, tx.Height
	screenPos := cam.WorldToScreen(transform.Position)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(frameW)/2, -float64(frameH)/2)
	scaleX := transform.Scale.X * cam.Zoom
	scaleY := transform.Scale.Y * cam.Zoom
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	cr, cg, cb, ca := colorOrDefault(transform.Color).RGBA()
	op.ColorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	screen.DrawImage(tx.Image, op)
}
