// Package render defines camera state and collects sprites into ordered
// [DrawList] values for a renderer to draw.
package render

import (
	"cmp"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// DrawList contains the camera view and sorted draw items for one frame.
type DrawList struct {
	// Camera is the resolved primary camera view.
	Camera CameraView
	// Items is the sorted set of drawables.
	// When returned by [Collector.Collect], the slice is collector-owned; copy it
	// before modifying it or retaining it across later collection calls.
	Items []DrawItem
}

type workingItem struct {
	DrawItem
	layer     uint8
	sortOrder int8
	worldY    float64 // interpolated, the Y-fallback sort key
}

var zeroVec geom.Vector2

// Collector collects draw items for a world and can be reused across frames.
type Collector struct {
	camera     *CameraResolver
	win        Window
	loader     *asset.Loader
	images     *core.Query
	primitives *core.Query
	texts      *core.Query
	animators  *core.Query
	working    []workingItem
	// view is the published [CameraView] resource; successful collection
	// with a camera updates it.
	view *CameraView
	// SortedItems is the storage behind the items returned by [Collector.Collect].
	SortedItems []DrawItem
}

func visibleQuery[C any](world *core.World, visible func(C) bool) *core.Query {
	var marker C
	return core.NewQuery(world).
		With(marker, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
			c, ok := e.Component[C]()
			return ok && visible(c)
		})
}

// NewCollector creates a [Collector] for world.
func NewCollector(world *core.World) *Collector {
	hidden := func(s sprite.Style) bool { return s.Hidden }
	return &Collector{
		camera: NewCameraResolver(world),
		images: visibleQuery(world, func(s sprite.Sprite) bool {
			return !hidden(s.Style)
		}),
		primitives: visibleQuery(world, func(p sprite.Primitive) bool {
			return !hidden(p.Style)
		}),
		texts: visibleQuery(world, func(x sprite.Text) bool {
			return !hidden(x.Style)
		}),
		animators: visibleQuery(world, func(a animation.Animator) bool {
			return !hidden(a.Style)
		}),
		working:     make([]workingItem, 0),
		SortedItems: make([]DrawItem, 0),
	}
}

func collectAll[C any](q *core.Query, fn func(e core.Entry, comp C, prev core.PrevTransform, tr core.Transform) error) error {
	for e := range q.Execute() {
		comp, _ := e.Component[C]()
		tr, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()
		if err := fn(e, comp, prev, tr); err != nil {
			return err
		}
	}
	return nil
}

func interpolate(prev core.PrevTransform, tr core.Transform, alpha float64) (geom.Vector2, float64, geom.Vector2) {
	if tr.Scale.X == 0 {
		tr.Scale.X = 1
	}
	if tr.Scale.Y == 0 {
		tr.Scale.Y = 1
	}
	return prev.Position.Lerp(tr.Position, alpha),
		prev.Rotation + (tr.Rotation-prev.Rotation)*alpha,
		prev.Scale.Lerp(tr.Scale, alpha)
}

