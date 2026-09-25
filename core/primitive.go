package core

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Primitive is the shared style of a shape primitive — the shape
// counterpart of Sprite. Pair it with exactly one geometry variant
// (RectPrimitive, CirclePrimitive, or LinePrimitive), a Transform,
// and a PrevTransform, and the collector resolves it into a DrawItem
// carrying a Shape.
//
// The zero value is the shown, opaque, filled shape: a declared
// primitive draws unless told otherwise. Hide it with Hidden; see
// through it with Transparency; outline it with Outline.
type Primitive struct {
	// Layer orders the shape against every other drawable, 0-31,
	// back to front.
	Layer uint8
	// SortOrder orders within the layer: higher draws on top, with the
	// world-Y fallback below that.
	SortOrder int8
	// Hidden skips collection entirely. false — the zero value —
	// means shown.
	Hidden bool
	// Color is the fill or stroke color. nil — the zero value — draws
	// black.
	Color color.Color
	// Outline strokes the shape's border instead of filling it.
	// false — the zero value — fills, the prototyping default.
	// Outline requires a positive StrokeWidth, enforced by Validate.
	Outline bool
	// StrokeWidth is the outline's width in world units. Ignored
	// unless Outline is set.
	StrokeWidth float64
	// Transparency fades the shape: 0.0 — the zero value — is fully
	// opaque, 1.0 fully see-through.
	Transparency float32
}

func (p Primitive) Validate() error {
	if p.Layer > 31 {
		return fmt.Errorf("layer must be between 0 and 31")
	}
	if p.Outline && p.StrokeWidth <= 0 {
		return fmt.Errorf("outline requires a positive stroke width")
	}
	if p.StrokeWidth < 0 {
		return fmt.Errorf("stroke width must not be negative")
	}
	if p.Transparency < 0 || p.Transparency > 1 {
		return fmt.Errorf("transparency must be between 0 and 1")
	}
	return nil
}

// RectPrimitive is the rectangle geometry variant: Size is the full
// extent in world units, centered on the transform position and
// scaled by the transform's scale.
type RectPrimitive struct {
	Size geom.Vector2
}

func (r RectPrimitive) Validate() error {
	if r.Size.X <= 0 || r.Size.Y <= 0 {
		return fmt.Errorf("size must be positive")
	}
	return nil
}

// CirclePrimitive is the circle geometry variant: Radius in world
// units before transform scale.
type CirclePrimitive struct {
	Radius float64
}

func (c CirclePrimitive) Validate() error {
	if c.Radius <= 0 {
		return fmt.Errorf("radius must be positive")
	}
	return nil
}

// LinePrimitive is the line-segment geometry variant: To is the
// second endpoint relative to the transform position, so the whole
// line moves rigidly and interpolates for free — no second
// prev-state to manage.
type LinePrimitive struct {
	To geom.Vector2
}

func (l LinePrimitive) Validate() error {
	if l.To.IsZero() {
		return fmt.Errorf("endpoint must not be zero")
	}
	return nil
}

// Shape is a primitive's geometry on a DrawItem — the collector-facing
// mirror of the geometry variants. It is a sealed sum: the only
// implementations are RectShape, CircleShape, and LineShape, and the
// geometry variants map one-to-one onto them. nil on a DrawItem means
// the item is a texture sprite.
type Shape interface {
	isShape()
}

// RectShape draws a filled or outlined rectangle of Size world units,
// centered on the item's position and scaled by its scale.
type RectShape struct {
	Size geom.Vector2
}

// CircleShape draws a filled or outlined circle of Radius world units,
// centered on the item's position and scaled by its scale.
type CircleShape struct {
	Radius float64
}

// LineShape draws a segment from the item's position to
// position+To, scaled by the item's scale.
type LineShape struct {
	To geom.Vector2
}

func (RectShape) isShape()   {}
func (CircleShape) isShape() {}
func (LineShape) isShape()   {}
