package geom

import "fmt"

//
type Rect struct {
	Min Vector2
	Max Vector2
}

func NewRect(min, max Vector2) Rect {
	return Rect{Min: min, Max: max}
}

func (r Rect) Width() float64 {
	return r.Max.X - r.Min.X
}

func (r Rect) Height() float64 {
	return r.Max.Y - r.Min.Y
}

func (r Rect) Area() float64 {
	return r.Width() * r.Height()
}

func (r Rect) Contains(point Vector2) bool {
	return point.X >= r.Min.X && point.X <= r.Max.X &&
		point.Y >= r.Min.Y && point.Y <= r.Max.Y
}

func (r Rect) Intersects(other Rect) bool {
	return r.Min.X < other.Max.X && r.Max.X > other.Min.X &&
		r.Min.Y < other.Max.Y && r.Max.Y > other.Min.Y
}

func (r Rect) String() string {
	return fmt.Sprintf("Rect{Min: %v, Max: %v}", r.Min, r.Max)
}
