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
	// Texture is the texture the sprite's pixels come from: a
	// standalone texture's path, or the atlas's texture path.
	Texture asset.ID
	// Rect is the region of Texture to draw, in pixels. Empty means
	// the whole texture - dimension knowledge lives in the blit.
	Rect image.Rectangle
	// Position is the interpolated position of the sprite in world
	// space.
	Position geom.Vector2
	// Rotation is the rotation of the sprite in world space, in
	// radians.
	Rotation float64
	// Scale is the scale of the sprite in world space.
	Scale geom.Vector2
	// FlipH and FlipV mirror the sprite horizontally and vertically.
	FlipH, FlipV bool
	// Tint is the color to multiply the sprite's pixels by. nil means
	// no tint.
	Tint color.Color
	// Transparency fades the item: 0.0 — the zero value — is fully
	// opaque, 1.0 is fully see-through.
	Transparency float32
}

// CameraView is the resolved render camera: the interpolated position
// (previous → current by the collect alpha) and the current zoom -
// everything the blit needs to project world space to the screen.
type CameraView struct {
	Position geom.Vector2
	Zoom     float64
}

// DrawList is one collected frame: the resolved camera and the sorted
// draw items. Items views the collector's reused buffer; do not modify
// it or retain the list beyond the frame it was collected for.
type DrawList struct {
	Camera CameraView
	Items  []DrawItem
}

// workingItem carries the sort keys through collection. The keys are
// consumed by the sort and stripped before items join the DrawList:
// consumers see the post-sort draw list, lean, without ordering data.
type workingItem struct {
	DrawItem
	layer     uint8
	sortOrder int8
	worldY    float64 // interpolated, the Y-fallback sort key
}

// Collector assembles each frame's draw list: it owns the camera and
// sprite queries, resolves sprite sources through the asset server,
// culls against the camera viewport, and sorts by layer, sort order,
// then world Y. Reused across frames - construct once per game; the
// runner's engine draw calls Collect every frame.
type Collector struct {
	camera,
	textureSprites,
	atlasSprites *Query
	// working is the pre-sort buffer; SortedItems the post-sort
	// output. Both are reused across frames: steady-state collection
	// is zero-alloc.
	working []workingItem
	// SortedItems is the sorted output the returned DrawList views.
	// It is reused across frames; do not modify or retain it.
	SortedItems []DrawItem
}

// NewCollector builds a Collector over world: one primary-camera
// query and the two sprite-source queries, each predicate-filtered
// (Primary; sprites not Hidden). The first primary camera in
// deterministic query order frames the world; sprites must pair Sprite
// with a source variant (TextureSprite or AtlasSprite), a Transform,
// and a PrevTransform to match.
func NewCollector(world *World) *Collector {
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
			return !s.Hidden
		})

	atlasSprites := NewQuery(world).
		With(AtlasSprite{}, Sprite{}, Transform{}, PrevTransform{}).
		Where(func(e Entry) bool {
			s, ok := e.Component[Sprite]()
			if !ok {
				return false
			}
			return !s.Hidden
		})
	return &Collector{
		camera:         camera,
		textureSprites: textureSprites,
		atlasSprites:   atlasSprites,
		working:        make([]workingItem, 0),
		SortedItems:    make([]DrawItem, 0),
	}
}

// Collect resolves the primary camera, collects both sprite source
// variants, interpolates positions from PrevTransform by ctx.Alpha,
// culls against the camera viewport, and sorts by layer → SortOrder →
// world Y. It returns a DrawList viewing the collector's buffers; do
// not retain it across frames.
//
// No primary camera: an empty DrawList, no error - overlays still run.
// Any unresolvable sprite source - an unregistered atlas or region, or
// a texture that will not load - fails the collect, naming the handles:
// registration problems surface at the first rendered frame.
func (c *Collector) Collect(ctx *Context) (DrawList, error) {
	// First primary wins; query iteration order is deterministic.
	entry, ok := c.camera.First()
	if !ok {
		return DrawList{}, nil
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
		return DrawList{}, fmt.Errorf("castrum: collect: resolve asset server: %w", err)
	}

	c.working = c.working[:0]

	for e := range c.textureSprites.Execute() {
		src, _ := e.Component[TextureSprite]()
		sprite, _ := e.Component[Sprite]()
		transform, _ := e.Component[Transform]()
		prev, _ := e.Component[PrevTransform]()

		data, err := server.Load[asset.TextureData](string(src.Texture))
		if err != nil {
			return DrawList{}, fmt.Errorf("castrum: collect: texture %q: %w", src.Texture, err)
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
			return DrawList{}, fmt.Errorf("castrum: collect: atlas %q: %w", src.Atlas, err)
		}
		region, err := atlas.Region(src.Region)
		if err != nil {
			return DrawList{}, fmt.Errorf("castrum: collect: atlas %q: %w", src.Atlas, err)
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
	return DrawList{Camera: camera, Items: c.SortedItems}, nil
}

// stage interpolates one sprite's position, culls it against the
// viewport, and appends it to the working buffer. width and height are
// the sprite's source dimensions in pixels - the atlas region's, or
// the standalone texture's - before world scale; rect is empty for
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
		Texture:      texture,
		Rect:         rect,
		Position:     position,
		Rotation:     transform.Rotation,
		Scale:        transform.Scale,
		FlipH:        sprite.FlipH,
		FlipV:        sprite.FlipV,
		Tint:         sprite.Tint,
		Transparency: sprite.Transparency,
		layer:        sprite.Layer,
		sortOrder:    sprite.SortOrder,
		worldY:       position.Y,
	})
}
