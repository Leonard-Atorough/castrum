package render

import (
	"image"
	"image/color"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// DrawItem describes one drawable for a frame.
type DrawItem struct {
	// Position is the interpolated position in world space.
	Position geom.Vector2
	// Rotation is the rotation in world space, in radians.
	Rotation float64
	// Scale is the drawable's scale in world space.
	Scale geom.Vector2
	// Color tints texture pixels or sets the text or shape color. Its alpha
	// controls opacity; nil means no tint on a texture item.
	Color color.Color

	payload DrawPayload
}

// Payload returns the draw data for the item, or nil for a zero [DrawItem].
func (d DrawItem) Payload() DrawPayload {
	return d.payload
}

// DrawPayload holds the kind-specific data for a [DrawItem].
// The collector emits [TextureDraw], [TextDraw], or [ShapeDraw] values.
type DrawPayload interface{ drawPayload() }

// TextureDraw describes a texture or a region of a texture to draw.
type TextureDraw struct {
	// Texture identifies the source texture.
	Texture asset.ID
	// Rect selects the source region in pixels; an empty rectangle selects the whole texture.
	Rect image.Rectangle
	// FlipH mirrors the image horizontally without changing the item transform.
	FlipH bool
	// FlipV mirrors the image vertically without changing the item transform.
	FlipV bool
}

// TextDraw describes measured text ready to draw.
type TextDraw struct {
	// Text is the text to draw.
	Text string
	// Font is the font asset for the text.
	Font asset.ID
	// Size is the font size in pixels.
	Size float64
	// Bounds is the measured width and height in pixels before applying scale.
	Bounds geom.Vector2
}

// ShapeDraw describes a geometric shape to draw.
type ShapeDraw struct {
	// Shape supplies the geometry in local space.
	Shape sprite.DrawShape
	// Outline draws the shape's border instead of filling it; it has no effect on lines.
	Outline bool
	// StrokeWidth is the line or outline width in world units.
	StrokeWidth float64
}

func (TextureDraw) drawPayload() {}
func (TextDraw) drawPayload()    {}
func (ShapeDraw) drawPayload()   {}
