package geom

import "math"

// Circle is a two-dimensional circle represented by its center and radius.
type Circle struct {
	// Center is the center point.
	Center Vector2
	// Radius is the radius; its magnitude is used, so negative values behave
	// like their positive counterparts.
	Radius float64
}

// Contains reports whether point lies inside or on the circle's boundary.
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

// OverlapsOrTouches reports whether c and other overlap or share a boundary
// point.
func (c Circle) OverlapsOrTouches(other Circle) bool {
	radius := math.Abs(c.Radius) + math.Abs(other.Radius)
	return c.Center.DistanceSquared(other.Center) <= radius*radius
}

// OverlapsOrTouchesRect reports whether c overlaps or touches r. It returns
// false when r is invalid.
func (c Circle) OverlapsOrTouchesRect(r Rect) bool {
	if !r.IsValid() {
		return false
	}
	radius := math.Abs(c.Radius)
	return r.DistanceSquared(c.Center) <= radius*radius
}

// BoundingBox returns the smallest [Rect] containing c.
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
