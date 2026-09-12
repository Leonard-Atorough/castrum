package geom

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidPolygon          = errors.New("invalid polygon")
	ErrPolygonTooFewPoints     = fmt.Errorf("%w: at least 3 points are required", ErrInvalidPolygon)
	ErrPolygonNonFinitePoint   = fmt.Errorf("%w: points must be finite", ErrInvalidPolygon)
	ErrPolygonDuplicatePoint   = fmt.Errorf("%w: points must be unique", ErrInvalidPolygon)
	ErrPolygonCollinearPoint   = fmt.Errorf("%w: consecutive points must not be collinear", ErrInvalidPolygon)
	ErrPolygonZeroArea         = fmt.Errorf("%w: area must be non-zero", ErrInvalidPolygon)
	ErrPolygonSelfIntersecting = fmt.Errorf("%w: edges must not intersect", ErrInvalidPolygon)
)

// Polygon is an immutable, simple polygon with implicit closure.
//
// Points are stored without a duplicate closing point. The edge from the last
// point to the first is included by Edge, Edges, Area, and the other geometry
// methods. NewPolygon copies its input and stores points in clockwise order.
type Polygon struct {
	points []Vector2
}

// NewPolygon creates a validated polygon from points.
//
// A repeated final point is treated as an explicit closing point and removed.
// Counter-clockwise input is reversed so every returned polygon is clockwise.
func NewPolygon(points []Vector2) (Polygon, error) {
	points = append([]Vector2(nil), points...)
	if len(points) > 1 && points[0] == points[len(points)-1] {
		points = points[:len(points)-1]
	}
	polygon := Polygon{points: points}
	if err := polygon.Validate(); err != nil {
		return Polygon{}, err
	}
	if polygon.SignedArea() > 0 {
		reversePoints(points)
	}
	return Polygon{points: points}, nil
}

// Len returns the number of stored vertices, excluding the implicit closing point.
func (p Polygon) Len() int {
	return len(p.points)
}

// Point returns the vertex at index, or false when index is out of range.
func (p Polygon) Point(index int) (Vector2, bool) {
	if index < 0 || index >= len(p.points) {
		return Vector2{}, false
	}
	return p.points[index], true
}

// Points returns a defensive copy of the polygon vertices.
func (p Polygon) Points() []Vector2 {
	return append([]Vector2(nil), p.points...)
}

// Edge returns the edge at index, including the implicit last-to-first edge.
func (p Polygon) Edge(index int) (Segment, bool) {
	if index < 0 || index >= len(p.points) {
		return Segment{}, false
	}
	next := (index + 1) % len(p.points)
	return Segment{Start: p.points[index], End: p.points[next]}, true
}

// Edges returns all polygon edges, including the implicit closing edge.
func (p Polygon) Edges() []Segment {
	edges := make([]Segment, 0, len(p.points))
	for i := 0; i < len(p.points); i++ {
		next := (i + 1) % len(p.points)
		edges = append(edges, Segment{Start: p.points[i], End: p.points[next]})
	}
	return edges
}

// Perimeter calculates the perimeter by summing the lengths of all edges.
func (p Polygon) Perimeter() float64 {
	perimeter := 0.0
	for index := range p.points {
		edge, _ := p.Edge(index)
		dx := edge.End.X - edge.Start.X
		dy := edge.End.Y - edge.Start.Y
		perimeter += math.Sqrt(dx*dx + dy*dy)
	}
	return perimeter
}

// Area calculates the non-negative area using the shoelace formula.
func (p Polygon) Area() float64 {
	area := 0.0
	n := len(p.points)
	for i := range n {
		j := (i + 1) % n
		area += p.points[i].X * p.points[j].Y
		area -= p.points[j].X * p.points[i].Y
	}
	return math.Abs(area) / 2.0
}

// SignedArea calculates the signed area of the polygon.
// The sign of the area indicates the orientation of the vertices:
// positive if the vertices are ordered counter-clockwise, negative if clockwise.
func (p Polygon) SignedArea() float64 {
	area := 0.0
	n := len(p.points)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += p.points[i].X * p.points[j].Y
		area -= p.points[j].X * p.points[i].Y
	}
	return area / 2.0
}

// IsClockwise reports whether the vertices have clockwise winding.
func (p Polygon) IsClockwise() bool {
	return p.SignedArea() < 0
}

