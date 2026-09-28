package geom

import (
	"math"
	"testing"
)

func TestCircle_Contains(t *testing.T) {
	c := Circle{Center: Vector2{}, Radius: 5}
	cases := []struct {
		name  string
		point Vector2
		want  bool
	}{
		{"center", Vector2{}, true},
		{"on edge", Vector2{X: 5, Y: 0}, true},
		{"outside", Vector2{X: 6, Y: 0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.Contains(tc.point); got != tc.want {
				t.Errorf("Contains(%v) = %v, want %v", tc.point, got, tc.want)
			}
		})
	}
	// A negative radius is sized, not inverted.
	mirrored := Circle{Center: Vector2{}, Radius: -5}
	if !mirrored.Contains(Vector2{X: 0, Y: 5}) {
		t.Error("negative radius should size the circle, not invert the test")
	}
}

func TestCircle_AreaAndCircumference(t *testing.T) {
	c := Circle{Radius: 2}
	if got, want := c.Area(), math.Pi*4; math.Abs(got-want) > 1e-9 {
		t.Errorf("Area() = %v, want %v", got, want)
	}
	if got, want := c.Circumference(), 2*math.Pi*2; math.Abs(got-want) > 1e-9 {
		t.Errorf("Circumference() = %v, want %v", got, want)
	}
}

func TestCircle_BoundingBox(t *testing.T) {
	c := Circle{Center: Vector2{X: 10, Y: 20}, Radius: 5}
	want := Rect{
		Min: Vector2{X: 5, Y: 15},
		Max: Vector2{X: 15, Y: 25},
	}
	if got := c.BoundingBox(); got != want {
		t.Errorf("BoundingBox() = %v, want %v", got, want)
	}
}
