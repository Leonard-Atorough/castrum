package core

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Sprite is the single drawable component: the shared style plus what
// to draw. The Drawable sum carries the picture — an atlas region or
// a standalone texture — or a shape: rect, circle, or line. Pair it
// with a Transform and a PrevTransform and the collector renders it,
// interpolated, culled, and sorted.
//
// The zero value is the shown, opaque sprite. With a nil Drawable it
// declares style but no picture and simply does not draw until a
// Drawable is set. Hide it with Hidden; see through it with
// Transparency; outline shapes with Outline.
type Sprite struct {
	// Layer orders the sprite against every other drawable, 0-31,
	// back to front.
	Layer uint8
	// SortOrder orders within the layer: higher draws on top, with
	// the world-Y fallback below that.
	SortOrder int8
	// Hidden skips collection entirely. false — the zero value —
	// means shown.
	Hidden bool
	// FlipH and FlipV mirror the sprite horizontally and vertically.
	// Applies when Drawable is a texture source; shapes have no flip.
	FlipH, FlipV bool
	// Drawable is what to draw. nil means style without a picture:
	// the sprite is legal but not drawn.
	Drawable Drawable
	// Outline strokes the shape's border instead of filling it.
	// Applies when Drawable is a shape; false — the zero value —
	// fills, the prototyping default. Outline requires a positive
	// StrokeWidth, enforced by Validate.
	Outline bool
	// StrokeWidth is the outline's width in world units. Applies when
	// Drawable is a shape and Outline is set.
	StrokeWidth float64
	// Transparency fades the sprite: 0.0 — the zero value — is fully
	// opaque, 1.0 fully see-through.
	Transparency float32
	// Tint is the color to multiply a texture's pixels by, or a
	// shape's fill and stroke color. nil means no tint on a texture
	// source; shapes default it to black at collection.
	Tint color.Color
}

func (s Sprite) Validate() error {
	if s.Layer > 31 {
		return fmt.Errorf("layer must be between 0 and 31")
	}
	if s.Transparency < 0 || s.Transparency > 1 {
		return fmt.Errorf("transparency must be between 0 and 1")
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
		if d.Radius <= 0 {
			return fmt.Errorf("circle radius must be positive")
		}
	case LineShape:
		if d.To.IsZero() {
			return fmt.Errorf("line endpoint must not be zero")
		}
	default:
		return fmt.Errorf("unknown drawable %T", s.Drawable)
	}
	return nil
}

// Drawable is what a Sprite draws: one of the two texture sources or
// one of the three shapes. It is a sealed sum — the only
// implementations are in this file — so exactly-one-drawable is a
// construction guarantee: a sprite cannot declare two pictures, and
// the collector type-switches one field instead of querying per
// variant.
type Drawable interface {
	isDrawable()
}

// AtlasSource draws one named region of a registered atlas.
type AtlasSource struct {
	Atlas  asset.AtlasID
	Region string
}

// TextureSource draws a standalone texture, whole.
type TextureSource struct {
	Texture asset.ID
}

// Shape is a Drawable that is geometry instead of an image — the sum
// a DrawItem carries when it resolved from a shape sprite. Sealed:
// RectShape, CircleShape, and LineShape.
type Shape interface {
	Drawable
	isShape()
}

// RectShape draws a filled or outlined rectangle of Size world units,
// centered on the position and scaled by the scale.
type RectShape struct {
	Size geom.Vector2
}

// CircleShape draws a filled or outlined circle of Radius world
// units, centered on the position and scaled by the scale.
type CircleShape struct {
	Radius float64
}

// LineShape draws a segment from the position to position+To, scaled
// by the scale. To is relative, so the whole line moves rigidly and
// interpolates for free.
type LineShape struct {
	To geom.Vector2
}

func (AtlasSource) isDrawable()   {}
func (TextureSource) isDrawable() {}
func (RectShape) isDrawable()     {}
func (CircleShape) isDrawable()   {}
func (LineShape) isDrawable()     {}

func (RectShape) isShape()   {}
func (CircleShape) isShape() {}
func (LineShape) isShape()   {}
