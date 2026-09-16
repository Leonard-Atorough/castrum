package components

import (
	"fmt"
	"image/color"

	"github.com/leonard-atorough/castrum/geom"
)

// PrimitiveType represents the type of a procedural shape for rendering.
type PrimitiveType uint8

const (
	PrimitiveKindRectangle PrimitiveType = iota
	PrimitiveKindCircle
	PrimitiveKindLine
	PrimitiveKindPolygon
)

// Sprite is the renderable component. It carries either a texture reference
// or a primitive shape (Primitive).
//
// For textured sprites, set TexturePath for a standalone texture, or set
// AtlasID and RegionName for an atlas region. TexturePath is optional when
// AtlasID is set — the renderer resolves the texture path from the atlas.
//
// RegionName is the initial/fallback region used for static sprites. When
// the entity also has an [Animation] component, the animation system
// advances the frame index each tick and the renderer uses the animation's
// current frame instead of RegionName.
//
// For primitive sprites, Size is the base dimension in pixels and
// Transform.Scale is a multiplier on top of it. A 32x32 rectangle with
// Scale{1,1} renders at 32x32; with Scale{2,2} it renders at 64x64.
//
// FlipH and FlipV mirror the sprite along its horizontal and vertical axes.
// They apply to both textured and primitive sprites and do not affect
// physics or collision — they are purely visual.
//
// Opacity controls overall transparency independently of Color. A value of
// 1.0 is fully opaque; 0.0 is fully transparent. The renderer multiplies the
// Color alpha by Opacity at draw time.
//
// Polygon holds the vertex data when Primitive is PrimitiveKindPolygon. It
// is ignored for all other primitive types and textured sprites.
type Sprite struct {
	TexturePath string
	AtlasID     string
	RegionName  string
	Primitive   PrimitiveType
	Color       color.Color
	Size        geom.Vector2 // base dimensions in pixels for primitives; ignored by textured sprites
	RenderLayer uint8        // which of 32 layers to render on (0-31)
	SortOrder   int8         // [-128..127], higher values render on top within the layer
	Visible     bool
	FlipH       bool         // mirror horizontally
	FlipV       bool         // mirror vertically
	Opacity     float32      // 0.0 (transparent) to 1.0 (opaque); multiplies Color alpha
	Polygon     geom.Polygon // vertex data for PrimitiveKindPolygon; zero value for other types
}

// NewSprite creates a new Sprite component with the specified properties.
func NewSprite(texturePath string, atlasID string, regionName string, primitive PrimitiveType, color color.Color, size geom.Vector2, renderLayer uint8, sortOrder int8, visible bool, flipH bool, flipV bool, opacity float32, polygon geom.Polygon) (Sprite, error) {
	if renderLayer > 31 {
		renderLayer = 31
	}
	if primitive < PrimitiveKindRectangle || primitive > PrimitiveKindPolygon {
		primitive = PrimitiveKindRectangle
	}
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	if atlasID != "" && regionName == "" || atlasID == "" && regionName != "" {
		return Sprite{}, fmt.Errorf("atlasID and regionName must be specified together")
	}
	sprite := Sprite{
		TexturePath: texturePath,
		AtlasID:     atlasID,
		RegionName:  regionName,
		Primitive:   primitive,
		Color:       color,
		Size:        size,
		RenderLayer: renderLayer,
		SortOrder:   sortOrder,
		Visible:     visible,
		FlipH:       flipH,
		FlipV:       flipV,
		Opacity:     opacity,
		Polygon:     polygon,
	}
	if err := sprite.Validate(); err != nil {
		return Sprite{}, err
	}
	return sprite, nil
}

func (s Sprite) Validate() error {
	if s.AtlasID != "" && s.RegionName == "" || s.AtlasID == "" && s.RegionName != "" {
		return fmt.Errorf("atlasID and regionName must be specified together")
	}
	if s.RenderLayer > 31 {
		return fmt.Errorf("renderLayer must be between 0 and 31")
	}
	if s.Primitive < PrimitiveKindRectangle || s.Primitive > PrimitiveKindPolygon {
		return fmt.Errorf("invalid primitive type")
	}
	if s.Opacity < 0 || s.Opacity > 1 {
		return fmt.Errorf("opacity must be between 0.0 and 1.0")
	}
	return nil
}

