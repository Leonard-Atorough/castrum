package core

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// DrawItem is one resolved drawable, ready to blit: a texture sprite
// or a shape primitive. The render source is fully resolved at
// collection - an atlas region became (Texture, Rect) here, and a
// geometry variant became Shape - so the runner never deals in atlas
// IDs or component variants. World-space, interpolated; the blit
// projects to screen.
type DrawItem struct {
	// Texture is the texture the sprite's pixels come from: a
	// standalone texture's path, or the atlas's texture path. Applies
	// when Shape is nil.
	Texture asset.ID
	// Rect is the region of Texture to draw, in pixels. Empty means
	// the whole texture - dimension knowledge lives in the blit.
	// Applies when Shape is nil.
	Rect image.Rectangle
	// Shape is the primitive to draw instead of a texture. nil means
	// this item is a texture sprite and Texture and Rect apply. When
	// non-nil, Tint is the shape's color and is never nil: the
	// collector defaults a nil Primitive.Color to black.
	Shape Shape
	// Position is the interpolated position of the drawable in world
	// space.
	Position geom.Vector2
	// Rotation is the rotation of the drawable in world space, in
	// radians.
	Rotation float64
	// Scale is the scale of the drawable in world space.
	Scale geom.Vector2
	// FlipH and FlipV mirror the sprite horizontally and vertically.
	// Applies when Shape is nil; shapes have no flip.
	FlipH, FlipV bool
	// Tint is the color to multiply the sprite's pixels by, or the
	// shape's fill and stroke color. nil means no tint on a sprite;
	// shape items always carry a concrete color.
	Tint color.Color
	// Transparency fades the item: 0.0 - the zero value - is fully
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
// sprite queries, resolves texture sources through the asset server,
// culls against the camera viewport, and sorts by layer, sort order,
// then world Y. Reused across frames - construct once per game; the
// runner's engine draw calls Collect every frame.
type Collector struct {
	camera,
	sprites *Query
	// working is the pre-sort buffer; SortedItems the post-sort
	// output. Both are reused across frames: steady-state collection
	// is zero-alloc.
	working []workingItem
	// SortedItems is the sorted output the returned DrawList views.
	// It is reused across frames; do not modify or retain it.
	SortedItems []DrawItem
}

// NewCollector builds a Collector over world: one primary-camera
// query and one sprite query — every Drawable kind flows through the
// same query, because Sprite carries its Drawable as data rather than
// as component variants. Each is predicate-filtered (Primary; sprites
// not Hidden). The first primary camera in deterministic query order
// frames the world; sprites pair with a Transform and a PrevTransform
// to match.
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

	sprites := NewQuery(world).
		With(Sprite{}, Transform{}, PrevTransform{}).
		Where(func(e Entry) bool {
			s, ok := e.Component[Sprite]()
			if !ok {
				return false
			}
			return !s.Hidden
		})

	return &Collector{
		camera:      camera,
		sprites:     sprites,
		working:     make([]workingItem, 0),
		SortedItems: make([]DrawItem, 0),
	}
}

