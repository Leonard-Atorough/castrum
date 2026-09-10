package geom

import "testing"

func TestPolygon_PointManagement(t *testing.T) {
	polygon := Polygon{}
	points := []Vector2{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 4, Y: 3},
	}

	for _, point := range points {
		polygon.AddPoint(point)
	}

	if got := polygon.NumPoints(); got != len(points) {
		t.Fatalf("NumPoints() = %v, want %v", got, len(points))
	}
	for index, want := range points {
		if got := polygon.GetPoint(index); got != want {
			t.Fatalf("GetPoint(%v) = %v, want %v", index, got, want)
		}
	}

	polygon.RemovePoint(1)
	wantPoints := []Vector2{points[0], points[2]}
	if got := polygon.GetPoints(); len(got) != len(wantPoints) {
		t.Fatalf("GetPoints() after RemovePoint() returned %v points, want %v", len(got), len(wantPoints))
	}
	for index, want := range wantPoints {
		if got := polygon.GetPoint(index); got != want {
			t.Fatalf("GetPoint(%v) after RemovePoint() = %v, want %v", index, got, want)
		}
	}

	polygon.RemovePoint(-1)
	polygon.RemovePoint(polygon.NumPoints())
	if polygon.NumPoints() != len(wantPoints) {
		t.Fatalf("invalid RemovePoint() changed polygon, got %v points", polygon.NumPoints())
	}

	polygon.Clear()
	if polygon.NumPoints() != 0 {
		t.Fatalf("NumPoints() after Clear() = %v, want 0", polygon.NumPoints())
	}
}

func TestPolygon_GetPoint_InvalidIndex(t *testing.T) {
	polygon := Polygon{Points: []Vector2{{X: 1, Y: 2}}}

	for _, index := range []int{-1, 1} {
		if got := polygon.GetPoint(index); got != (Vector2{}) {
			t.Fatalf("GetPoint(%v) = %v, want zero vector", index, got)
		}
	}
}

func TestPolygon_GetEdges(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 4, Y: 3},
	}}
	want := [][2]Vector2{
		{{X: 0, Y: 0}, {X: 4, Y: 0}},
		{{X: 4, Y: 0}, {X: 4, Y: 3}},
		{{X: 4, Y: 3}, {X: 0, Y: 0}},
	}

	if got := polygon.GetEdges(); len(got) != len(want) {
		t.Fatalf("GetEdges() returned %v edges, want %v", len(got), len(want))
	} else {
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("GetEdges()[%v] = %v, want %v", index, got[index], want[index])
			}
		}
	}

	if got := (&Polygon{}).GetEdges(); len(got) != 0 {
		t.Fatalf("empty polygon GetEdges() returned %v edges, want 0", len(got))
	}
}

func TestPolygon_Perimeter(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 4, Y: 3},
		{X: 0, Y: 3},
	}}

	if got := polygon.Perimeter(); !almostEqual(got, 14) {
		t.Fatalf("Perimeter() = %v, want 14", got)
	}
	if got := (&Polygon{}).Perimeter(); got != 0 {
		t.Fatalf("empty polygon Perimeter() = %v, want 0", got)
	}
}

func TestPolygon_Area(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 0, Y: 0},
		{X: 4, Y: 0},
		{X: 4, Y: 3},
		{X: 0, Y: 3},
	}}

	if got := polygon.Area(); !almostEqual(got, 12) {
		t.Fatalf("Area() = %v, want 12", got)
	}

	reversed := Polygon{Points: []Vector2{
		{X: 0, Y: 3},
		{X: 4, Y: 3},
		{X: 4, Y: 0},
		{X: 0, Y: 0},
	}}
	if got := reversed.Area(); !almostEqual(got, 12) {
		t.Fatalf("Area() for reversed polygon = %v, want 12", got)
	}

	for _, points := range [][]Vector2{nil, {{X: 0, Y: 0}}, {{X: 0, Y: 0}, {X: 1, Y: 1}}} {
		if got := (&Polygon{Points: points}).Area(); got != 0 {
			t.Fatalf("Area() for %v points = %v, want 0", len(points), got)
		}
	}
}

func TestPolygon_Contains(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 10, Y: 10},
		{X: 0, Y: 10},
	}}

	cases := []struct {
		name  string
		point Vector2
		want  bool
	}{
		{"inside", Vector2{X: 5, Y: 5}, true},
		{"outside", Vector2{X: 11, Y: 5}, false},
		{"on edge", Vector2{X: 0, Y: 5}, true},
		{"on vertex", Vector2{X: 0, Y: 0}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := polygon.Contains(tc.point); got != tc.want {
				t.Fatalf("Contains(%v) = %v, want %v", tc.point, got, tc.want)
			}
		})
	}

	if got := (&Polygon{}).Contains(Vector2{}); got {
		t.Fatal("empty polygon Contains() = true, want false")
	}
}

func TestPolygon_BoundingBox(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 3, Y: 8},
		{X: -2, Y: 4},
		{X: 5, Y: -1},
	}}
	want := Rect{Min: Vector2{X: -2, Y: -1}, Max: Vector2{X: 5, Y: 8}}

	if got := polygon.BoundingBox(); got != want {
		t.Fatalf("BoundingBox() = %v, want %v", got, want)
	}
	if got := (&Polygon{}).BoundingBox(); got != (Rect{}) {
		t.Fatalf("empty polygon BoundingBox() = %v, want zero rect", got)
	}
}

func TestPolygon_Centroid(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 0, Y: 0},
		{X: 6, Y: 0},
		{X: 6, Y: 4},
		{X: 0, Y: 4},
	}}

	if got := polygon.Centroid(); !vecAlmostEqual(got, Vector2{X: 3, Y: 2}) {
		t.Fatalf("Centroid() = %v, want {3 2}", got)
	}
	if got := (&Polygon{}).Centroid(); got != (Vector2{}) {
		t.Fatalf("empty polygon Centroid() = %v, want zero vector", got)
	}
}

func TestPolygon_Translate(t *testing.T) {
	polygon := Polygon{Points: []Vector2{
		{X: 1, Y: 2},
		{X: 3, Y: 4},
	}}
	polygon.Translate(Vector2{X: -1, Y: 5})

	want := []Vector2{{X: 0, Y: 7}, {X: 2, Y: 9}}
	for index, point := range want {
		if got := polygon.GetPoint(index); got != point {
			t.Fatalf("GetPoint(%v) after Translate() = %v, want %v", index, got, point)
		}
	}
}
