package render

import (
	"context"
	"fmt"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/atlas"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/geom"
)

type renderItem struct {
	entityID  ecs.EntityID
	sprite    components.Sprite
	transform components.Transform
	animation *components.Animation
}

type RenderConfig struct {
	DrawDebugInfo bool
}

// TextureProvider is the interface for loading individual textures.
// Defined here (the consumer) rather than in the assets package, so Renderer
// only couples to the behavior it needs.
type TextureProvider interface {
	Load(ctx context.Context, id assets.ID) (*ebiten.Image, int, int, error)
	SubImage(ctx context.Context, assetId assets.ID, atlasID atlas.ID, regionName string) (*ebiten.Image, int, int, error)
}

type Renderer struct {
	textureProvider      TextureProvider
	clipStore            *animation.AnimationClipStore
	world                *ecs.World
	Primitive            *PrimitiveRenderer
	cameraQuery          *ecs.Query
	spriteTransformQuery *ecs.Query
	renderItems          []renderItem
	renderErrors         []RenderError
	config               RenderConfig
}

func New(textureProvider TextureProvider, world *ecs.World, config RenderConfig) *Renderer {
	return &Renderer{
		textureProvider: textureProvider,
		world:           world,
		Primitive:       NewPrimitiveRenderer(),
		renderItems:     make([]renderItem, 0, 16),  // pre-allocate buffer for hot path
		renderErrors:    make([]RenderError, 0, 16), // pre-allocate buffer for capturing rendering errors
		config:          config,
	}
}

func (r *Renderer) Clear(screen *ebiten.Image, c color.Color) {
	screen.Fill(c)
}