// Collect resolves the draw list for the current frame using alpha to
// interpolate transforms. An alpha of 1 uses current transforms.
//
// Items outside the camera view are culled; the rest are sorted by layer,
// sort order, and world Y. Ties preserve class order: sprites, primitives,
// text, then animators. An animator contributes one item per weighted frame
// layer.
//
// If no primary camera is available, Collect returns an empty list without an
// error. A successful collection with a camera publishes a [CameraView];
// failure or absence of a camera leaves the previously published view unchanged.
// It returns an error if camera, resource, asset, or animation resolution fails.
func (c *Collector) Collect(ctx *core.Context, alpha float64) (DrawList, error) {
	if c.win == nil {
		win, err := ctx.World.Resource[Window]()
		if err != nil {
			return DrawList{}, fmt.Errorf("castrum: collect: resolve window: %w", err)
		}
		c.win = win
	}
	vp := c.win.Viewport()
	camera, ok, err := c.camera.Resolve(alpha, vp.LogicalWidth, vp.LogicalHeight)
	if err != nil {
		return DrawList{}, err
	}
	if !ok {
		return DrawList{}, nil
	}

	viewport := camera.aabb
	c.working = c.working[:0]

	if err := collectAll(c.images, func(_ core.Entry, s sprite.Sprite, prev core.PrevTransform, tr core.Transform) error {
		return c.collectImage(ctx, viewport, alpha, s, prev, tr)
	}); err != nil {
		return DrawList{}, err
	}

	if err := collectAll(c.primitives, func(_ core.Entry, p sprite.Primitive, prev core.PrevTransform, tr core.Transform) error {
		return c.collectPrimitive(viewport, alpha, p, prev, tr)
	}); err != nil {
		return DrawList{}, err
	}

	if err := collectAll(c.texts, func(_ core.Entry, x sprite.Text, prev core.PrevTransform, tr core.Transform) error {
		return c.collectTextItem(ctx, viewport, alpha, x, prev, tr)
	}); err != nil {
		return DrawList{}, err
	}

	var sets animation.Sets
	if _, ok := c.animators.First(); ok {
		if ctx.World.HasResource[animation.Sets]() {
			sets, err = ctx.World.Resource[animation.Sets]()
			if err != nil {
				return DrawList{}, fmt.Errorf("castrum: collect: resolve animation sets: %w", err)
			}
		}
	}
	if err := collectAll(c.animators, func(e core.Entry, a animation.Animator, prev core.PrevTransform, tr core.Transform) error {
		frames, err := a.Frames(sets)
		if err != nil {
			return fmt.Errorf("castrum: collect: animator: entity %d: %w", e.ID(), err)
		}
		position, rotation, scale := interpolate(prev, tr, alpha)
		for _, frame := range frames {
			style := a.Style
			if frame.Weight < 1 {
				style.Color = blendAlpha(style.Color, frame.Weight)
			}
			item, texSize, err := c.collectSheet(ctx, style, rotation, scale, frame.Source, frame.Region)
			if err != nil {
				return err
			}
			c.stage(viewport, a.Style.Layer, a.Style.SortOrder, position, texSize, item)
		}
		return nil
	}); err != nil {
		return DrawList{}, err
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

	if err := c.initializeView(ctx.World); err != nil {
		return DrawList{}, err
	}
	*c.view = camera
	return DrawList{Camera: camera, Items: c.SortedItems}, nil
}

func (c *Collector) assetLoader(ctx *core.Context) (*asset.Loader, error) {
	if c.loader == nil {
		loader, err := ctx.World.Resource[*asset.Loader]()
		if err != nil {
			return nil, fmt.Errorf("castrum: collect: resolve asset loader: %w", err)
		}
		c.loader = loader
	}
	return c.loader, nil
}

func (c *Collector) collectImage(ctx *core.Context, viewport geom.Rect, alpha float64, s sprite.Sprite, prev core.PrevTransform, tr core.Transform) error {
	if s.Asset == "" {
		// Style without a picture: legal, not drawn.
		return nil
	}
	position, rotation, scale := interpolate(prev, tr, alpha)

	var item DrawItem
	var texSize geom.Vector2
	var err error
	if s.Region == "" {
		item, texSize, err = c.collectTexture(ctx, s, rotation, scale)
	} else {
		item, texSize, err = c.collectSheet(ctx, s.Style, rotation, scale, asset.FileSheet(s.Asset), s.Region)
	}
	if err != nil {
		return err
	}
	c.stage(viewport, s.Style.Layer, s.Style.SortOrder, position, texSize, item)
	return nil
}

func (c *Collector) collectPrimitive(viewport geom.Rect, alpha float64, p sprite.Primitive, prev core.PrevTransform, tr core.Transform) error {
	if p.Shape == sprite.ShapeNone {
		// Style without geometry: legal, not drawn.
		return nil
	}
	position, rotation, scale := interpolate(prev, tr, alpha)

	var item DrawItem
	switch p.Shape {
	case sprite.ShapeRect:
		item = shapeItem(p.Style, rotation, scale, sprite.RectShape{Size: p.Rect})
	case sprite.ShapeCircle:
		item = shapeItem(p.Style, rotation, scale, sprite.CircleShape{Radii: p.Radii})
	case sprite.ShapeLine:
		item = shapeItem(p.Style, rotation, scale, sprite.LineShape{From: p.From, To: p.To})
	default:
		// Unreachable: Validate rejects unknown shapes at the boundary.
		return nil
	}
	c.stage(viewport, p.Style.Layer, p.Style.SortOrder, position, zeroVec, item)
	return nil
}

func (c *Collector) collectTextItem(ctx *core.Context, viewport geom.Rect, alpha float64, x sprite.Text, prev core.PrevTransform, tr core.Transform) error {
	if x.Asset == "" || x.Text == "" {
		// No font or nothing to say: legal, not drawn.
		return nil
	}
	position, rotation, scale := interpolate(prev, tr, alpha)

	item, err := c.collectText(ctx, x, rotation, scale)
	if err != nil {
		return err
	}
	c.stage(viewport, x.Style.Layer, x.Style.SortOrder, position, zeroVec, item)
	return nil
}

func (c *Collector) collectSheet(ctx *core.Context, style sprite.Style, rotation float64, scale geom.Vector2, ref asset.SheetRef, region string) (DrawItem, geom.Vector2, error) {
	srv, err := c.assetLoader(ctx)
	if err != nil {
		return DrawItem{}, geom.Vector2{}, err
	}
	sheet, err := ref.Resolve(srv)
	if err != nil {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: %w", err)
	}
	regionRect, err := sheet.Region(region)
	if err != nil {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: %w", err)
	}
	texture, err := srv.Load[asset.TextureData](string(sheet.Texture))
	if err != nil {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: sheet region %q: %w", region, err)
	}
	if texture.Image == nil {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: sheet texture %q has no decoded image", sheet.Texture)
	}
	if !regionRect.In(texture.Image.Bounds()) {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: sheet region %q falls outside texture %q bounds %v",
			region, sheet.Texture, texture.Image.Bounds())
	}
	texSize := geom.Vector2{X: float64(texture.Width), Y: float64(texture.Height)}
	return textureItem(style, rotation, scale, sheet.Texture, regionRect), texSize, nil
}

