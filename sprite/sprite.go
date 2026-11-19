// Package sprite defines the visual components: one component per visual
// concept - [Sprite] for textures and sheet regions, [Text] for text, and
// [Primitive] for geometry.
//
// Each component is plain data with JSON tags and holds asset IDs, never
// runtime pointers; asset resolution happens in the render collector. An
// entity may carry components from more than one kind - a Sprite and a Text,
// say - and each contributes its own draw item. Each kind permits at most
// one component per entity, per the world's one-value-per-type rule.
package sprite

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Style controls sprite ordering, color, outline, flips, and visibility.
// The zero value is a visible, plain sprite on layer 0.
type Style struct {
	// Outline strokes a primitive's border instead of filling it. It requires a
	// positive StrokeWidth and has no effect on non-primitive components.
	Outline bool `json:"outline,omitempty"`
	// StrokeWidth is the outline width in world units. It must not be
	// negative and must be positive when Outline is set.
	StrokeWidth float64 `json:"strokeWidth,omitempty"`
	// Color tints texture pixels or sets the text or primitive color. A nil
	// value leaves textures untinted, defaults text to white, and defaults
	// primitives to black; an RGBA with zero alpha is explicitly transparent.
	Color *color.RGBA `json:"color,omitempty"`
	// Layer orders sprites from back to front, from 0 through 31.
	Layer uint8 `json:"layer,omitempty"`
	// SortOrder orders visuals within a layer; higher values draw on top.
	// Equal values are ordered by world Y.
	SortOrder int8 `json:"sortOrder,omitempty"`
	// Hidden excludes the component from collection.
	Hidden bool `json:"hidden,omitempty"`
	// FlipH mirrors Sprite's texture or sheet region horizontally; text
	// and primitives are not flipped.
	FlipH bool `json:"flipH,omitempty"`
	// FlipV mirrors Sprite's texture or sheet region vertically.
	FlipV bool `json:"flipV,omitempty"`
}

// Validate reports an error if the layer or stroke width is invalid.
func (st Style) Validate() error {
	if st.Layer > 31 {
		return fmt.Errorf("layer must be 0-31")
	}
	if !isFinite(st.StrokeWidth) {
		return fmt.Errorf("stroke width must be finite")
	}
	if st.Outline && st.StrokeWidth <= 0 {
		return fmt.Errorf("outline requires a positive stroke width")
	}
	if st.StrokeWidth < 0 {
		return fmt.Errorf("stroke width must not be negative")
	}
	return nil
}

// Sprite draws an image: a whole texture named by Asset, or, when Region is
// set, the named Region of the sheet document named by Asset. Region
// presence selects the source: an empty Asset draws nothing.
//
// The zero value is valid and draws nothing. Authored file content may not
// omit the asset; see [Sprite.Authored].
type Sprite struct {
	// Asset is the texture ID, or the sheet document ID when Region is set.
	Asset asset.ID `json:"asset,omitempty"`
	// Region is the sheet region to draw. An empty value selects the whole
	// texture named by Asset.
	Region string `json:"region,omitempty"`
	// Style holds the visual style.
	Style Style `json:"style,omitempty"`
}

// Authored reports an error when the sprite has no asset. File content must
// name one; runtime zero values stay legal.
func (s Sprite) Authored() error {
	if s.Asset == "" {
		return fmt.Errorf("authored sprite requires an asset")
	}
	return nil
}

