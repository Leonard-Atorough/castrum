package geom

import "math"

// Segment is a finite line segment between Start and End.
type Segment struct {
	Start Vector2
	End   Vector2
}

// Length returns the distance between the segment's endpoints.
func (s Segment) Length() float64 {
	dx := s.End.X - s.Start.X
	dy := s.End.Y - s.Start.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Midpoint returns the point halfway between the segment's endpoints.
func (s Segment) Midpoint() Vector2 {
	return Vector2{
		X: (s.Start.X + s.End.X) / 2,
		Y: (s.Start.Y + s.End.Y) / 2,
	}
}

// ClosestPoint returns the point on the segment nearest to point. A
// zero-length segment returns its single endpoint.
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

// DistanceToPoint returns the shortest distance from point to the
// segment. Points inside project to zero.
func (s Segment) DistanceToPoint(point Vector2) float64 {
	closest := s.ClosestPoint(point)
	return point.Distance(closest)
}

// BoundingBox returns the canonical axis-aligned rectangle containing
// the segment.
func (s Segment) BoundingBox() Rect {
	return Rect{
		Min: s.Start.Min(s.End),
		Max: s.Start.Max(s.End),
	}
}
