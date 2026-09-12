package geom

import (
	"fmt"
	"math"
)

// Rect is a two-dimensional axis-aligned bounding rectangle.
// Canonical rectangles have Min no greater than Max on either axis.
type Rect struct {
	Min Vector2
	Max Vector2
}

// RectFromCenterSize returns a rectangle centered at center with the given size.
// Negative size components produce the same canonical bounds as their absolute values.
func RectFromCenterSize(center, size Vector2) Rect {
	halfSize := Vector2{X: math.Abs(size.X) / 2, Y: math.Abs(size.Y) / 2}
	return Rect{
		Min: Vector2{X: center.X - halfSize.X, Y: center.Y - halfSize.Y},
		Max: Vector2{X: center.X + halfSize.X, Y: center.Y + halfSize.Y},
	}
}

// RectFromMinMax returns a rectangle from two opposite corners.
// The corners are normalized so Min is no greater than Max on either axis.
func RectFromMinMax(min, max Vector2) Rect {
	return Rect{
		Min: min.Min(max),
		Max: min.Max(max),
	}
}

// Center returns the midpoint between the rectangle's corners.
func (r Rect) Center() Vector2 {
	return Vector2{
		X: (r.Min.X + r.Max.X) / 2,
		Y: (r.Min.Y + r.Max.Y) / 2,
	}
}

// Size returns the rectangle's width and height.
func (r Rect) Size() Vector2 {
	return Vector2{
		X: r.Width(),
		Y: r.Height(),
	}
}

// Width returns the distance between the rectangle's vertical sides.
// It is negative for a non-canonical rectangle created by a struct literal.
func (r Rect) Width() float64 {
	return r.Max.X - r.Min.X
}

// Height returns the distance between the rectangle's horizontal sides.
// It is negative for a non-canonical rectangle created by a struct literal.
func (r Rect) Height() float64 {
	return r.Max.Y - r.Min.Y
}

// Area returns the rectangle's area.
// Invalid or non-canonical rectangles have zero area.
func (r Rect) Area() float64 {
	if !r.IsValid() {
		return 0
	}
	return r.Width() * r.Height()
}

// Extents returns half the rectangle's size in each dimension.
func (r Rect) Extents() Vector2 {
	return Vector2{
		X: r.Width() / 2,
		Y: r.Height() / 2,
	}
}

// Contains reports whether point lies inside or on the rectangle's boundary.
func (r Rect) Contains(point Vector2) bool {
	if !r.IsValid() {
		return false
	}
	return point.X >= r.Min.X && point.X <= r.Max.X &&
		point.Y >= r.Min.Y && point.Y <= r.Max.Y
}

// Intersects reports whether two rectangles overlap with positive area.
// Rectangles that only touch at an edge or corner do not intersect.
func (r Rect) Intersects(other Rect) bool {
	return RectsIntersect(r, other)
}

// OverlapsOrTouches reports whether two rectangles overlap or share any boundary point.
func (r Rect) OverlapsOrTouches(other Rect) bool {
	if !r.IsValid() || !other.IsValid() {
		return false
	}
	return r.Min.X <= other.Max.X && r.Max.X >= other.Min.X &&
		r.Min.Y <= other.Max.Y && r.Max.Y >= other.Min.Y
}

// IsEmpty reports whether the rectangle has zero area or invalid dimensions.
func (r Rect) IsEmpty() bool {
	return !r.IsValid() || r.Width() == 0 || r.Height() == 0
}

// IsValid reports whether the rectangle has finite, canonical bounds.
func (r Rect) IsValid() bool {
	return isFinite(r.Min.X) && isFinite(r.Min.Y) &&
		isFinite(r.Max.X) && isFinite(r.Max.Y) &&
		r.Min.X <= r.Max.X && r.Min.Y <= r.Max.Y
}

// Normalize returns a canonical rectangle whose Min components are no greater
// than their corresponding Max components.
func (r Rect) Normalize() Rect {
	return RectFromMinMax(r.Min, r.Max)
}

// Union returns the smallest rectangle containing r and other.
func (r Rect) Union(other Rect) Rect {
	r = r.Normalize()
	other = other.Normalize()
	return Rect{
		Min: r.Min.Min(other.Min),
		Max: r.Max.Max(other.Max),
	}
}

// Intersection returns the positive-area overlap of r and other.
// The second result is false when the rectangles do not overlap positively.
func (r Rect) Intersection(other Rect) (Rect, bool) {
	if !r.Intersects(other) {
		return Rect{}, false
	}
	return Rect{
		Min: r.Min.Max(other.Min),
		Max: r.Max.Min(other.Max),
	}, true
}

// Expand grows r outward by amount on every side.
// Negative amounts shrink the rectangle and clamp each extent at zero.
func (r Rect) Expand(amount float64) Rect {
	center := r.Center()
	extents := r.Extents()
	extents.X = math.Max(0, math.Abs(extents.X)+amount)
	extents.Y = math.Max(0, math.Abs(extents.Y)+amount)
	return RectFromCenterSize(center, extents.Mul(2))
}

// ClosestPoint returns the point in r nearest to point.
// An invalid rectangle returns the zero vector.
func (r Rect) ClosestPoint(point Vector2) Vector2 {
	if !r.IsValid() {
		return Vector2{}
	}
	return point.Clamp(r.Min, r.Max)
}

// DistanceSquared returns the squared distance from point to r.
// It returns zero for points inside or on the rectangle and avoids a square root.
func (r Rect) DistanceSquared(point Vector2) float64 {
	if !r.IsValid() {
		return 0
	}
	return point.DistanceSquared(r.ClosestPoint(point))
}

// String returns a human-readable representation of r.
func (r Rect) String() string {
	return fmt.Sprintf("Rect{Min: %v, Max: %v}", r.Min, r.Max)
}

// BoundingBox returns r because a rectangle is already axis-aligned.
func (r Rect) BoundingBox() Rect {
	return r
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
