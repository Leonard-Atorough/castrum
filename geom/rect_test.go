package geom

import (
	"testing"
)

func TestRect_Dimensions(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 5}}

	if got := r.Width(); got != 10 {
		t.Fatalf("Width() = %v, want 10", got)
	}
	if got := r.Height(); got != 5 {
		t.Fatalf("Height() = %v, want 5", got)
	}
	if got := r.Area(); got != 50 {
		t.Fatalf("Area() = %v, want 50", got)
	}
}

func TestRect_Contains(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 10}}

	cases := []struct {
		name  string
		point Vector2
		want  bool
	}{
		{"inside", Vector2{X: 5, Y: 5}, true},
		{"on boundary", Vector2{X: 10, Y: 10}, true},
		{"outside", Vector2{X: 11, Y: 5}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.Contains(tc.point); got != tc.want {
				t.Fatalf("Contains(%v) = %v, want %v", tc.point, got, tc.want)
			}
		})
	}
}

func TestRect_Intersects(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 10}}

	cases := []struct {
		name  string
		other Rect
		want  bool
	}{
		{"overlapping", Rect{Min: Vector2{X: 5, Y: 5}, Max: Vector2{X: 15, Y: 15}}, true},
		{"disjoint", Rect{Min: Vector2{X: 20, Y: 20}, Max: Vector2{X: 30, Y: 30}}, false},
		{"touching edge only", Rect{Min: Vector2{X: 10, Y: 0}, Max: Vector2{X: 20, Y: 10}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.Intersects(tc.other); got != tc.want {
				t.Fatalf("Intersects(%v) = %v, want %v", tc.other, got, tc.want)
			}
		})
	}
}

func TestRect_String(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 1, Y: 1}}
	if got := r.String(); got == "" {
		t.Fatal("String() should not be empty")
	}
}

func TestNewRect(t *testing.T) {
	cases := []struct {
		name string
		min, max Vector2
		want     Rect
	}{
		{"standard rect", Vector2{X: 0, Y: 0}, Vector2{X: 10, Y: 5}, Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 5}}},
		{"negative coords", Vector2{X: -5, Y: -3}, Vector2{X: 5, Y: 3}, Rect{Min: Vector2{X: -5, Y: -3}, Max: Vector2{X: 5, Y: 3}}},
		{"point rect", Vector2{X: 2, Y: 2}, Vector2{X: 2, Y: 2}, Rect{Min: Vector2{X: 2, Y: 2}, Max: Vector2{X: 2, Y: 2}}},
		{"float coords", Vector2{X: 1.5, Y: 2.5}, Vector2{X: 8.7, Y: 9.2}, Rect{Min: Vector2{X: 1.5, Y: 2.5}, Max: Vector2{X: 8.7, Y: 9.2}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewRect(tc.min, tc.max)
			if got != tc.want {
				t.Fatalf("NewRect(%v, %v) = %v, want %v", tc.min, tc.max, got, tc.want)
			}
		})
	}
}

func TestRect_BoundingBox(t *testing.T) {
	r := Rect{Min: Vector2{X: 1, Y: 2}, Max: Vector2{X: 10, Y: 15}}
	got := r.BoundingBox()
	if got != r {
		t.Fatalf("BoundingBox() = %v, want %v", got, r)
	}
}
