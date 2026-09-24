package core

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// DrawItem is one resolved sprite, ready to blit. The render source is
// fully resolved at collection: an atlas region became (Texture, Rect)
// here, so the runner never deals in atlas IDs. World-space,
// interpolated; the blit projects to screen.
type DrawItem struct {
	// The texture ID for the sprite or sprite atlas.
	Texture asset.ID
	// The rectangle within the texture to use for this sprite. If empty, the entire texture is used.
	Rect image.Rectangle
	// The interpolated position of the sprite in world space.
	Position geom.Vector2
	// The rotation of the sprite in world space, in radians.
	Rotation float64
	// The scale of the sprite in world space.
	Scale geom.Vector2
	// Whether the sprite is flipped horizontally and vertically.
	FlipH, FlipV bool
	// The tint color to apply to the sprite. nil means no tint.
	Tint color.Color
	// The opacity of the sprite, from 0.0 (fully transparent) to 1.0 (fully opaque).
	Opacity float32
}

// CameraView is the resolved render camera: interpolated position and
// zoom, everything the blit needs to project.
type CameraView struct {
	Position geom.Vector2
	Zoom     float64
}

// Scene is one collected frame: the camera and the sorted draw list.
// Items views the collector's reused buffer of draw items. It should
// not be modified directly or retained beyond the frame it was collected for.
type Scene struct {
	Camera CameraView
	Items  []DrawItem
}

// workingItem carries the sort keys through collection. The keys are
// consumed by the sort and stripped before items join the Scene:
// consumers see the post-sort draw list, lean, without ordering data.
type workingItem struct {
	DrawItem
	layer     uint8
	sortOrder int8
	worldY    float64 // interpolated, the Y-fallback sort key
}

// Collector is responsible for collecting draw items from various queries
// and maintaining a sorted list of items ready for rendering.
type Collector struct {
	camera,
	textureSprites,
	atlasSprites *Query
	// working is the pre-sort buffer; SortedItems the post-sort output.
	// Both are reused across frames: steady-state collection is zero-alloc.
	working     []workingItem
	SortedItems []DrawItem
}

func NewCollector(world *World) *Collector {
	// predicate checks for camera component with primary field set to true
	camera := NewQuery(world).
		With(Camera{}, Transform{}, PrevTransform{}).
		Where(func(e Entry) bool {
			cam, ok := e.Component[Camera]()
			if !ok {
				return false
			}
			return cam.Primary
		})

	textureSprites := NewQuery(world).
		With(TextureSprite{}, Sprite{}, Transform{}, PrevTransform{}).
		Where(func(e Entry) bool {
			s, ok := e.Component[Sprite]()
			if !ok {
				return false
			}
			return s.Visible
		})

	atlasSprites := NewQuery(world).
		With(AtlasSprite{}, Sprite{}, Transform{}, PrevTransform{}).
		Where(func(e Entry) bool {
			s, ok := e.Component[Sprite]()
			if !ok {
				return false
			}
			return s.Visible
		})
	return &Collector{
		camera:         camera,
		textureSprites: textureSprites,
		atlasSprites:   atlasSprites,
		working:        make([]workingItem, 0),
		SortedItems:    make([]DrawItem, 0),
	}
}