func (c *Collector) collectTexture(ctx *core.Context, s sprite.Sprite, rotation float64, scale geom.Vector2) (DrawItem, geom.Vector2, error) {
	srv, err := c.assetLoader(ctx)
	if err != nil {
		return DrawItem{}, geom.Vector2{}, err
	}
	data, err := srv.Load[asset.TextureData](string(s.Asset))
	if err != nil {
		return DrawItem{}, geom.Vector2{}, fmt.Errorf("castrum: collect: texture %q: %w", s.Asset, err)
	}
	texSize := geom.Vector2{X: float64(data.Width), Y: float64(data.Height)}
	return textureItem(s.Style, rotation, scale, s.Asset, image.Rectangle{}), texSize, nil
}

func (c *Collector) collectText(ctx *core.Context, x sprite.Text, rotation float64, scale geom.Vector2) (DrawItem, error) {
	srv, err := c.assetLoader(ctx)
	if err != nil {
		return DrawItem{}, err
	}
	font, err := srv.Load[asset.FontData](string(x.Asset))
	if err != nil {
		return DrawItem{}, fmt.Errorf("castrum: collect: font %q: %w", x.Asset, err)
	}
	textWidth, textHeight := font.Measure(x.TextSize, x.Text)
	fill := styleTint(x.Style.Color, color.White)
	item := DrawItem{
		Rotation: rotation,
		Scale:    scale,
		Color:    fill,
		payload: TextDraw{
			Text:   x.Text,
			Font:   x.Asset,
			Size:   x.TextSize,
			Bounds: geom.Vector2{X: textWidth, Y: textHeight},
		},
	}
	return item, nil
}

