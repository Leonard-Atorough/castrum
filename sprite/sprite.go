// Package sprite defines the drawable component and its visual style.
//
// The component is plain data with JSON tags: Kind selects the source, and
// every field is a value or an asset ID. There are no interface fields, so
// the component serializes through the standard JSON codec without custom
// wire code. Components reference assets by ID and never hold runtime
// pointers; asset resolution happens in the render collector.
package sprite

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
)

// Kind selects which Source fields carry meaning. Its zero value is
// KindNone: a valid sprite that draws nothing. The JSON tag strings are
// wire identities: renaming one is a breaking change.
type Kind uint8

const (
	// KindNone draws nothing. It is the zero value.
	KindNone Kind = iota
	// KindTexture draws a whole standalone texture named by Asset.
	KindTexture
	// KindSheet draws the named Region of the sheet document named by Asset.
	KindSheet
	// KindText draws one line of text centered on the sprite's position,
	// using the font named by Asset with the sprite's scale applying to
	// TextSize.
	KindText
	// KindRect draws a rectangle of size Rect centered on the sprite's
	// position, filled or outlined by the sprite's style.
	KindRect
	// KindCircle draws a circle with radii Radii centered on the sprite's
	// position, or an ellipse when the radii differ.
	KindCircle
	// KindLine draws a segment with endpoints From and To relative to the
	// sprite's position.
	KindLine
)

// String returns the wire identity of the kind.
func (k Kind) String() string {
	switch k {
	case KindNone:
		return "none"
	case KindTexture:
		return "texture"
	case KindSheet:
		return "sheet"
	case KindText:
		return "text"
	case KindRect:
		return "rect"
	case KindCircle:
		return "circle"
	case KindLine:
		return "line"
	default:
		return fmt.Sprintf("Kind(%d)", uint8(k))
	}
}

// MarshalJSON writes the kind as its wire identity. UnmarshalJSON rejects
// unknown identities by name.
func (k Kind) MarshalJSON() ([]byte, error) {
	return []byte(`"` + k.String() + `"`), nil
}

func (k *Kind) UnmarshalJSON(data []byte) error {
	var name string
	if err := jsonUnmarshalString(data, &name); err != nil {
		return err
	}
	switch name {
	case "none":
		*k = KindNone
	case "texture":
		*k = KindTexture
	case "sheet":
		*k = KindSheet
	case "text":
		*k = KindText
	case "rect":
		*k = KindRect
	case "circle":
		*k = KindCircle
	case "line":
		*k = KindLine
	default:
		return fmt.Errorf("unknown sprite kind %q", name)
	}
	return nil
}

// Source is the sprite's drawable as data: one discriminating Kind and the
// fields it reads. Doc comments on each field name the kinds that use it;
// Validate enforces that grouping. The zero value is a valid KindNone
// source that draws nothing.
type Source struct {
	// Kind selects the source.
	Kind Kind `json:"kind"`
	// Asset is the texture ID (KindTexture), the sheet document ID
	// (KindSheet), or the font ID (KindText).
	Asset asset.ID `json:"asset,omitempty"`
	// Region is the sheet region to draw (KindSheet).
	Region string `json:"region,omitempty"`
	// Text is the single line to draw (KindText).
	Text string `json:"text,omitempty"`
	// TextSize is the font size in pixels before sprite scaling
	// (KindText).
	TextSize float64 `json:"textSize,omitempty"`
	// Rect is the rectangle's width and height in world units (KindRect).
	Rect geom.Vector2 `json:"rect,omitempty"`
	// Radii are the circle's horizontal and vertical radii in world
	// units (KindCircle).
	Radii geom.Vector2 `json:"radii,omitempty"`
	// To and From are the segment endpoints relative to the sprite's
	// position (KindLine).
	From geom.Vector2 `json:"from,omitempty"`
	To   geom.Vector2 `json:"to,omitempty"`
}

