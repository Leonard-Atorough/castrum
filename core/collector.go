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

// DrawItem is a drawable resolved for rendering. Texture items carry their
// texture and source rectangle; shape items carry their geometry. Positions
// are interpolated in world space for the current frame.
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
	// non-nil, Color is the shape's color and is never nil: the
	// collector defaults a nil Sprite.Color to black.
	Shape Shape
	// Text is the string of a text item; empty on every other item.
	Text string
	// Font is the font of a text item; empty on every other item. It also
	// identifies text items.
	Font asset.ID
	// TextSize is the text item's font size in pixels at scale one.
	TextSize float64
	// TextWidth and TextHeight are the measured size of the item's
	// text at scale one, from the font's metrics. The blit anchors by
	// them, so an item draws exactly as it was measured.
	TextWidth, TextHeight float64
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
	// Color is the color to multiply the sprite's pixels by, or the
	// shape's fill and stroke color; its alpha channel is the
	// drawable's opacity. nil means no color on a sprite; shape
	// items always carry a concrete color.
	Color color.Color
	// Outline strokes the shape's border instead of filling it.
	// Applies when Shape is non-nil.
	Outline bool
	// StrokeWidth is the outline's width in world units; the blit
	// scales it by zoom. Applies when Shape is non-nil and Outline
	// is set.
	StrokeWidth float64
}

// DrawList contains the resolved camera and draw items for one frame.
type DrawList struct {
	// Camera is the resolved primary camera view for this frame.
	Camera CameraView
	// Items is the sorted set of drawables for this frame. When returned by
	// [Collector.Collect], the slice is owned by the collector and is
	// overwritten by its next call.
	Items []DrawItem
}

type workingItem struct {
	DrawItem
	layer     uint8
	sortOrder int8
	worldY    float64 // interpolated, the Y-fallback sort key
}

// Collector builds each frame's draw list by resolving drawable assets,
// culling against the camera viewport, and sorting by layer, sort order, and
// world Y. Reuse a Collector across frames.
type Collector struct {
	userCamera,
	engineCamera,
	sprites *Query
	working []workingItem
	// SortedItems is the sorted output the returned DrawList views.
	// It is reused across frames; do not modify or retain it.
	SortedItems []DrawItem
}

// NewCollector creates a [Collector] for world. A primary user camera takes
// precedence over the engine camera.
func NewCollector(world *World) *Collector {
	// User-spawned primaries are preferred; the engine camera
	// (SpawnEngineCamera) is the fallback when none exists.
	userCamera := NewQuery(world).
		With(Camera{}, Transform{}, PrevTransform{}).
		Without(engineCamera{}).
		Where(func(e Entry) bool {
			cam, ok := e.Component[Camera]()
			if !ok {
				return false
			}
			return cam.Primary
		})
	engineCamera := NewQuery(world).
		With(Camera{}, Transform{}, PrevTransform{}, engineCamera{}).
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
		userCamera:   userCamera,
		engineCamera: engineCamera,
		sprites:      sprites,
		working:      make([]workingItem, 0),
		SortedItems:  make([]DrawItem, 0),
	}
}