// Contains determines if a given point is inside the polygon using the ray-casting algorithm.
// It returns true if the point is inside the polygon, and false otherwise.
func (p Polygon) Contains(point Vector2) bool {
	n := len(p.points)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := range n {
		pi := p.points[i]
		pj := p.points[j]
		if pointOnSegment(point, pj, pi) {
			return true
		}
		if ((pi.Y > point.Y) != (pj.Y > point.Y)) &&
			(point.X < (pj.X-pi.X)*(point.Y-pi.Y)/(pj.Y-pi.Y)+pi.X) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// BoundingBox calculates the axis-aligned bounding box of the polygon.
func (p Polygon) BoundingBox() Rect {
	if len(p.points) == 0 {
		return Rect{}
	}
	min := p.points[0]
	max := p.points[0]
	for _, point := range p.points[1:] {
		if point.X < min.X {
			min.X = point.X
		}
		if point.Y < min.Y {
			min.Y = point.Y
		}
		if point.X > max.X {
			max.X = point.X
		}
		if point.Y > max.Y {
			max.Y = point.Y
		}
	}
	return Rect{Min: min, Max: max}
}

// Centroid calculates the centroid (geometric center) of the polygon.
// It returns a zero vector if the polygon has no points.
func (p Polygon) Centroid() Vector2 {
	n := len(p.points)
	if n == 0 {
		return Vector2{}
	}
	signedArea := p.SignedArea()
	if signedArea == 0 {
		return Vector2{}
	}
	centroid := Vector2{}
	for index, point := range p.points {
		next := p.points[(index+1)%n]
		cross := point.X*next.Y - next.X*point.Y
		centroid.X += (point.X + next.X) * cross
		centroid.Y += (point.Y + next.Y) * cross
	}
	return centroid.Mul(1 / (6 * signedArea))
}

// Translate returns a translated copy of p. The original polygon is unchanged.
func (p Polygon) Translate(offset Vector2) Polygon {
	translated := make([]Vector2, len(p.points))
	for index, point := range p.points {
		translated[index] = point.Add(offset)
	}
	return Polygon{points: translated}
}

// AddPoint returns a validated polygon with point appended to p.
func (p Polygon) AddPoint(point Vector2) (Polygon, error) {
	points := append(p.Points(), point)
	return NewPolygon(points)
}

// RemovePoint returns a validated polygon with the indexed point removed.
func (p Polygon) RemovePoint(index int) (Polygon, error) {
	if index < 0 || index >= len(p.points) {
		return Polygon{}, fmt.Errorf("%w: point index %d is out of range", ErrInvalidPolygon, index)
	}
	points := p.Points()
	points = append(points[:index], points[index+1:]...)
	return NewPolygon(points)
}

// IsValid reports whether p satisfies the polygon invariants.
func (p Polygon) IsValid() bool {
	return p.Validate() == nil
}

// Validate checks that p is finite, non-degenerate, and simple.
func (p Polygon) Validate() error {
	if len(p.points) < 3 {
		return ErrPolygonTooFewPoints
	}
	for index, point := range p.points {
		if !isFinite(point.X) || !isFinite(point.Y) {
			return ErrPolygonNonFinitePoint
		}
		for other := 0; other < index; other++ {
			if point == p.points[other] {
				return ErrPolygonDuplicatePoint
			}
		}
	}
	for index := range p.points {
		previous := (index + len(p.points) - 1) % len(p.points)
		next := (index + 1) % len(p.points)
		if p.points[index] == p.points[next] {
			return ErrPolygonDuplicatePoint
		}
		if p.points[index].Sub(p.points[previous]).Cross(p.points[next].Sub(p.points[index])) == 0 {
			return ErrPolygonCollinearPoint
		}
	}
	for first := 0; first < len(p.points); first++ {
		firstNext := (first + 1) % len(p.points)
		for second := first + 1; second < len(p.points); second++ {
			secondNext := (second + 1) % len(p.points)
			if first == second || firstNext == second || secondNext == first {
				continue
			}
			if segmentsIntersect(p.points[first], p.points[firstNext], p.points[second], p.points[secondNext]) {
				return ErrPolygonSelfIntersecting
			}
		}
	}
	if p.SignedArea() == 0 {
		return ErrPolygonZeroArea
	}
	return nil
}

func reversePoints(points []Vector2) {
	for left, right := 0, len(points)-1; left < right; left, right = left+1, right-1 {
		points[left], points[right] = points[right], points[left]
	}
}

func pointOnSegment(point, start, end Vector2) bool {
	if (end.Sub(start)).Cross(point.Sub(start)) != 0 {
		return false
	}
	return point.X >= math.Min(start.X, end.X) && point.X <= math.Max(start.X, end.X) &&
		point.Y >= math.Min(start.Y, end.Y) && point.Y <= math.Max(start.Y, end.Y)
}

func segmentsIntersect(firstStart, firstEnd, secondStart, secondEnd Vector2) bool {
	orientation := func(a, b, c Vector2) float64 {
		return b.Sub(a).Cross(c.Sub(a))
	}
	o1 := orientation(firstStart, firstEnd, secondStart)
	o2 := orientation(firstStart, firstEnd, secondEnd)
	o3 := orientation(secondStart, secondEnd, firstStart)
	o4 := orientation(secondStart, secondEnd, firstEnd)
	if o1 == 0 && pointOnSegment(secondStart, firstStart, firstEnd) {
		return true
	}
	if o2 == 0 && pointOnSegment(secondEnd, firstStart, firstEnd) {
		return true
	}
	if o3 == 0 && pointOnSegment(firstStart, secondStart, secondEnd) {
		return true
	}
	if o4 == 0 && pointOnSegment(firstEnd, secondStart, secondEnd) {
		return true
	}
	return (o1 > 0) != (o2 > 0) && (o3 > 0) != (o4 > 0)
}
