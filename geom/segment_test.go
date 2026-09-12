package geom

import "testing"

func TestSegmentLengthAndMidpoint(t *testing.T) {
	segment := Segment{Start: Vector2{X: 1, Y: 2}, End: Vector2{X: 4, Y: 6}}
	if got := segment.Length(); !almostEqual(got, 5) {
		t.Errorf("Length() = %v, want 5", got)
	}
	if got := segment.Midpoint(); got != (Vector2{X: 2.5, Y: 4}) {
		t.Errorf("Midpoint() = %v, want {2.5 4}", got)
	}
}

func TestSegmentClosestPointAndDistance(t *testing.T) {
	segment := Segment{Start: Vector2{X: 0, Y: 0}, End: Vector2{X: 4, Y: 0}}
	tests := []struct {
		name  string
		point Vector2
		want  Vector2
	}{
		{"before", Vector2{X: -2, Y: 3}, Vector2{X: 0, Y: 0}},
		{"on line", Vector2{X: 1, Y: 3}, Vector2{X: 1, Y: 0}},
		{"after", Vector2{X: 6, Y: -2}, Vector2{X: 4, Y: 0}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := segment.ClosestPoint(test.point); got != test.want {
				t.Errorf("ClosestPoint(%v) = %v, want %v", test.point, got, test.want)
			}
			if got := segment.DistanceToPoint(test.point); !almostEqual(got, test.point.Distance(test.want)) {
				t.Errorf("DistanceToPoint(%v) = %v, want %v", test.point, got, test.point.Distance(test.want))
			}
		})
	}

	point := Segment{Start: Vector2{X: 2, Y: 3}, End: Vector2{X: 2, Y: 3}}
	if got := point.ClosestPoint(Vector2{}); got != point.Start {
		t.Errorf("zero-length ClosestPoint() = %v, want %v", got, point.Start)
	}
}

func TestSegmentBoundingBox(t *testing.T) {
	segment := Segment{Start: Vector2{X: 3, Y: -1}, End: Vector2{X: -2, Y: 4}}
	want := Rect{Min: Vector2{X: -2, Y: -1}, Max: Vector2{X: 3, Y: 4}}
	if got := segment.BoundingBox(); got != want {
		t.Errorf("BoundingBox() = %v, want %v", got, want)
	}
}

func TestSegmentIntersects(t *testing.T) {
	segment := Segment{Start: Vector2{X: 0, Y: 0}, End: Vector2{X: 4, Y: 4}}
	tests := []struct {
		name       string
		other      Segment
		intersects bool
	}{
		{"crossing", Segment{Start: Vector2{X: 0, Y: 4}, End: Vector2{X: 4, Y: 0}}, true},
		{"endpoint touch", Segment{Start: Vector2{X: 4, Y: 4}, End: Vector2{X: 6, Y: 2}}, true},
		{"collinear overlap", Segment{Start: Vector2{X: 2, Y: 2}, End: Vector2{X: 6, Y: 6}}, true},
		{"disjoint", Segment{Start: Vector2{X: 5, Y: 0}, End: Vector2{X: 6, Y: 1}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := segment.Intersects(test.other); got != test.intersects {
				t.Errorf("Intersects(%v) = %v, want %v", test.other, got, test.intersects)
			}
		})
	}

	point := Segment{Start: Vector2{X: 1, Y: 1}, End: Vector2{X: 1, Y: 1}}
	if !point.Intersects(segment) {
		t.Error("zero-length segment at a segment point did not intersect")
	}
}