// Validate reports an error if the source is unusable for its kind: an
// unknown kind, a missing asset, a missing region, non-finite or
// non-positive geometry, a zero or non-finite text size, or multi-line
// text. KindNone is always valid.
func (src Source) Validate() error {
	switch src.Kind {
	case KindNone:
		return nil
	case KindTexture:
		if src.Asset == "" {
			return fmt.Errorf("texture source requires an asset")
		}
	case KindSheet:
		if err := asset.FileSheet(src.Asset).Validate(); err != nil {
			return fmt.Errorf("sheet source: %w", err)
		}
		if src.Region == "" {
			return fmt.Errorf("sheet source requires a region")
		}
	case KindText:
		if src.Asset == "" {
			return fmt.Errorf("text source requires a font")
		}
		if src.TextSize <= 0 || math.IsNaN(src.TextSize) || math.IsInf(src.TextSize, 0) {
			return fmt.Errorf("text size must be a positive number of pixels")
		}
		if strings.ContainsAny(src.Text, "\r\n") {
			return fmt.Errorf("text source is single-line; use one sprite per line")
		}
	case KindRect:
		if !isFiniteVector(src.Rect) {
			return fmt.Errorf("rect size must be finite")
		}
		if src.Rect.X <= 0 || src.Rect.Y <= 0 {
			return fmt.Errorf("rect size must be positive")
		}
	case KindCircle:
		if !isFiniteVector(src.Radii) {
			return fmt.Errorf("circle radii must be finite")
		}
		if src.Radii.X <= 0 || src.Radii.Y <= 0 {
			return fmt.Errorf("circle radii must be positive")
		}
	case KindLine:
		if !isFiniteVector(src.From) || !isFiniteVector(src.To) {
			return fmt.Errorf("line endpoints must be finite")
		}
		if src.From == src.To {
			return fmt.Errorf("line endpoints must be distinct")
		}
	default:
		return fmt.Errorf("unknown source kind %d", uint8(src.Kind))
	}
	return nil
}

// Style controls sprite ordering, color, outline, flips, and visibility.
// The zero value is a visible, plain sprite on layer 0.
type Style struct {
	// Outline strokes a shape's border instead of filling it. It requires a
	// positive StrokeWidth and has no effect on non-shape sources.
	Outline bool `json:"outline,omitempty"`
	// StrokeWidth is the outline width in world units. It must not be
	// negative and must be positive when Outline is set.
	StrokeWidth float64 `json:"strokeWidth,omitempty"`
	// Color tints texture pixels or sets the text or shape color. A nil
	// value leaves textures untinted, defaults text to white, and defaults
	// shapes to black; an RGBA with zero alpha is explicitly transparent.
	// Decoding always produces color.RGBA values for non-nil colors.
	Color *color.RGBA `json:"color,omitempty"`
	// Layer orders sprites from back to front, from 0 through 31.
	Layer uint8 `json:"layer,omitempty"`
	// SortOrder orders sprites within a layer; higher values draw on top.
	// Equal values are ordered by world Y.
	SortOrder int8 `json:"sortOrder,omitempty"`
	// Hidden excludes the sprite from collection.
	Hidden bool `json:"hidden,omitempty"`
	// FlipH and FlipV mirror texture-backed sources horizontally and
	// vertically; shapes are not flipped.
	FlipH bool `json:"flipH,omitempty"`
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

// Sprite is a drawable component with shared style and a source.
//
// The zero value is visible but draws nothing until Source is set. It is
// plain data with JSON tags; no field is an interface or a runtime
// pointer.
type Sprite struct {
	// Source is what the sprite draws.
	Source Source `json:"source"`
	// Style holds the sprite's visual style.
	Style Style `json:"style,omitempty"`
}

// Validate reports whether the sprite's source and style are valid. It
// does not resolve referenced assets.
func (s Sprite) Validate() error {
	if err := s.Style.Validate(); err != nil {
		return fmt.Errorf("style: %w", err)
	}
	if err := s.Source.Validate(); err != nil {
		return err
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isFiniteVector(v geom.Vector2) bool {
	return isFinite(v.X) && isFinite(v.Y)
}

// Shape is a geometry source used by the render pipeline's dispatch list.
// Implementations are [RectShape], [CircleShape], and [LineShape]. Shapes
// are not components: components spell shapes as Source data, and the
// collector builds these values when it resolves a frame.
type Shape interface {
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