// DrawScene renders the scene from the perspective of the primary camera.
// Entities with Sprite and Transform components are considered renderable.
func (r *Renderer) DrawScene(ctx context.Context, screen *ebiten.Image) {
	var primaryCamera components.Camera
	var cameraFound bool

	if r.cameraQuery == nil {
		r.cameraQuery = r.world.NewQuery().WithRequiredComponents(components.Camera{})
	}
	for result := range r.cameraQuery.Execute() {
		cameraID := result.EntityID
		cam, err := r.world.GetComponent[components.Camera](cameraID)
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

	r.renderItems = r.renderItems[:0]
	r.renderErrors = r.renderErrors[:0]

	viewportBounds := primaryCamera.ViewportBounds()

	if r.spriteTransformQuery == nil {
		r.spriteTransformQuery = r.world.NewQuery().WithRequiredComponents(components.Sprite{}, components.Transform{})
	}
	for entry := range r.spriteTransformQuery.Execute() {
		if !entry.Entity.IsAlive() {
			continue
		}
		renderable, _ := entry.Get[components.Sprite]()
		if !renderable.Visible {
			continue
		}
		transform, _ := entry.Get[components.Transform]()

		var anim *components.Animation
		if animation, err := entry.Get[components.Animation](); err == nil {
			anim = &animation
		}

		// Compute rendered half-extents for viewport culling.
		var halfW, halfH float64
		cullable := true
		if renderable.TexturePath != "" {
			w, h, ok := r.resolveTextureDimensions(ctx, renderable, anim)
			if ok {
				halfW = float64(w) * transform.Scale.X / 2
				halfH = float64(h) * transform.Scale.Y / 2
			} else {
				// Can't resolve dimensions (texture missing, clip not
				// ready) — skip culling; the render pass will handle it.
				cullable = false
			}
		} else {
			// Primitive: base size from Sprite.Size, multiplied by Scale.
			halfW = renderable.Size.X * transform.Scale.X / 2
			halfH = renderable.Size.Y * transform.Scale.Y / 2
		}

		if cullable {
			entityBounds := geom.Rect{
				Min: geom.Vector2{X: transform.Position.X - halfW, Y: transform.Position.Y - halfH},
				Max: geom.Vector2{X: transform.Position.X + halfW, Y: transform.Position.Y + halfH},
			}
			if !viewportBounds.Intersects(entityBounds) {
				continue
			}
		}

		r.renderItems = append(r.renderItems, renderItem{
			entityID:  entry.EntityID,
			sprite:    renderable,
			transform: transform,
			animation: anim,
		})
	}

	slices.SortStableFunc(r.renderItems, func(a, b renderItem) int {
		if a.sprite.RenderLayer != b.sprite.RenderLayer {
			return int(a.sprite.RenderLayer) - int(b.sprite.RenderLayer)
		}
		if a.sprite.SortOrder != b.sprite.SortOrder {
			return int(a.sprite.SortOrder) - int(b.sprite.SortOrder)
		}
		if a.transform.Position.Y != b.transform.Position.Y {
			if a.transform.Position.Y < b.transform.Position.Y {
				return -1
			}
			return 1
		}
		return 0
	})

	for _, item := range r.renderItems {
		if err := r.renderItem(ctx, screen, primaryCamera, item); err != nil {
			r.renderErrors = append(r.renderErrors, RenderError{
				EntityID: item.entityID,
				Err:      err,
				Step:     "render",
				Message:  "failed to render item",
			})
		}
	}

	if r.config.DrawDebugInfo {
		r.drawDebugInfo(screen, primaryCamera)
	}
}

func (r *Renderer) renderItem(ctx context.Context, screen *ebiten.Image, cam components.Camera, item renderItem) error {
	if err := r.validateRenderItem(item); err != nil {
		return err
	}

	if item.sprite.TexturePath != "" {
		if item.animation != nil {
			return r.renderAnimation(ctx, screen, cam, item)
		}
		return r.renderSprite(ctx, screen, cam, item)
	}
	// TODO: Fix sprite not containing color
	// if all else fails, use the primitive renderer
	return r.Primitive.Draw(screen, cam, item.transform, item.sprite)

}

func (r *Renderer) renderAnimation(ctx context.Context, screen *ebiten.Image, cam components.Camera, item renderItem) error {
	if r.clipStore == nil {
		clipStore, ok := r.world.GetResource[*animation.AnimationClipStore]()
		if !ok {
			return fmt.Errorf("failed to get animation clip store")
		}
		r.clipStore = clipStore
	}

	clip := r.clipStore.Get(item.animation.ClipPath)
	if clip == nil {
		//fallback to static texture
		return r.renderSprite(ctx, screen, cam, item)
	}

	if item.animation.FrameIndex >= len(clip.Frames) {
		return fmt.Errorf("animation frame index out of range: %d, total frames: %d", item.animation.FrameIndex, len(clip.Frames))
	}

	regionName := clip.Frames[item.animation.FrameIndex]
	if clip.Atlas == nil {
		return fmt.Errorf("animation clip atlas is nil")
	}
	subTex, w, h, err := r.textureProvider.SubImage(ctx, assets.ID(item.sprite.TexturePath), clip.Atlas.ID(), regionName)
	if err != nil || subTex == nil {
		return fmt.Errorf("failed to get subimage for region: %s", regionName)
	}

	r.drawImage(ctx, screen, cam, item.transform, item.sprite, subTex, w, h)
	return nil
}

func (r *Renderer) renderSprite(ctx context.Context, screen *ebiten.Image, cam components.Camera, item renderItem) error {
	var subTex *ebiten.Image
	var w, h int
	if item.sprite.AtlasID != "" && item.sprite.RegionName != "" {
		var err error
		subTex, w, h, err = r.textureProvider.SubImage(ctx, assets.ID(item.sprite.TexturePath), atlas.ID(item.sprite.AtlasID), item.sprite.RegionName)
		if err != nil || subTex == nil {
			return fmt.Errorf("failed to get subimage for region: %s", item.sprite.RegionName)
		}
	}

	if subTex == nil {
		tex, tW, tH, err := r.textureProvider.Load(ctx, assets.ID(item.sprite.TexturePath))
		if err != nil || tex == nil {
			return fmt.Errorf("failed to load texture: %s", item.sprite.TexturePath)
		}
		subTex = tex
		w = tW
		h = tH
	}
	r.drawImage(ctx, screen, cam, item.transform, item.sprite, subTex, w, h)
	return nil
}

func (r *Renderer) drawImage(_ context.Context, screen *ebiten.Image, cam components.Camera, transform components.Transform, sprite components.Sprite, img *ebiten.Image, w, h int) {
	screenPos := cam.WorldToScreen(transform.Position)

	op := &ebiten.DrawImageOptions{}
	// first we set the position of the sprite on the screen by updating the DrawImageOptions
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	// Apply scaling (including camera zoom), rotation, and other transforms from Transform
	scaleX := transform.Scale.X * cam.Zoom
	scaleY := transform.Scale.Y * cam.Zoom
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(transform.Rotation)
	op.GeoM.Translate(screenPos.X, screenPos.Y)

	cr, cg, cb, ca := colorOrDefault(sprite.Color).RGBA()
	op.ColorScale.Scale(float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff)

	screen.DrawImage(img, op)
}

func (r *Renderer) validateRenderItem(item renderItem) error {
	if err := item.sprite.Validate(); err != nil {
		return err
	}
	if item.animation != nil {
		if err := item.animation.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// resolveTextureDimensions returns the rendered width and height of a
// textured sprite so the cull pass can compute accurate viewport bounds. The
// texture provider caches, so the render pass pays no extra cost.
func (r *Renderer) resolveTextureDimensions(ctx context.Context, sprite components.Sprite, anim *components.Animation) (int, int, bool) {
	if anim != nil {
		if r.clipStore == nil {
			clipStore, ok := r.world.GetResource[*animation.AnimationClipStore]()
			if !ok {
				return 0, 0, false
			}
			r.clipStore = clipStore
		}
		clip := r.clipStore.Get(anim.ClipPath)
		if clip == nil || clip.Atlas == nil {
			return 0, 0, false
		}
		if anim.FrameIndex < 0 || anim.FrameIndex >= len(clip.Frames) {
			return 0, 0, false
		}
		_, w, h, err := r.textureProvider.SubImage(ctx, assets.ID(sprite.TexturePath), clip.Atlas.ID(), clip.Frames[anim.FrameIndex])
		if err != nil {
			return 0, 0, false
		}
		return w, h, true
	}

	if sprite.AtlasID != "" && sprite.RegionName != "" {
		_, w, h, err := r.textureProvider.SubImage(ctx, assets.ID(sprite.TexturePath), atlas.ID(sprite.AtlasID), sprite.RegionName)
		if err != nil {
			return 0, 0, false
		}
		return w, h, true
	}

	_, w, h, err := r.textureProvider.Load(ctx, assets.ID(sprite.TexturePath))
	if err != nil {
		return 0, 0, false
	}
	return w, h, true
}

func (r *Renderer) drawDebugInfo(screen *ebiten.Image, cam components.Camera) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.1f\nTPS: %0.1f\nCamera Position: %v\n", ebiten.ActualFPS(), ebiten.ActualTPS(), cam.Position))
}

type RenderError struct {
	EntityID ecs.EntityID
	Step     string
	Message  string
	Err      error
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("RendererError: EntityID=%v, Step=%s, Message=%s, Err=%v", e.EntityID, e.Step, e.Message, e.Err)
}

func (e *RenderError) Unwrap() error {
	return e.Err
}
