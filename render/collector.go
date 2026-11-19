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

// DrawItem is one drawable resolved from a [sprite.Sprite], [sprite.Text],
// [sprite.Primitive], or [animation.AnimatedSprite] for a frame.
type DrawItem struct {
	// Texture is the source texture for a texture item; empty for text and shapes.
	Texture asset.ID
	// Rect is the source region in Texture, in pixels. An empty rectangle
	// selects the whole texture.
	Rect image.Rectangle
	// Shape is the geometry of a shape item; nil for texture and text items.
	Shape sprite.DrawShape
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
	// the collector owns the slice and overwrites it on its next call; callers
	// must not modify or retain it.
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
	camera          *CameraResolver
	images          *core.Query
	primitives      *core.Query
	texts           *core.Query
	animatedSprites *core.Query
	working         []workingItem
	// view is the published [*CameraView] resource; every Collect writes
	// the resolved view into it.
	view *CameraView
	// SortedItems is the storage behind [DrawList.Items] and follows its
	// ownership rule.
	SortedItems []DrawItem
}

// NewCollector creates a [Collector] for world. A user primary camera takes
// precedence over the engine camera.
func NewCollector(world *core.World) *Collector {
	images := core.NewQuery(world).
		With(sprite.Sprite{}, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
			s, ok := e.Component[sprite.Sprite]()
			return ok && !s.Style.Hidden
		})

	primitives := core.NewQuery(world).
		With(sprite.Primitive{}, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
			p, ok := e.Component[sprite.Primitive]()
			return ok && !p.Style.Hidden
		})

	texts := core.NewQuery(world).
		With(sprite.Text{}, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
			x, ok := e.Component[sprite.Text]()
			return ok && !x.Style.Hidden
		})

	animated := core.NewQuery(world).
		With(animation.AnimatedSprite{}, core.Transform{}, core.PrevTransform{}).
		Where(func(e core.Entry) bool {
			a, ok := e.Component[animation.AnimatedSprite]()
			return ok && !a.Style.Hidden
		})

	return &Collector{
		camera:          NewCameraResolver(world),
		images:          images,
		primitives:      primitives,
		texts:           texts,
		animatedSprites: animated,
		working:         make([]workingItem, 0),
		SortedItems:     make([]DrawItem, 0),
	}
}