// Collect returns the draw list for the current frame. It interpolates
// drawable transforms using [Context.Alpha], culls against the camera
// viewport, and sorts items by layer, sort order, then world Y. Text bounds
// come from [asset.FontData.Measure].
//
// If no primary camera is available, Collect returns an empty list without
// an error. Errors resolving fonts, textures, atlases, or atlas regions are
// returned; shapes require no assets.
func (c *Collector) Collect(ctx *Context) (DrawList, error) {
	// A user-spawned primary wins; the engine camera is the
	// fallback. Query iteration order is deterministic within each.
	entry, ok := c.userCamera.First()
	if !ok {
		entry, ok = c.engineCamera.First()
	}
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

		if transform.Scale.X == 0 {
			transform.Scale.X = 1
		}
		if transform.Scale.Y == 0 {
			transform.Scale.Y = 1
		}

		// Rotation and scale interpolate alongside position; size
		// and culling below use the interpolated scale.
		scale := prev.Scale.Lerp(transform.Scale, ctx.Alpha)
		rotation := prev.Rotation + (transform.Rotation-prev.Rotation)*ctx.Alpha

		switch drawable := sprite.Drawable.(type) {
		case nil:
			// Style without a picture: legal, not drawn.
			continue
		case AtlasSource:
			atlas, err := server.Store().Atlas(drawable.Atlas)
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: %w", err)
			}
			region, err := atlas.Region(drawable.Region)
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: %w", err)
			}
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // sprites are centered on the position
				geom.Vector2{
					X: float64(region.W) * scale.X,
					Y: float64(region.H) * scale.Y,
				},
				DrawItem{
					Texture:  atlas.TexturePath(),
					Rect:     region.Rect(),
					Rotation: rotation,
					Scale:    scale,
					FlipH:    sprite.FlipH,
					FlipV:    sprite.FlipV,
					Color:    sprite.Color,
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
					X: float64(data.Width) * scale.X,
					Y: float64(data.Height) * scale.Y,
				},
				DrawItem{
					Texture:  drawable.Texture,
					Rect:     image.Rectangle{},
					Rotation: rotation,
					Scale:    scale,
					FlipH:    sprite.FlipH,
					FlipV:    sprite.FlipV,
					Color:    sprite.Color,
				})
		case TextSource:
			if drawable.Text == "" {
				// Nothing to draw, like a nil drawable.
				continue
			}
			font, err := server.Load[asset.FontData](string(drawable.Font))
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: font %q: %w", drawable.Font, err)
			}
			textWidth, textHeight := font.Measure(drawable.Size, drawable.Text)
			fill := sprite.Color
			if fill == nil {
				// Text defaults to white, not black: text on a screen
				// has no authored color to inherit.
				fill = color.White
			}
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // text is centered on the position
				geom.Vector2{
					X: textWidth * scale.X,
					Y: textHeight * scale.Y,
				},
				DrawItem{
					Text:       drawable.Text,
					Font:       drawable.Font,
					TextSize:   drawable.Size,
					TextWidth:  textWidth,
					TextHeight: textHeight,
					Rotation:   rotation,
					Scale:      scale,
					Color:      fill,
				})
		case RectShape:
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // rects are centered on the position
				geom.Vector2{
					X: drawable.Size.X * scale.X,
					Y: drawable.Size.Y * scale.Y,
				},
				shapeItem(sprite, rotation, scale, drawable))
		case CircleShape:
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position,
				geom.Vector2{}, // circles are centered on the position
				geom.Vector2{
					X: 2 * drawable.Radii.X * scale.X,
					Y: 2 * drawable.Radii.Y * scale.Y,
				},
				shapeItem(sprite, rotation, scale, drawable))
		case LineShape:
			// The segment spans the two relative endpoints, so its
			// culling bounds are the segment's own AABB, offset from
			// the position by its center.
			segment := geom.Segment{
				Start: geom.Vector2{
					X: drawable.From.X * scale.X,
					Y: drawable.From.Y * scale.Y,
				},
				End: geom.Vector2{
					X: drawable.To.X * scale.X,
					Y: drawable.To.Y * scale.Y,
				},
			}
			bounds := segment.BoundingBox()
			c.stage(viewport, ctx.Alpha, sprite.Layer, sprite.SortOrder,
				prev.Position, transform.Position, bounds.Center(), bounds.Size(),
				shapeItem(sprite, rotation, scale, drawable))
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

func shapeItem(sprite Sprite, rotation float64, scale geom.Vector2, shape Shape) DrawItem {
	fill := sprite.Color
	if fill == nil {
		fill = color.Black
	}
	return DrawItem{
		Shape:       shape,
		Rotation:    rotation,
		Scale:       scale,
		Color:       fill,
		Outline:     sprite.Outline,
		StrokeWidth: sprite.StrokeWidth,
	}
}
