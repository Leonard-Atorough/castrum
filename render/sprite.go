package render

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Sprite is a drawable component with shared style and an optional drawable.
//
// The zero value is visible but draws nothing until [Sprite.Drawable] is set.
type Sprite struct {
	// Layer orders sprites from back to front, from 0 through 31.
	Layer uint8
	// SortOrder orders sprites within a layer; higher values draw on top.
	// Equal values are ordered by world Y.
	SortOrder int8
	// Hidden excludes the sprite from collection.
	Hidden bool
	// FlipH and FlipV mirror texture-backed drawables horizontally and
	// vertically; shapes are not flipped.
	FlipH, FlipV bool
	// Drawable is the image, text, or shape to draw. A nil value draws nothing.
	Drawable Drawable
	// Outline strokes a shape's border instead of filling it. It requires a
	// positive StrokeWidth and has no effect on non-shape drawables.
	Outline bool
	// StrokeWidth is the outline width in world units. It must not be negative
	// and must be positive when Outline is set.
	StrokeWidth float64
	// Color tints texture pixels or sets the text or shape color. Its alpha
	// controls opacity. A nil value leaves textures untinted, defaults text
	// to white, and defaults shapes to black.
	Color color.Color
}

// Validate reports whether the sprite's style and drawable parameters are
// valid. It does not resolve referenced assets.
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

// Drawable is the image, text, or shape rendered by a [Sprite]. Its
// implementations are [AtlasSource],
// [TextureSource], [TextSource], [RectShape], [CircleShape], and [LineShape].
type Drawable interface {
	isDrawable()
}

// AtlasSource draws a named region of a registered atlas.
type AtlasSource struct {
	// Atlas is the ID of the registered atlas.
	Atlas asset.AtlasID
	// Region is the name of the region to draw.
	Region string
}

// TextureSource draws a whole standalone texture.
type TextureSource struct {
	// Texture is the ID of the texture asset to draw.
	Texture asset.ID
}

// TextSource draws one line of text centered on the sprite's position. The
// sprite's scale applies to the font size.
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
