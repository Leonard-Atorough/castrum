package geom

import (
	"math"
	"testing"
)

func TestRect_ConstructorsCanonicalizeBounds(t *testing.T) {
	centered := RectFromCenterSize(Vector2{X: -200, Y: -200}, Vector2{X: -10, Y: 10})
	want := Rect{Min: Vector2{X: -205, Y: -205}, Max: Vector2{X: -195, Y: -195}}
	if centered != want {
		t.Errorf("RectFromCenterSize() = %v, want %v", centered, want)
	}

	fromCorners := RectFromMinMax(Vector2{X: 10, Y: -5}, Vector2{X: -10, Y: 5})
	want = Rect{Min: Vector2{X: -10, Y: -5}, Max: Vector2{X: 10, Y: 5}}
	if fromCorners != want {
		t.Errorf("RectFromMinMax() = %v, want %v", fromCorners, want)
	}
}

func TestRect_Dimensions(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 5}}

	if got := r.Width(); got != 10 {
		t.Errorf("Width() = %v, want 10", got)
	}
	if got := r.Height(); got != 5 {
		t.Errorf("Height() = %v, want 5", got)
	}
	if got := r.Area(); got != 50 {
		t.Errorf("Area() = %v, want 50", got)
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
				t.Errorf("Contains(%v) = %v, want %v", tc.point, got, tc.want)
			}
		})
	}
}

func TestRect_OverlapsOrTouches(t *testing.T) {
	r := Rect{Min: Vector2{}, Max: Vector2{X: 10, Y: 10}}
	for _, tc := range []struct {
		name  string
		other Rect
		want  bool
	}{
		{"overlap", Rect{Min: Vector2{X: 5, Y: 5}, Max: Vector2{X: 15, Y: 15}}, true},
		{"edge", Rect{Min: Vector2{X: 10}, Max: Vector2{X: 20, Y: 10}}, true},
		{"corner", Rect{Min: Vector2{X: 10, Y: 10}, Max: Vector2{X: 20, Y: 20}}, true},
		{"disjoint", Rect{Min: Vector2{X: 11}, Max: Vector2{X: 20, Y: 10}}, false},
		{"inverted", Rect{Min: Vector2{X: 5}, Max: Vector2{X: 1, Y: 10}}, false},
		{"non-finite", Rect{Max: Vector2{X: math.Inf(1), Y: 10}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.OverlapsOrTouches(tc.other); got != tc.want {
				t.Errorf("OverlapsOrTouches(%v) = %t, want %t", tc.other, got, tc.want)
			}
			if got := tc.other.OverlapsOrTouches(r); got != tc.want {
				t.Errorf("reverse OverlapsOrTouches = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestRect_ValidationAndNormalization(t *testing.T) {
	invalid := Rect{Min: Vector2{X: 10, Y: 10}, Max: Vector2{X: 0, Y: 0}}
	if invalid.IsValid() {
		t.Error("inverted rectangle IsValid() = true, want false")
	}
	if !invalid.IsEmpty() {
		t.Error("inverted rectangle IsEmpty() = false, want true")
	}
	if got := invalid.Area(); got != 0 {
		t.Errorf("inverted rectangle Area() = %v, want 0", got)
	}
	if got, want := invalid.Normalize(), (Rect{Min: Vector2{}, Max: Vector2{X: 10, Y: 10}}); got != want {
		t.Errorf("Normalize() = %v, want %v", got, want)
	}

	nonFinite := Rect{Min: Vector2{}, Max: Vector2{X: math.Inf(1), Y: 1}}
	if nonFinite.IsValid() {
		t.Error("non-finite rectangle IsValid() = true, want false")
	}
}

func TestRect_Union(t *testing.T) {
	a := Rect{Min: Vector2{}, Max: Vector2{X: 10, Y: 10}}
	b := Rect{Min: Vector2{X: 15, Y: 5}, Max: Vector2{X: 5, Y: -5}}
	want := Rect{Min: Vector2{Y: -5}, Max: Vector2{X: 15, Y: 10}}
	if got := a.Union(b); got != want {
		t.Errorf("Union() = %v, want %v", got, want)
	}
	if got := b.Union(a); got != want {
		t.Errorf("reverse Union() = %v, want %v", got, want)
	}
}

func TestRect_ExpandClosestPointAndDistance(t *testing.T) {
	r := RectFromCenterSize(Vector2{X: -200, Y: -200}, Vector2{X: 10, Y: 10})
	if got, want := r.Expand(5), (Rect{Min: Vector2{X: -210, Y: -210}, Max: Vector2{X: -190, Y: -190}}); got != want {
		t.Errorf("Expand() = %v, want %v", got, want)
	}
	if got, want := r.Expand(-10), (Rect{Min: Vector2{X: -200, Y: -200}, Max: Vector2{X: -200, Y: -200}}); got != want {
		t.Errorf("Expand() with negative amount = %v, want %v", got, want)
	}

	point := Vector2{X: -190, Y: -210}
	if got, want := r.ClosestPoint(point), (Vector2{X: -195, Y: -205}); got != want {
		t.Errorf("ClosestPoint() = %v, want %v", got, want)
	}
	if got, want := r.DistanceSquared(point), 50.0; got != want {
		t.Errorf("DistanceSquared() = %v, want %v", got, want)
	}
}

func TestRect_String(t *testing.T) {
	r := Rect{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 1, Y: 1}}
	if got := r.String(); got == "" {
		t.Error("String() should not be empty")
	}
}