func (s Sprite) Serialize() (map[string]any, error) {
	data := map[string]any{
		"texturePath": s.TexturePath,
		"atlasID":     s.AtlasID,
		"regionName":  s.RegionName,
		"primitive":   uint8(s.Primitive),
		"size": map[string]float64{
			"x": s.Size.X,
			"y": s.Size.Y,
		},
		"renderLayer": s.RenderLayer,
		"sortOrder":   s.SortOrder,
		"visible":     s.Visible,
		"flipH":       s.FlipH,
		"flipV":       s.FlipV,
		"opacity":     s.Opacity,
	}

	if s.Color != nil {
		r, g, b, a := s.Color.RGBA()
		data["color"] = map[string]uint32{
			"r": uint32(r),
			"g": uint32(g),
			"b": uint32(b),
			"a": uint32(a),
		}
	}

	if s.Primitive == PrimitiveKindPolygon && s.Polygon.Len() > 0 {
		points := s.Polygon.Points()
		serializedPoints := make([]map[string]float64, len(points))
		for i, p := range points {
			serializedPoints[i] = map[string]float64{"x": p.X, "y": p.Y}
		}
		data["polygon"] = serializedPoints
	}

	return data, nil
}

func (s Sprite) Deserialize(data map[string]any) (Sprite, error) {
	if v, ok := data["texturePath"].(string); ok {
		s.TexturePath = v
	}
	if v, ok := data["atlasID"].(string); ok {
		s.AtlasID = v
	}
	if v, ok := data["regionName"].(string); ok {
		s.RegionName = v
	}
	if v, ok := data["primitive"].(float64); ok {
		s.Primitive = PrimitiveType(uint8(v))
	}
	if size, ok := data["size"].(map[string]any); ok {
		if x, ok := size["x"].(float64); ok {
			s.Size.X = x
		}
		if y, ok := size["y"].(float64); ok {
			s.Size.Y = y
		}
	}
	if v, ok := data["renderLayer"]; ok {
		switch val := v.(type) {
		case float64:
			s.RenderLayer = uint8(val)
		case uint8:
			s.RenderLayer = val
		}
	}
	if v, ok := data["sortOrder"]; ok {
		switch val := v.(type) {
		case float64:
			s.SortOrder = int8(val)
		case int8:
			s.SortOrder = val
		}
	}
	if v, ok := data["visible"].(bool); ok {
		s.Visible = v
	}
	if v, ok := data["flipH"].(bool); ok {
		s.FlipH = v
	}
	if v, ok := data["flipV"].(bool); ok {
		s.FlipV = v
	}
	if v, ok := data["opacity"].(float64); ok {
		s.Opacity = float32(v)
	}
	if colorData, ok := data["color"].(map[string]any); ok {
		r, _ := colorData["r"].(float64)
		g, _ := colorData["g"].(float64)
		b, _ := colorData["b"].(float64)
		a, _ := colorData["a"].(float64)
		s.Color = color.RGBA{
			R: uint8(r / 0x101),
			G: uint8(g / 0x101),
			B: uint8(b / 0x101),
			A: uint8(a / 0x101),
		}
	}
	if points, ok := data["polygon"].([]any); ok {
		verts := make([]geom.Vector2, 0, len(points))
		for _, raw := range points {
			if pt, ok := raw.(map[string]any); ok {
				x, _ := pt["x"].(float64)
				y, _ := pt["y"].(float64)
				verts = append(verts, geom.Vector2{X: x, Y: y})
			}
		}
		if len(verts) > 0 {
			polygon, err := geom.NewPolygon(verts)
			if err != nil {
				return s, fmt.Errorf("failed to deserialize polygon: %w", err)
			}
			s.Polygon = polygon
		}
	}

	return s, s.Validate()
}
