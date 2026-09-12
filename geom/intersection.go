package geom

// CirclesIntersect reports whether two circles overlap or touch.
// Circle radii are treated by magnitude so a negative radius cannot invert the test.
func CirclesIntersect(a, b Circle) bool {
	radiusSum := abs(a.Radius) + abs(b.Radius)
	return a.Center.DistanceSquared(b.Center) <= radiusSum*radiusSum
}

// RectsIntersect reports whether two axis-aligned rectangles overlap with
// positive area. Rectangles that only touch at an edge or corner do not
// intersect.
func RectsIntersect(a, b Rect) bool {
	if !a.IsValid() || !b.IsValid() {
		return false
	}
	return a.Min.X < b.Max.X && a.Max.X > b.Min.X &&
		a.Min.Y < b.Max.Y && a.Max.Y > b.Min.Y
}

// CircleRectIntersects reports whether circle and rect overlap or touch.
// The rectangle must be axis-aligned; use physics collision routines for
// transformed or oriented rectangles.
func CircleRectIntersects(circle Circle, rect Rect) bool {
	if !rect.IsValid() {
		return false
	}
	closest := circle.Center.Clamp(rect.Min, rect.Max)
	radius := abs(circle.Radius)
	return circle.Center.DistanceSquared(closest) <= radius*radius
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