// Collect resolves the primary camera (interpolated), collects both
// sprite source variants, interpolates positions from PrevTransform by
// ctx.Alpha, culls against the camera viewport, and sorts by layer →
// SortOrder → world Y. It returns a Scene viewing the collector's
// buffers; do not retain it across frames.
//
// No primary camera: an empty Scene, no error — overlays still run.
// Any unresolvable sprite source such as an unregistered atlas or region, or
// a texture that will not load fails the collect, naming the handles:
// registration problems surface at the first rendered frame.
func (c *Collector) Collect(ctx *Context) (Scene, error) {
	// First primary wins; query iteration order is deterministic.
	entry, ok := c.camera.First()
	if !ok {
		return Scene{}, nil
	}
	cam, _ := entry.Component[Camera]()
	camTransform, _ := entry.Component[Transform]()
	camPrev, _ := entry.Component[PrevTransform]()
	camera := CameraView{
		Position: camPrev.Position.Lerp(camTransform.Position, ctx.Alpha),
		Zoom:     cam.Zoom,
	}

	// The culling viewport: the logical resolution divided by zoom, in
	// world space, centered on the interpolated camera.
	viewport := geom.RectFromCenterSize(camera.Position, geom.Vector2{
		X: float64(ctx.LogicalWidth) / camera.Zoom,
		Y: float64(ctx.LogicalHeight) / camera.Zoom,
	})

	server, err := ctx.World.Resource[*asset.Server]()
	if err != nil {
		return Scene{}, fmt.Errorf("castrum: collect: resolve asset server: %w", err)
	}

	c.working = c.working[:0]

	for e := range c.textureSprites.Execute() {
		src, _ := e.Component[TextureSprite]()
		sprite, _ := e.Component[Sprite]()
		transform, _ := e.Component[Transform]()
		prev, _ := e.Component[PrevTransform]()

		data, err := server.Load[asset.TextureData](string(src.Texture))
		if err != nil {
			return Scene{}, fmt.Errorf("castrum: collect: texture %q: %w", src.Texture, err)
		}
		c.stage(viewport, ctx.Alpha, sprite, transform, prev,
			src.Texture, image.Rectangle{}, data.Width, data.Height)
	}

	for e := range c.atlasSprites.Execute() {
		src, _ := e.Component[AtlasSprite]()
		sprite, _ := e.Component[Sprite]()
		transform, _ := e.Component[Transform]()
		prev, _ := e.Component[PrevTransform]()

		atlas, err := server.Store().Atlas(src.Atlas)
		if err != nil {
			return Scene{}, fmt.Errorf("castrum: collect: atlas %q: %w", src.Atlas, err)
		}
		region, err := atlas.Region(src.Region)
		if err != nil {
			return Scene{}, fmt.Errorf("castrum: collect: atlas %q: %w", src.Atlas, err)
		}
		c.stage(viewport, ctx.Alpha, sprite, transform, prev,
			atlas.TexturePath(), region.Rect(), region.W, region.H)
	}

	slices.SortStableFunc(c.working, func(a, b workingItem) int {
		if a.layer != b.layer {
			return cmp.Compare(a.layer, b.layer)
		}
		if a.sortOrder != b.sortOrder {
			return cmp.Compare(a.sortOrder, b.sortOrder)
		}
		return cmp.Compare(a.worldY, b.worldY)
	})

	c.SortedItems = c.SortedItems[:0]
	for _, item := range c.working {
		c.SortedItems = append(c.SortedItems, item.DrawItem)
	}
	return Scene{Camera: camera, Items: c.SortedItems}, nil
}

// stage interpolates one sprite's position, culls it against the
// viewport, and appends it to the working buffer. width and height are
// the sprite's source dimensions in pixels — the atlas region's, or
// the standalone texture's — before world scale; rect is empty for
// whole textures.
func (c *Collector) stage(viewport geom.Rect, alpha float64, sprite Sprite, transform Transform, prev PrevTransform, texture asset.ID, rect image.Rectangle, width, height int) {
	position := prev.Position.Lerp(transform.Position, alpha)
	bounds := geom.RectFromCenterSize(position, geom.Vector2{
		X: float64(width) * transform.Scale.X,
		Y: float64(height) * transform.Scale.Y,
	})
	if !viewport.OverlapsOrTouches(bounds) {
		return
	}
	c.working = append(c.working, workingItem{
		Texture:   texture,
		Rect:      rect,
		Position:  position,
		Rotation:  transform.Rotation,
		Scale:     transform.Scale,
		FlipH:     sprite.FlipH,
		FlipV:     sprite.FlipV,
		Tint:      sprite.Tint,
		Opacity:   sprite.Opacity,
		layer:     sprite.Layer,
		sortOrder: sprite.SortOrder,
		worldY:    position.Y,
	})
}
