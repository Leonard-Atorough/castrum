package components

import (
	"image/color"
	"testing"

	"github.com/leonard-atorough/castrum/geom"
)

func TestTransformComponent(t *testing.T) {
	t.Run("Create New Transform Component with no color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, nil)
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
	})

	t.Run("Create New Transform Component with color", func(t *testing.T) {
		transform := NewTransform(geom.Vector2{X: 0, Y: 0}, 0, geom.Vector2{X: 1, Y: 1}, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Color != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
			t.Errorf("Expected color to be red, got %v", transform.Color)
		}
	})

	t.Run("Create New Transform Component with default values", func(t *testing.T) {
		transform := NewTransformWithDefault()
		if transform.Position.X != 0 || transform.Position.Y != 0 {
			t.Errorf("Expected position to be (0,0), got (%v,%v)", transform.Position.X, transform.Position.Y)
		}
		if transform.Rotation != 0 {
			t.Errorf("Expected rotation to be 0, got %v", transform.Rotation)
		}
		if transform.Scale.X != 1 || transform.Scale.Y != 1 {
			t.Errorf("Expected scale to be (1,1), got (%v,%v)", transform.Scale.X, transform.Scale.Y)
		}
		if transform.Color != color.Transparent {
			t.Errorf("Expected color to be transparent, got %v", transform.Color)
		}
	})
}
