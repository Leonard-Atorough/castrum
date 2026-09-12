package geom

import (
	"errors"
	"math"
	"testing"
)

func rectanglePolygon(t *testing.T) Polygon {
	t.Helper()
	polygon, err := NewPolygon([]Vector2{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 4, Y: 3},
		{X: 0, Y: 3},
	})
	if err != nil {
		t.Fatalf("NewPolygon() error = %v", err)
	}
	return polygon
}

func TestNewPolygonCopiesAndNormalizesInput(t *testing.T) {
	points := []Vector2{{0, 0}, {0, 3}, {4, 3}, {4, 0}, {0, 0}}
	polygon, err := NewPolygon(points)
	if err != nil {
		t.Fatalf("NewPolygon() error = %v", err)
	}
	points[0] = Vector2{100, 100}

	if polygon.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", polygon.Len())
	}
	if !polygon.IsClockwise() {
		t.Error("NewPolygon() did not normalize winding to clockwise")
	}
	if got := polygon.Points(); got[0] == points[0] {
		t.Error("polygon retained caller point storage")
	}
	copyOfPoints := polygon.Points()
	copyOfPoints[0] = Vector2{200, 200}
	if got, _ := polygon.Point(0); got == copyOfPoints[0] {
		t.Error("Points() exposed polygon storage")
	}
}

func TestPolygonPointAccessAndImplicitClosure(t *testing.T) {
	polygon := rectanglePolygon(t)
	if got := polygon.Len(); got != 4 {
		t.Errorf("Len() = %d, want 4", got)
	}
	if _, ok := polygon.Point(-1); ok {
		t.Error("Point(-1) returned a point")
	}
	if _, ok := polygon.Point(polygon.Len()); ok {
		t.Error("Point(Len()) returned a point")
	}
	last, _ := polygon.Point(3)
	first, _ := polygon.Point(0)
	closing, ok := polygon.Edge(3)
	if !ok || closing.Start != last || closing.End != first {
		t.Errorf("closing edge = %v, want %v -> %v", closing, last, first)
	}
	if got := polygon.Edges(); len(got) != 4 {
		t.Errorf("Edges() length = %d, want 4", len(got))
	}
}

func TestPolygonValidation(t *testing.T) {
	tests := []struct {
		name   string
		points []Vector2
		err    error
	}{
		{"too few", []Vector2{{0, 0}, {1, 1}}, ErrPolygonTooFewPoints},
		{"duplicate", []Vector2{{0, 0}, {4, 0}, {4, 3}, {4, 0}}, ErrPolygonDuplicatePoint},
		{"collinear", []Vector2{{0, 0}, {2, 0}, {4, 0}, {4, 3}, {0, 3}}, ErrPolygonCollinearPoint},
		{"zero area", []Vector2{{0, 0}, {1, 0}, {2, 0}}, ErrPolygonCollinearPoint},
		{"non finite", []Vector2{{0, 0}, {1, 0}, {math.Inf(1), 1}}, ErrPolygonNonFinitePoint},
		{"self crossing", []Vector2{{0, 0}, {4, 4}, {0, 4}, {4, 0}}, ErrPolygonSelfIntersecting},
		{"non adjacent touch", []Vector2{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {2, -1}}, ErrPolygonSelfIntersecting},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewPolygon(test.points)
			if !errors.Is(err, test.err) {
				t.Fatalf("NewPolygon() error = %v, want %v", err, test.err)
			}
		})
	}

	concave, err := NewPolygon([]Vector2{{0, 0}, {4, 0}, {4, 4}, {2, 2}, {0, 4}})
	if err != nil {
		t.Fatalf("concave NewPolygon() error = %v", err)
	}
	if !concave.IsValid() || !concave.IsClockwise() {
		t.Error("valid concave polygon failed its invariants")
	}
}

func TestPolygonGeometry(t *testing.T) {
	polygon := rectanglePolygon(t)
	if got := polygon.Perimeter(); !almostEqual(got, 14) {
		t.Errorf("Perimeter() = %v, want 14", got)
	}
	if got := polygon.Area(); !almostEqual(got, 12) {
		t.Errorf("Area() = %v, want 12", got)
	}
	if got := polygon.SignedArea(); !almostEqual(got, -12) {
		t.Errorf("SignedArea() = %v, want -12", got)
	}
	if got := polygon.Centroid(); !vecAlmostEqual(got, Vector2{X: 2, Y: 1.5}) {
		t.Errorf("Centroid() = %v, want {2 1.5}", got)
	}
	if got := polygon.BoundingBox(); got != (Rect{Min: Vector2{0, 0}, Max: Vector2{4, 3}}) {
		t.Errorf("BoundingBox() = %v", got)
	}
}

func TestPolygonContainsBoundary(t *testing.T) {
	polygon := rectanglePolygon(t)
	for _, test := range []struct {
		point Vector2
		want  bool
	}{
		{Vector2{2, 1}, true},
		{Vector2{5, 1}, false},
		{Vector2{0, 1}, true},
		{Vector2{0, 0}, true},
	} {
		if got := polygon.Contains(test.point); got != test.want {
			t.Errorf("Contains(%v) = %v, want %v", test.point, got, test.want)
		}
	}
}

func TestPolygonValueOperationsDoNotMutateOriginal(t *testing.T) {
	polygon := rectanglePolygon(t)
	originalPoints := polygon.Points()
	translated := polygon.Translate(Vector2{X: 10, Y: -2})
	if got := polygon.Points(); !equalPoints(got, originalPoints) {
		t.Errorf("original points after Translate() = %v, want %v", got, originalPoints)
	}
	if got, _ := translated.Point(0); got != originalPoints[0].Add(Vector2{10, -2}) {
		t.Errorf("translated first point = %v", got)
	}

	if _, err := polygon.AddPoint(originalPoints[1]); err == nil {
		t.Error("AddPoint() accepted a duplicate point")
	}
	removed, err := polygon.RemovePoint(0)
	if err != nil || removed.Len() != 3 {
		t.Errorf("RemovePoint() = (%v, %v), want valid triangle", removed, err)
	}
	if polygon.Len() != 4 {
		t.Error("value operation changed original polygon")
	}
}

func equalPoints(first, second []Vector2) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func TestEmptyPolygonMethods(t *testing.T) {
	polygon := Polygon{}
	if polygon.IsValid() || polygon.Contains(Vector2{}) {
		t.Error("zero Polygon reported valid or contained a point")
	}
	if polygon.Perimeter() != 0 || polygon.Area() != 0 || polygon.BoundingBox() != (Rect{}) || polygon.Centroid() != (Vector2{}) {
		t.Error("zero Polygon returned non-zero geometry")
	}
}
