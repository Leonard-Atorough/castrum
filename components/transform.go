package components

import (
	"fmt"

	"github.com/leonard-atorough/castrum/geom"
)

// Transform represents the spatial state of an entity: position, rotation,
// and scale. Scale is always a multiplier, not a pixel size — {1,1} means
// no scaling. For primitives, the base size comes from Sprite.Size; for
// textured sprites, it comes from the image or atlas region dimensions.
type Transform struct {
	Position geom.Vector2
	Rotation float64
	Scale    geom.Vector2
	Origin   geom.Vector2
}

// NewTransform creates a Transform component with the specified position,
// rotation, scale, and origin.
func NewTransform(position geom.Vector2, rotation float64, scale geom.Vector2, origin geom.Vector2) Transform {
	return Transform{
		Position: position,
		Rotation: rotation,
		Scale:    scale,
		Origin:   origin,
	}
}

func (t Transform) Serialize() (map[string]any, error) {
	return map[string]any{
		"position": map[string]any{"x": t.Position.X, "y": t.Position.Y},
		"rotation": t.Rotation,
		"scale":    map[string]any{"x": t.Scale.X, "y": t.Scale.Y},
		"origin":   map[string]any{"x": t.Origin.X, "y": t.Origin.Y},
	}, nil
}

func (t Transform) Deserialize(data map[string]any) (Transform, error) {
	if position, ok := data["position"].(map[string]any); ok {
		if x, ok := position["x"].(float64); ok {
			t.Position.X = x
		}
		if y, ok := position["y"].(float64); ok {
			t.Position.Y = y
		}
	}
	if rotation, ok := data["rotation"].(float64); ok {
		t.Rotation = rotation
	}
	if scale, ok := data["scale"].(map[string]any); ok {
		if x, ok := scale["x"].(float64); ok {
			t.Scale.X = x
		}
		if y, ok := scale["y"].(float64); ok {
			t.Scale.Y = y
		}
	}
	if origin, ok := data["origin"].(map[string]any); ok {
		if x, ok := origin["x"].(float64); ok {
			t.Origin.X = x
		}
		if y, ok := origin["y"].(float64); ok {
			t.Origin.Y = y
		}
	}
	return t, t.Validate()
}

func (t Transform) Validate() error {
	if t.Scale.X == 0 || t.Scale.Y == 0 {
		return fmt.Errorf("scale components must be non-zero")
	}
	return nil
}
