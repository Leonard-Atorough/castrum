package geom

import "testing"

func TestCirclesIntersect(t *testing.T) {
	a := Circle{Center: Vector2{}, Radius: 5}

	for _, test := range []struct {
		name string
		b    Circle
		want bool
	}{
		{name: "overlap", b: Circle{Center: Vector2{X: 8}, Radius: 5}, want: true},
		{name: "tangent", b: Circle{Center: Vector2{X: 10}, Radius: 5}, want: true},
		{name: "separate", b: Circle{Center: Vector2{X: 11}, Radius: 5}, want: false},
		{name: "negative radius uses magnitude", b: Circle{Center: Vector2{X: 8}, Radius: -5}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := CirclesIntersect(a, test.b); got != test.want {
				t.Errorf("CirclesIntersect(%v, %v) = %v, want %v", a, test.b, got, test.want)
			}
		})
	}
}

func TestRectsIntersect(t *testing.T) {
	a := RectFromMinMax(Vector2{}, Vector2{X: 10, Y: 10})
	if !RectsIntersect(a, RectFromMinMax(Vector2{X: 5, Y: 5}, Vector2{X: 15, Y: 15})) {
		t.Error("RectsIntersect() returned false for overlapping rectangles")
	}
	if RectsIntersect(a, RectFromMinMax(Vector2{X: 10, Y: 0}, Vector2{X: 20, Y: 10})) {
		t.Error("RectsIntersect() returned true for edge-only contact")
	}
	if RectsIntersect(a, Rect{Min: Vector2{X: 10}, Max: Vector2{X: 0, Y: 10}}) {
		t.Error("RectsIntersect() returned true for invalid bounds")
	}
}

func TestCircleRectIntersects(t *testing.T) {
	rect := RectFromMinMax(Vector2{}, Vector2{X: 10, Y: 10})
	for _, test := range []struct {
		name   string
		circle Circle
		want   bool
	}{
		{name: "inside", circle: Circle{Center: Vector2{X: 5, Y: 5}, Radius: 1}, want: true},
		{name: "corner tangent", circle: Circle{Center: Vector2{X: -1, Y: -1}, Radius: 1.4142135623730951}, want: true},
		{name: "separate", circle: Circle{Center: Vector2{X: 20, Y: 5}, Radius: 1}, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := CircleRectIntersects(test.circle, rect); got != test.want {
				t.Errorf("CircleRectIntersects(%v, %v) = %v, want %v", test.circle, rect, got, test.want)
			}
		})
	}
}
