package geom

import "math"

// Segment is a finite line segment between Start and End.
type Segment struct {
	Start Vector2
	End   Vector2
}

// Length returns the distance between the segment endpoints.
func (s Segment) Length() float64 {
	dx := s.End.X - s.Start.X
	dy := s.End.Y - s.Start.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Midpoint returns the point halfway between the segment endpoints.
func (s Segment) Midpoint() Vector2 {
	return Vector2{
		X: (s.Start.X + s.End.X) / 2,
		Y: (s.Start.Y + s.End.Y) / 2,
	}
}

// ClosestPoint returns the point on the segment nearest to point.
// A zero-length segment returns its single endpoint.
func (s Segment) ClosestPoint(point Vector2) Vector2 {
	dx := s.End.X - s.Start.X
	dy := s.End.Y - s.Start.Y
	if dx == 0 && dy == 0 {
		return s.Start
	}
	t := ((point.X-s.Start.X)*dx + (point.Y-s.Start.Y)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		return s.Start
	}
	if t > 1 {
		return s.End
	}
	return Vector2{
		X: s.Start.X + t*dx,
		Y: s.Start.Y + t*dy,
	}
}

// DistanceToPoint returns the shortest distance from point to the segment.
func (s Segment) DistanceToPoint(point Vector2) float64 {
	closest := s.ClosestPoint(point)
	dx := point.X - closest.X
	dy := point.Y - closest.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// BoundingBox returns the canonical axis-aligned bounding box containing the segment.
func (s Segment) BoundingBox() Rect {
	min := Vector2{
		X: math.Min(s.Start.X, s.End.X),
		Y: math.Min(s.Start.Y, s.End.Y),
	}
	max := Vector2{
		X: math.Max(s.Start.X, s.End.X),
		Y: math.Max(s.Start.Y, s.End.Y),
	}
	return Rect{Min: min, Max: max}
}

// Intersects reports whether s and other share at least one point.
// This includes endpoint touching, collinear overlap, and zero-length segments.
func (s Segment) Intersects(other Segment) bool {
	return segmentsIntersect(s.Start, s.End, other.Start, other.End)
}
