package geom

import (
	"math"
	"testing"
)

func TestCircle_Contains(t *testing.T) {
	c := Circle{Center: Vector2{X: 0, Y: 0}, Radius: 5}

	cases := []struct {
		name  string
		point Vector2
		want  bool
	}{
		{"center", Vector2{X: 0, Y: 0}, true},
		{"on edge", Vector2{X: 5, Y: 0}, true},
		{"outside", Vector2{X: 6, Y: 0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.Contains(tc.point); got != tc.want {
				t.Fatalf("Contains(%v) = %v, want %v", tc.point, got, tc.want)
			}
		})
	}
}

func TestCircle_Intersects(t *testing.T) {
	c := Circle{Center: Vector2{X: 0, Y: 0}, Radius: 5}

	cases := []struct {
		name  string
		other Circle
		want  bool
	}{
		{"overlapping", Circle{Center: Vector2{X: 8, Y: 0}, Radius: 5}, true},
		{"disjoint", Circle{Center: Vector2{X: 20, Y: 0}, Radius: 5}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.Intersects(tc.other); got != tc.want {
				t.Fatalf("Intersects(%v) = %v, want %v", tc.other, got, tc.want)
			}
		})
	}
}

func TestCircle_AreaAndCircumference(t *testing.T) {
	c := Circle{Radius: 2}

	if got, want := c.Area(), math.Pi*4; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Area() = %v, want %v", got, want)
	}
	if got, want := c.Circumference(), 2*math.Pi*2; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Circumference() = %v, want %v", got, want)
	}
}