// Collect resolves the primary camera, collects every sprite — the
// Drawable sum picks the path: texture sources resolve through the
// asset server, shapes carry geometry alone, nil is style without a
// picture and skips — interpolates positions from PrevTransform by
// ctx.Alpha, culls against the camera viewport, and sorts by layer →
// SortOrder → world Y. It returns a DrawList viewing the collector's
// buffers; do not retain it across frames.
//
// No primary camera: an empty DrawList, no error - overlays still run.
// An unresolvable texture source - an unregistered atlas or region,
// or a texture that will not load - fails the collect, naming the
// handles: registration problems surface at the first rendered frame.
// Shapes cannot fail: there is nothing to resolve.
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

	for e := range c.sprites.Execute() {
		sprite, _ := e.Component[Sprite]()
		transform, _ := e.Component[Transform]()
		prev, _ := e.Component[PrevTransform]()

		switch drawable := sprite.Drawable.(type) {
		case nil:
			// Style without a picture: legal, not drawn.
			continue
		case AtlasSource:
			atlas, err := server.Store().Atlas(drawable.Atlas)
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: atlas %q: %w", drawable.Atlas, err)
			}
			region, err := atlas.Region(drawable.Region)
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: atlas %q: %w", drawable.Atlas, err)
			}
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // sprites are centered on the position
				geom.Vector2{
					X: float64(region.W) * transform.Scale.X,
					Y: float64(region.H) * transform.Scale.Y,
				},
				DrawItem{
					Texture:      atlas.TexturePath(),
					Rect:         region.Rect(),
					Rotation:     transform.Rotation,
					Scale:        transform.Scale,
					FlipH:        sprite.FlipH,
					FlipV:        sprite.FlipV,
					Tint:         sprite.Tint,
					Transparency: sprite.Transparency,
				})
		case TextureSource:
			data, err := server.Load[asset.TextureData](string(drawable.Texture))
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: texture %q: %w", drawable.Texture, err)
			}
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // sprites are centered on the position
				geom.Vector2{
					X: float64(data.Width) * transform.Scale.X,
					Y: float64(data.Height) * transform.Scale.Y,
				},
				DrawItem{
					Texture:      drawable.Texture,
					Rect:         image.Rectangle{},
					Rotation:     transform.Rotation,
					Scale:        transform.Scale,
					FlipH:        sprite.FlipH,
					FlipV:        sprite.FlipV,
					Tint:         sprite.Tint,
					Transparency: sprite.Transparency,
				})
		case RectShape:
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // rects are centered on the position
				geom.Vector2{
					X: drawable.Size.X * transform.Scale.X,
					Y: drawable.Size.Y * transform.Scale.Y,
				},
				shapeItem(sprite, transform, drawable))
		case CircleShape:
			diameter := 2 * drawable.Radius
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // circles are centered on the position
				geom.Vector2{
					X: diameter * transform.Scale.X,
					Y: diameter * transform.Scale.Y,
				},
				shapeItem(sprite, transform, drawable))
		case LineShape:
			// A segment spans position to position+To, so its bounds
			// center halfway along the segment, not on the position.
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{
					X: drawable.To.X * transform.Scale.X / 2,
					Y: drawable.To.Y * transform.Scale.Y / 2,
				},
				geom.Vector2{
					X: math.Abs(drawable.To.X * transform.Scale.X),
					Y: math.Abs(drawable.To.Y * transform.Scale.Y),
				},
				shapeItem(sprite, transform, drawable))
		default:
			// Unreachable: the sum is sealed and Validate covers it.
			continue
		}
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

// stage interpolates one drawable's position between prev and curr,
// culls it against the viewport, and appends it to the working buffer
// with its sort keys. offset shifts the bounds center off the
// position - zero for centered drawables (sprites, rects, circles),
// half the segment for lines; size is the drawable's world-space
// extent, already scaled. Rotation is ignored for culling, as with
// sprites.
func (c *Collector) stage(viewport geom.Rect, alpha float64, layer uint8, sortOrder int8, prev, curr, offset, size geom.Vector2, item DrawItem) {
	position := prev.Lerp(curr, alpha)
	bounds := geom.RectFromCenterSize(position.Add(offset), size)
	if !viewport.OverlapsOrTouches(bounds) {
		return
	}
	item.Position = position
	c.working = append(c.working, workingItem{
		DrawItem:  item,
		layer:     layer,
		sortOrder: sortOrder,
		worldY:    position.Y,
	})
}

// shapeItem assembles a shape sprite's DrawItem: the geometry carried
// straight through as the item's Shape, snapped rotation and scale,
// and style. A nil tint draws black, so shape items always carry a
// concrete color.
func shapeItem(sprite Sprite, transform Transform, shape Shape) DrawItem {
	fill := sprite.Tint
	if fill == nil {
		fill = color.Black
	}
	return DrawItem{
		Shape:        shape,
		Rotation:     transform.Rotation,
		Scale:        transform.Scale,
		Tint:         fill,
		Transparency: sprite.Transparency,
	}
}