// Collect resolves the draw list for the current frame. Items outside the
// camera view are culled, and the rest are sorted by layer, sort order, then
// world Y.
//
// When a primary camera is available, Collect publishes the resolved view as
// the world's [*CameraView] resource once collection succeeds: the resource
// always describes a drawn frame. A failed Collect publishes nothing; the
// resource keeps the previous frame's view, and a first Collect that fails
// registers no resource. Calls without a primary camera leave the resource
// unchanged.
//
// If no primary camera is available, Collect returns an empty list without an
// error. Asset lookup and loading errors are returned; shapes require no assets.
// Sheet documents, textures, and text resolve through the world's [asset.Loader].
// Sheet regions are checked against the decoded texture before staging.
// An entity may carry [sprite.Sprite], [sprite.Text], [sprite.Primitive],
// and [animation.AnimatedSprite] together; each contributes its own item.
// Among items with equal layer, sort order, and world Y, class order decides:
// sprites, then primitives, then text, then animated sprites.
func (c *Collector) Collect(ctx *core.Context) (DrawList, error) {
	camera, ok, err := c.camera.Resolve(ctx.Alpha, ctx.LogicalWidth, ctx.LogicalHeight)
	if err != nil {
		return DrawList{}, err
	}
	if !ok {
		return DrawList{}, nil
	}

	viewport := camera.aabb

	var loader *asset.Loader
	assetLoader := func() (*asset.Loader, error) {
		if loader == nil {
			s, err := ctx.World.Resource[*asset.Loader]()
			if err != nil {
				return nil, fmt.Errorf("castrum: collect: resolve asset loader: %w", err)
			}
			loader = s
		}
		return loader, nil
	}

	c.working = c.working[:0]

	for e := range c.images.Execute() {
		s, _ := e.Component[sprite.Sprite]()
		transform, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()
		if err := c.collectImage(viewport, ctx.Alpha, assetLoader, s, prev, transform); err != nil {
			return DrawList{}, err
		}
	}

	for e := range c.primitives.Execute() {
		p, _ := e.Component[sprite.Primitive]()
		transform, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()
		if err := c.collectPrimitive(viewport, ctx.Alpha, p, prev, transform); err != nil {
			return DrawList{}, err
		}
	}

	for e := range c.texts.Execute() {
		x, _ := e.Component[sprite.Text]()
		transform, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()
		if err := c.collectTextSprite(viewport, ctx.Alpha, assetLoader, x, prev, transform); err != nil {
			return DrawList{}, err
		}
	}

	for e := range c.animatedSprites.Execute() {
		a, _ := e.Component[animation.AnimatedSprite]()
		transform, _ := e.Component[core.Transform]()
		prev, _ := e.Component[core.PrevTransform]()
		ref, region, ok, err := a.CurrentFrame()
		if err != nil {
			return DrawList{}, fmt.Errorf("castrum: collect: animated sprite: entity %d: %w", e.ID(), err)
		}
		if !ok {
			// A clipless sprite is legal and draws nothing.
			continue
		}
		if transform.Scale.X == 0 {
			transform.Scale.X = 1
		}
		if transform.Scale.Y == 0 {
			transform.Scale.Y = 1
		}
		scale := prev.Scale.Lerp(transform.Scale, ctx.Alpha)
		rotation := prev.Rotation + (transform.Rotation-prev.Rotation)*ctx.Alpha
		item, size, err := c.collectSheet(assetLoader, a.Style, rotation, scale, ref, region)
		if err != nil {
			return DrawList{}, err
		}
		var zero geom.Vector2
		c.stage(viewport, ctx.Alpha, a.Style.Layer, a.Style.SortOrder,
			prev.Position, transform.Position, zero, size, item)
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

func (c *Collector) collectImage(viewport geom.Rect, alpha float64, loadAsset func() (*asset.Loader, error), s sprite.Sprite, prev core.PrevTransform, transform core.Transform) error {
	if s.Asset == "" {
		// Style without a picture: legal, not drawn.
		return nil
	}
	if transform.Scale.X == 0 {
		transform.Scale.X = 1
	}
	if transform.Scale.Y == 0 {
		transform.Scale.Y = 1
	}

	scale := prev.Scale.Lerp(transform.Scale, alpha)
	rotation := prev.Rotation + (transform.Rotation-prev.Rotation)*alpha

	var item DrawItem
	var offset, size geom.Vector2
	var err error
	if s.Region == "" {
		if err := c.collectTexture(loadAsset, s, rotation, scale, &item, &size); err != nil {
			return err
		}
	} else {
		item, size, err = c.collectSheet(loadAsset, s.Style, rotation, scale, asset.FileSheet(s.Asset), s.Region)
		if err != nil {
			return err
		}
	}
	c.stage(viewport, alpha, s.Style.Layer, s.Style.SortOrder,
		prev.Position, transform.Position, offset, size, item)
	return nil
}

func (c *Collector) collectPrimitive(viewport geom.Rect, alpha float64, p sprite.Primitive, prev core.PrevTransform, transform core.Transform) error {
	if p.Shape == sprite.ShapeNone {
		// Style without geometry: legal, not drawn.
		return nil
	}
	if transform.Scale.X == 0 {
		transform.Scale.X = 1
	}
	if transform.Scale.Y == 0 {
		transform.Scale.Y = 1
	}

	scale := prev.Scale.Lerp(transform.Scale, alpha)
	rotation := prev.Rotation + (transform.Rotation-prev.Rotation)*alpha

	var item DrawItem
	var offset, size geom.Vector2
	switch p.Shape {
	case sprite.ShapeRect:
		shape := sprite.RectShape{Size: p.Rect}
		size = geom.Vector2{
			X: shape.Size.X * scale.X,
			Y: shape.Size.Y * scale.Y,
		}
		item = shapeItem(p.Style, rotation, scale, shape)
	case sprite.ShapeCircle:
		shape := sprite.CircleShape{Radii: p.Radii}
		size = geom.Vector2{
			X: 2 * shape.Radii.X * scale.X,
			Y: 2 * shape.Radii.Y * scale.Y,
		}
		item = shapeItem(p.Style, rotation, scale, shape)
	case sprite.ShapeLine:
		shape := sprite.LineShape{From: p.From, To: p.To}
		segment := geom.Segment{
			Start: geom.Vector2{
				X: shape.From.X * scale.X,
				Y: shape.From.Y * scale.Y,
			},
			End: geom.Vector2{
				X: shape.To.X * scale.X,
				Y: shape.To.Y * scale.Y,
			},
		}
		bounds := segment.BoundingBox()
		offset, size = bounds.Center(), bounds.Size()
		item = shapeItem(p.Style, rotation, scale, shape)
	default:
		// Unreachable: Validate rejects unknown shapes at the boundary.
		return nil
	}
	c.stage(viewport, alpha, p.Style.Layer, p.Style.SortOrder,
		prev.Position, transform.Position, offset, size, item)
	return nil
}

func (c *Collector) collectTextSprite(viewport geom.Rect, alpha float64, loadAsset func() (*asset.Loader, error), x sprite.Text, prev core.PrevTransform, transform core.Transform) error {
	if x.Asset == "" || x.Text == "" {
		// No font or nothing to say: legal, not drawn.
		return nil
	}
	if transform.Scale.X == 0 {
		transform.Scale.X = 1
	}
	if transform.Scale.Y == 0 {
		transform.Scale.Y = 1
	}

	scale := prev.Scale.Lerp(transform.Scale, alpha)
	rotation := prev.Rotation + (transform.Rotation-prev.Rotation)*alpha

	var item DrawItem
	var offset, size geom.Vector2
	if err := c.collectText(loadAsset, x, rotation, scale, &item, &size); err != nil {
		return err
	}
	c.stage(viewport, alpha, x.Style.Layer, x.Style.SortOrder,
		prev.Position, transform.Position, offset, size, item)
	return nil
}

func (c *Collector) collectSheet(loadAsset func() (*asset.Loader, error), style sprite.Style, rotation float64, scale geom.Vector2, ref asset.SheetRef, region string) (DrawItem, geom.Vector2, error) {
	srv, err := loadAsset()
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
	size := geom.Vector2{
		X: float64(regionRect.Dx()) * scale.X,
		Y: float64(regionRect.Dy()) * scale.Y,
	}
	return textureItem(style, rotation, scale, sheet.Texture, regionRect), size, nil
}

func (c *Collector) collectTexture(loadAsset func() (*asset.Loader, error), s sprite.Sprite, rotation float64, scale geom.Vector2, item *DrawItem, size *geom.Vector2) error {
	srv, err := loadAsset()
	if err != nil {
		return err
	}
	data, err := srv.Load[asset.TextureData](string(s.Asset))
	if err != nil {
		return fmt.Errorf("castrum: collect: texture %q: %w", s.Asset, err)
	}
	*size = geom.Vector2{
		X: float64(data.Width) * scale.X,
		Y: float64(data.Height) * scale.Y,
	}
	*item = textureItem(s.Style, rotation, scale, s.Asset, image.Rectangle{})
	return nil
}

func (c *Collector) collectText(loadAsset func() (*asset.Loader, error), x sprite.Text, rotation float64, scale geom.Vector2, item *DrawItem, size *geom.Vector2) error {
	srv, err := loadAsset()
	if err != nil {
		return err
	}
	font, err := srv.Load[asset.FontData](string(x.Asset))
	if err != nil {
		return fmt.Errorf("castrum: collect: font %q: %w", x.Asset, err)
	}
	textWidth, textHeight := font.Measure(x.TextSize, x.Text)
	fill := styleTint(x.Style.Color, color.White)
	*size = geom.Vector2{
		X: textWidth * scale.X,
		Y: textHeight * scale.Y,
	}
	*item = DrawItem{
		Text:       x.Text,
		Font:       x.Asset,
		TextSize:   x.TextSize,
		TextWidth:  textWidth,
		TextHeight: textHeight,
		Rotation:   rotation,
		Scale:      scale,
		Color:      fill,
	}
	return nil
}

func (c *Collector) stage(viewport geom.Rect, alpha float64, layer uint8, sortOrder int8, prev, curr, offset, size geom.Vector2, item DrawItem) {
	position := prev.Lerp(curr, alpha)
	half := geom.Vector2{X: math.Abs(size.X) / 2, Y: math.Abs(size.Y) / 2}
	_, line := item.Shape.(sprite.LineShape)
	if item.Outline || line {
		half.X += item.StrokeWidth / 2
		half.Y += item.StrokeWidth / 2
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
		Texture:  texture,
		Rect:     region,
		Rotation: rotation,
		Scale:    scale,
		FlipH:    style.FlipH,
		FlipV:    style.FlipV,
		Color:    tint,
	}
}

func shapeItem(style sprite.Style, rotation float64, scale geom.Vector2, shape sprite.DrawShape) DrawItem {
	fill := styleTint(style.Color, color.Black)
	return DrawItem{
		Shape:       shape,
		Rotation:    rotation,
		Scale:       scale,
		Color:       fill,
		Outline:     style.Outline,
		StrokeWidth: style.StrokeWidth,
	}
}

func styleTint(clr *color.RGBA, fallback color.Color) color.Color {
	if clr == nil {
		return fallback
	}
	return *clr
}
