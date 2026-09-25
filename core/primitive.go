package core

import (
	"fmt"
	"image/color"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Primitive is a shape primitive: one drawable component carrying
// both the style and the geometry. Pair it with a Transform and a
// PrevTransform, and the collector resolves it into a DrawItem
// carrying the same Shape.
//
// The style's zero value is the shown, opaque, filled shape: a
// declared primitive draws unless told otherwise. The geometry is
// never zero — Validate rejects a nil Shape and degenerate geometry
// at spawn.
type Primitive struct {
	// Layer orders the shape against every other drawable, 0-31,
	// back to front.
	Layer uint8
	// SortOrder orders within the layer: higher draws on top, with
	// the world-Y fallback below that.
	SortOrder int8
	// Hidden skips collection entirely. false — the zero value —
	// means shown.
	Hidden bool
	// Shape is the geometry to draw: RectShape, CircleShape, or
	// LineShape. Required — a nil Shape fails Validate at spawn.
	Shape Shape
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
	switch shape := p.Shape.(type) {
	case nil:
		return fmt.Errorf("shape is required")
	case RectShape:
		if shape.Size.X <= 0 || shape.Size.Y <= 0 {
			return fmt.Errorf("rect size must be positive")
		}
	case CircleShape:
		if shape.Radius <= 0 {
			return fmt.Errorf("circle radius must be positive")
		}
	case LineShape:
		if shape.To.IsZero() {
			return fmt.Errorf("line endpoint must not be zero")
		}
	default:
		return fmt.Errorf("unknown shape %T", p.Shape)
	}
	return nil
}

// Shape is a primitive's geometry, carried on a Primitive or a
// DrawItem. It is a sealed sum: the only implementations are
// RectShape, CircleShape, and LineShape, and Validate checks each
// one's geometry at spawn. Carrying the sum on the component (rather
// than splitting geometry across variant components) keeps one
// collector query for every shape and makes exactly-one-geometry a
// construction guarantee instead of a convention.
type Shape interface {
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

func (RectShape) isShape()   {}
func (CircleShape) isShape() {}
func (LineShape) isShape()   {}