// Validate reports whether the sprite's source and style are valid. An empty
// Asset draws nothing; a set Region requires a sheet-valid Asset.
func (s Sprite) Validate() error {
	if err := s.Style.Validate(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	if s.Asset == "" {
		if s.Region != "" {
			// A region names something in a sheet; without an asset it is
			// configuration nonsense, not a drawable.
			return fmt.Errorf("region %q without an asset", s.Region)
		}
		// Style without a picture: legal, not drawn.
		return nil
	}
	if s.Region == "" {
		return nil
	}
	if err := asset.FileSheet(s.Asset).Validate(); err != nil {
		return fmt.Errorf("sheet source: %w", err)
	}
	return nil
}

// Text draws one line of text centered on the entity's position, using the
// font named by Asset. The entity's transform scale applies to TextSize.
//
// The zero value is valid and draws nothing.
type Text struct {
	// Asset is the ID of the font asset to render with.
	Asset asset.ID `json:"asset,omitempty"`
	// Text is the single line to draw. Empty text draws nothing.
	Text string `json:"text,omitempty"`
	// TextSize is the font size in pixels before scaling.
	TextSize float64 `json:"textSize,omitempty"`
	// Style holds the visual style.
	Style Style `json:"style,omitempty"`
}

// Authored reports whether the text is acceptable as authored file
// content: file text requires a font. Runtime zero values stay legal.
func (t Text) Authored() error {
	if t.Asset == "" {
		return fmt.Errorf("authored text requires a font")
	}
	return nil
}

// Validate reports whether the text component and its style are valid. An
// empty Asset draws nothing; a set one requires a positive size and a
// single line.
func (t Text) Validate() error {
	if err := t.Style.Validate(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	if t.Asset == "" {
		// No font: nothing to draw.
		return nil
	}
	if t.TextSize <= 0 || math.IsNaN(t.TextSize) || math.IsInf(t.TextSize, 0) {
		return fmt.Errorf("text size must be a positive number of pixels")
	}
	if strings.ContainsAny(t.Text, "\r\n") {
		return fmt.Errorf("text is single-line; use one Text component per line")
	}
	return nil
}

// Shape selects the geometry a [Primitive] draws. Its zero value is
// ShapeNone. The wire identities below are contracts: renaming one is a
// breaking change.
type Shape uint8

const (
	// ShapeNone draws nothing. It is the zero value.
	ShapeNone Shape = iota
	// ShapeRect draws a rectangle of Rect size centered on the position.
	ShapeRect
	// ShapeCircle draws a circle with Radii centered on the position, or an
	// ellipse when the radii differ.
	ShapeCircle
	// ShapeLine draws a segment with From and To endpoints relative to the
	// position.
	ShapeLine
)

// String returns the wire identity of the shape.
func (s Shape) String() string {
	switch s {
	case ShapeNone:
		return "none"
	case ShapeRect:
		return "rect"
	case ShapeCircle:
		return "circle"
	case ShapeLine:
		return "line"
	default:
		return fmt.Sprintf("Shape(%d)", uint8(s))
	}
}

// MarshalJSON writes the shape as its wire identity.
func (s Shape) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON reads a shape from its wire identity. It returns an error
// for an unknown identity.
func (s *Shape) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	switch name {
	case "none":
		*s = ShapeNone
	case "rect":
		*s = ShapeRect
	case "circle":
		*s = ShapeCircle
	case "line":
		*s = ShapeLine
	default:
		return fmt.Errorf("unknown primitive shape %q", name)
	}
	return nil
}

// Primitive draws geometry: a rectangle, a circle, or a line segment, with
// fill or outline from Style.
//
// The zero value is valid and draws nothing. Authoring a shapeless
// primitive in file content is a mistake; see [Primitive.Authored].
type Primitive struct {
	// Shape selects the geometry.
	Shape Shape `json:"shape"`
	// Rect is the rectangle's width and height in world units (ShapeRect).
	Rect geom.Vector2 `json:"rect,omitempty"`
	// Radii are the circle's horizontal and vertical radii in world units
	// (ShapeCircle).
	Radii geom.Vector2 `json:"radii,omitempty"`
	// From is the segment's start relative to the position (ShapeLine).
	From geom.Vector2 `json:"from,omitempty"`
	// To is the segment's end relative to the position (ShapeLine).
	To geom.Vector2 `json:"to,omitempty"`
	// Style holds the visual style.
	Style Style `json:"style,omitempty"`
}

// Authored reports an error when the primitive has no shape. File content
// must name one; runtime zero values stay legal.
func (p Primitive) Authored() error {
	if p.Shape == ShapeNone {
		return fmt.Errorf("authored primitive requires a shape")
	}
	return nil
}

// Validate reports whether the primitive and its style are valid.
func (p Primitive) Validate() error {
	if err := p.Style.Validate(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	switch p.Shape {
	case ShapeNone:
		return nil
	case ShapeRect:
		if !isFiniteVector(p.Rect) {
			return fmt.Errorf("rect size must be finite")
		}
		if p.Rect.X <= 0 || p.Rect.Y <= 0 {
			return fmt.Errorf("rect size must be positive")
		}
	case ShapeCircle:
		if !isFiniteVector(p.Radii) {
			return fmt.Errorf("circle radii must be finite")
		}
		if p.Radii.X <= 0 || p.Radii.Y <= 0 {
			return fmt.Errorf("circle radii must be positive")
		}
	case ShapeLine:
		if !isFiniteVector(p.From) || !isFiniteVector(p.To) {
			return fmt.Errorf("line endpoints must be finite")
		}
		if p.From == p.To {
			return fmt.Errorf("line endpoints must be distinct")
		}
	default:
		return fmt.Errorf("unknown primitive shape %d", uint8(p.Shape))
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isFiniteVector(v geom.Vector2) bool {
	return isFinite(v.X) && isFinite(v.Y)
}

// DrawShape is a geometry value used by the render pipeline's dispatch list.
// Implementations are [RectShape], [CircleShape], and [LineShape]. Shapes
// are not components: components spell geometry as Primitive data, and the
// collector builds these values when it resolves a frame.
type DrawShape interface {
	isShape()
}

// RectShape is a rectangle centered on the drawn position. Its fill or
// outline comes from the item's style.
type RectShape struct {
	// Size is the rectangle's width and height in world units.
	Size geom.Vector2
}

// CircleShape is a circle centered on the drawn position, or an ellipse
// when its radii differ. Its fill or outline comes from the item's style.
type CircleShape struct {
	// Radii are the circle's horizontal and vertical radii in world units.
	Radii geom.Vector2
}

// LineShape is a segment whose endpoints are relative to the drawn
// position.
type LineShape struct {
	// From is the segment's start point relative to the drawn position.
	From geom.Vector2
	// To is the segment's end point relative to the drawn position.
	To geom.Vector2
}

func (RectShape) isShape()   {}
func (CircleShape) isShape() {}
func (LineShape) isShape()   {}
