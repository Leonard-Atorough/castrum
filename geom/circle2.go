package geom

import "math"

// Circle is a two-dimensional circle: a center and a radius.
type Circle struct {
	Center Vector2
	Radius float64
}

// Contains reports whether point lies inside or on the circle's
// boundary. The radius is treated by magnitude, so a negative radius
// cannot invert the test.
func (c Circle) Contains(point Vector2) bool {
	radius := math.Abs(c.Radius)
	return c.Center.DistanceSquared(point) <= radius*radius
}

// Area returns the area enclosed by the circle.
func (c Circle) Area() float64 {
	radius := math.Abs(c.Radius)
	return math.Pi * radius * radius
}

// Circumference returns the circle's circumference.
func (c Circle) Circumference() float64 {
	return 2 * math.Pi * math.Abs(c.Radius)
}

// BoundingBox returns the smallest axis-aligned rectangle containing
// the circle.
func (c Circle) BoundingBox() Rect {
	radius := math.Abs(c.Radius)
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
