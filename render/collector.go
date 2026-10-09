// Package render defines drawable components and camera state, then resolves
// them into ordered [DrawList] values for a renderer to draw.
package render

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// DrawItem is one drawable resolved from a [Sprite] for a frame.
type DrawItem struct {
	// Texture is the source texture for a texture item; empty for text and shapes.
	Texture asset.ID
	// Rect is the source region in Texture, in pixels. An empty rectangle
	// selects the whole texture.
	Rect image.Rectangle
	// Shape is the geometry of a shape item; nil for texture and text items.
	Shape Shape
	// Text is the text to draw; empty for non-text items.
	Text string
	// Font is the font asset for a text item; empty for non-text items.
	Font asset.ID
	// TextSize is the font size in pixels before applying Scale.
	TextSize float64
	// TextWidth and TextHeight are the text dimensions measured by the font
	// before applying Scale.
	TextWidth, TextHeight float64
	// Position is the interpolated position in world space.
	Position geom.Vector2
	// Rotation is the rotation in world space, in radians.
	Rotation float64
	// Scale is the drawable's scale in world space.
	Scale geom.Vector2
	// FlipH and FlipV mirror texture items horizontally and vertically.
	FlipH, FlipV bool
	// Color tints texture pixels or sets the text or shape color. Its alpha
	// controls opacity; nil means no tint on a texture item.
	Color color.Color
	// Outline strokes a shape's border instead of filling it.
	Outline bool
	// StrokeWidth is the outline width in world units.
	StrokeWidth float64
}

// DrawList contains the camera view and sorted draw items for one frame.
type DrawList struct {
	// Camera is the resolved primary camera view.
	Camera CameraView
	// Items is the sorted set of drawables. When returned by [Collector.Collect],
	// the collector owns the slice and overwrites it on its next call.
	Items []DrawItem
}

type workingItem struct {
	DrawItem
	layer     uint8
	sortOrder int8
	worldY    float64 // interpolated, the Y-fallback sort key
}

// Collector resolves drawable assets, culls against the camera viewport, and
// sorts draw items for each frame. Reuse a Collector across frames.
type Collector struct {
	userCamera,
	engineCamera,
	sprites *core.Query
	working []workingItem
	// SortedItems backs [DrawList.Items]. It is reused on the next
	// [Collector.Collect] call; callers must not modify or retain it.
	SortedItems []DrawItem
}

// NewCollector creates a [Collector] for world. A user primary camera takes
// precedence over the engine camera.
func NewCollector(world *core.World) *Collector {
	// User-spawned primaries are preferred; the engine camera
	// (SpawnEngineCamera) is the fallback when none exists.
	userCamera := core.NewQuery(world).
		With(Camera{}, core.Transform{}, core.PrevTransform{}).
		Without(engineCamera{}).
		Where(func(e core.Entry) bool {
			cam, ok := e.Component[Camera]()
			if !ok {
				return false
			}
			return cam.Primary
		})
	engineCamera := core.NewQuery(world).
		With(Camera{}, core.Transform{}, core.PrevTransform{}, engineCamera{}).
		Where(func(e core.Entry) bool {
			cam, ok := e.Component[Camera]()
			if !ok {
				return false
			}
			return cam.Primary
		})

	sprites := core.NewQuery(world).
		With(Sprite{}, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
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

// Collect resolves the draw list for the current frame. It interpolates
// transforms using [core.Context.Alpha], culls to the camera viewport, and
// sorts items by layer, sort order, then world Y. Text bounds use
// [asset.FontData.Measure].
//
// If no primary camera is available, Collect returns an empty list without an
// error. Asset lookup and loading errors are returned; shapes require no assets.
func (c *Collector) Collect(ctx *core.Context) (DrawList, error) {
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
	camTransform, _ := entry.Component[core.Transform]()
	camPrev, _ := entry.Component[core.PrevTransform]()
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

	// The asset server resolves lazily: a frame whose drawables are
	// all shapes never touches it, so a shape-only world needs no
	// asset wiring at all.
	var server *asset.Server
	assetServer := func() (*asset.Server, error) {
		if server == nil {
			s, err := ctx.World.Resource[*asset.Server]()
			if err != nil {
				return nil, fmt.Errorf("castrum: collect: resolve asset server: %w", err)
			}
			server = s
		}
		return server, nil
	}

	c.working = c.working[:0]

	for e := range c.sprites.Execute() {
		sprite, _ := e.Component[Sprite]()
		transform, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()

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
			srv, err := assetServer()
			if err != nil {
				return DrawList{}, err
			}
			atlas, err := srv.Atlas(drawable.Atlas)
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
			srv, err := assetServer()
			if err != nil {
				return DrawList{}, err
			}
			data, err := srv.Load[asset.TextureData](string(drawable.Texture))
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
			srv, err := assetServer()
			if err != nil {
				return DrawList{}, err
			}
			font, err := srv.Load[asset.FontData](string(drawable.Font))
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