func payloadBounds(p DrawPayload, scale, texSize geom.Vector2) (offset, size geom.Vector2, expand float64) {
	switch p := p.(type) {
	case TextureDraw:
		size = texSize
		if !p.Rect.Empty() {
			size = geom.Vector2{X: float64(p.Rect.Dx()), Y: float64(p.Rect.Dy())}
		}
		size = geom.Vector2{X: size.X * scale.X, Y: size.Y * scale.Y}
	case TextDraw:
		size = geom.Vector2{X: p.Bounds.X * scale.X, Y: p.Bounds.Y * scale.Y}
	case ShapeDraw:
		switch shape := p.Shape.(type) {
		case sprite.RectShape:
			size = geom.Vector2{X: shape.Size.X * scale.X, Y: shape.Size.Y * scale.Y}
		case sprite.CircleShape:
			size = geom.Vector2{X: 2 * shape.Radii.X * scale.X, Y: 2 * shape.Radii.Y * scale.Y}
		case sprite.LineShape:
			bounds := geom.Segment{
				Start: geom.Vector2{X: shape.From.X * scale.X, Y: shape.From.Y * scale.Y},
				End:   geom.Vector2{X: shape.To.X * scale.X, Y: shape.To.Y * scale.Y},
			}.BoundingBox()
			offset, size = bounds.Center(), bounds.Size()
		}
		if _, line := p.Shape.(sprite.LineShape); p.Outline || line {
			expand = p.StrokeWidth / 2
		}
	}
	return offset, size, expand
}

func (c *Collector) stage(viewport geom.Rect, layer uint8, sortOrder int8, position, texSize geom.Vector2, item DrawItem) {
	offset, size, expand := payloadBounds(item.payload, item.Scale, texSize)
	half := geom.Vector2{
		X: math.Abs(size.X)/2 + expand,
		Y: math.Abs(size.Y)/2 + expand,
	}
	sin, cos := math.Sincos(item.Rotation)
	extent := geom.Vector2{
		X: math.Abs(cos)*half.X + math.Abs(sin)*half.Y,
		Y: math.Abs(sin)*half.X + math.Abs(cos)*half.Y,
	}
	center := position.Add(geom.Vector2{
		X: offset.X*cos - offset.Y*sin,
		Y: offset.X*sin + offset.Y*cos,
	})
	bounds := geom.RectFromCenterSize(center, extent.Mul(2))
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

func (c *Collector) initializeView(world *core.World) error {
	if c.view != nil {
		return nil
	}
	existing, err := world.Resource[*CameraView]()
	if err == nil {
		c.view = existing
		return nil
	}
	if !errors.Is(err, core.ErrResourceNotRegistered) {
		return fmt.Errorf("castrum: collect: resolve existing camera view: %w", err)
	}
	view := &CameraView{}
	if err := world.Provide(func(*core.World) (*CameraView, error) {
		return view, nil
	}); err != nil {
		return fmt.Errorf("castrum: collect: provide camera view resource: %w", err)
	}
	c.view = view
	return nil
}

func textureItem(style sprite.Style, rotation float64, scale geom.Vector2, texture asset.ID, region image.Rectangle) DrawItem {
	var tint color.Color
	if style.Color != nil {
		tint = *style.Color
	}
	return DrawItem{
		Rotation: rotation,
		Scale:    scale,
		Color:    tint,
		payload: TextureDraw{
			Texture: texture,
			Rect:    region,
			FlipH:   style.FlipH,
			FlipV:   style.FlipV,
		},
	}
}

func shapeItem(style sprite.Style, rotation float64, scale geom.Vector2, shape sprite.DrawShape) DrawItem {
	fill := styleTint(style.Color, color.Black)
	return DrawItem{
		Rotation: rotation,
		Scale:    scale,
		Color:    fill,
		payload: ShapeDraw{
			Shape:       shape,
			Outline:     style.Outline,
			StrokeWidth: style.StrokeWidth,
		},
	}
}

// blendAlpha returns a tint whose alpha is scaled by a frame layer's blend
// weight; a nil style color becomes opaque white at that weight.
func blendAlpha(clr *color.RGBA, weight float64) *color.RGBA {
	out := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if clr != nil {
		out = *clr
	}
	out.A = uint8(math.Round(float64(out.A) * weight))
	return &out
}

func styleTint(clr *color.RGBA, fallback color.Color) color.Color {
	if clr == nil {
		return fallback
	}
	return *clr
}
