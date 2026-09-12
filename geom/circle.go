package geom

import "math"

type Circle struct {
	Center Vector2
	Radius float64
}

// Contains reports whether point lies inside or on the circle's boundary.
func (c Circle) Contains(point Vector2) bool {
	radius := abs(c.Radius)
	return c.Center.DistanceSquared(point) <= radius*radius
}

// Intersects reports whether c overlaps or touches other.
func (c Circle) Intersects(other Circle) bool {
	return CirclesIntersect(c, other)
}

// Area returns the area enclosed by c.
func (c Circle) Area() float64 {
	radius := abs(c.Radius)
	return math.Pi * radius * radius
}

// Circumference returns the circumference of c.
func (c Circle) Circumference() float64 {
	return 2 * math.Pi * abs(c.Radius)
}

// BoundingBox returns the smallest axis-aligned rectangle containing c.
func (c Circle) BoundingBox() Rect {
	radius := abs(c.Radius)
	return Rect{
		Min: Vector2{
			X: c.Center.X - radius,
			Y: c.Center.Y - radius,
		},
		Max: Vector2{
			X: c.Center.X + radius,
			Y: c.Center.Y + radius,
		},
	}
}
