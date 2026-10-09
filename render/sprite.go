package render

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Sprite is a drawable component with shared style and one optional picture
// or shape.
//
// The zero value is visible but draws nothing until [Sprite.Drawable] is set.
type Sprite struct {
	// Layer orders drawables from back to front and must be between 0 and 31.
	Layer uint8
	// SortOrder orders drawables within a layer; higher values draw on top.
	// Equal values are ordered by world Y.
	SortOrder int8
	// Hidden excludes the sprite from collection. The zero value is shown.
	Hidden bool
	// FlipH and FlipV mirror texture-backed drawables horizontally and
	// vertically. Shapes are not flipped.
	FlipH, FlipV bool
	// Drawable is what to draw. A nil value leaves the sprite without a
	// picture, so it is not drawn.
	Drawable Drawable
	// Outline strokes a shape's border instead of filling it. It requires a
	// positive StrokeWidth and has no effect on other drawable types.
	Outline bool
	// StrokeWidth is the outline width in world units. It must not be
	// negative and must be positive when outlining a shape.
	StrokeWidth float64
	// Color tints texture pixels or sets a shape's fill and stroke color.
	// Its alpha controls opacity. A nil color leaves textures untinted,
	// defaults text to white, and defaults shapes to black.
	Color color.Color
}

// Validate reports whether the sprite's style and drawable parameters are
// valid. It does not check whether referenced assets can be loaded.
func (s Sprite) Validate() error {
	if s.Layer > 31 {
		return fmt.Errorf("layer must be between 0 and 31")
	}
	if s.Outline && s.StrokeWidth <= 0 {
		return fmt.Errorf("outline requires a positive stroke width")
	}
	if s.StrokeWidth < 0 {
		return fmt.Errorf("stroke width must not be negative")
	}
	switch d := s.Drawable.(type) {
	case nil:
		// Style without a picture: legal, not drawn.
	case AtlasSource:
		if d.Atlas == "" {
			return fmt.Errorf("atlas source requires an atlas ID")
		}
		if d.Region == "" {
			return fmt.Errorf("atlas source requires a region")
		}
	case TextureSource:
		if d.Texture == "" {
			return fmt.Errorf("texture source requires a texture")
		}
	case RectShape:
		if d.Size.X <= 0 || d.Size.Y <= 0 {
			return fmt.Errorf("rect size must be positive")
		}
	case CircleShape:
		if d.Radii.X <= 0 || d.Radii.Y <= 0 {
			return fmt.Errorf("circle radii must be positive")
		}
	case LineShape:
		if d.From == d.To {
			return fmt.Errorf("line endpoints must be distinct")
		}
	case TextSource:
		if d.Font == "" {
			return fmt.Errorf("text source requires a font")
		}
		if d.Size <= 0 || math.IsNaN(d.Size) || math.IsInf(d.Size, 0) {
			return fmt.Errorf("text size must be a positive number of pixels")
		}
		if strings.ContainsAny(d.Text, "\r\n") {
			return fmt.Errorf("text source is single-line; use one sprite per line")
		}
	default:
		return fmt.Errorf("unknown drawable %T", s.Drawable)
	}
	return nil
}

// Drawable is the source rendered by a [Sprite]: an atlas region, texture,
// text string, or shape. Its implementations are [AtlasSource],
// [TextureSource], [TextSource], [RectShape], [CircleShape], and [LineShape].
type Drawable interface {
	isDrawable()
}

// AtlasSource draws one named region of a registered atlas.
type AtlasSource struct {
	// Atlas is the ID of the registered atlas.
	Atlas asset.AtlasID
	// Region is the name of the region to draw.
	Region string
}

// TextureSource draws a standalone texture, whole.
type TextureSource struct {
	// Texture is the ID of the texture asset to draw.
	Texture asset.ID
}

// TextSource draws a single line of text centered on the sprite's position.
// The sprite's scale applies to the font size like it does to other drawables.
type TextSource struct {
	// Font is the ID of the font asset to render with.
	Font asset.ID
	// Text is the single line to draw. Empty text draws nothing.
	Text string
	// Size is the font size in pixels before sprite scaling.
	Size float64
}

// Shape is a [Drawable] defined by geometry rather than an image.
// Implementations are [RectShape], [CircleShape], and [LineShape].
type Shape interface {
	Drawable
	isShape()
}

// RectShape is a rectangle centered on the sprite's position. Its fill or
// outline comes from the sprite's style.
type RectShape struct {
	// Size is the rectangle's width and height in world units.
	Size geom.Vector2
}

// CircleShape is a circle centered on the sprite's position, or an ellipse
// when its radii differ. Its fill or outline comes from the sprite's style.
type CircleShape struct {
	// Radii are the circle's horizontal and vertical radii in world units.
	Radii geom.Vector2
}

// LineShape is a segment whose endpoints are relative to the sprite's
// position.
type LineShape struct {
	// From is the segment's start point relative to the sprite's position.
	From geom.Vector2
	// To is the segment's end point relative to the sprite's position.
	To geom.Vector2
}

func (AtlasSource) isDrawable()   {}
func (TextSource) isDrawable()    {}
func (TextureSource) isDrawable() {}
func (RectShape) isDrawable()     {}
func (CircleShape) isDrawable()   {}
func (LineShape) isDrawable()     {}

func (RectShape) isShape()   {}
func (CircleShape) isShape() {}
func (LineShape) isShape()   {}
