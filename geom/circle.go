package geom

import "math"

type Circle struct {
	Center Vector2
	Radius float64
}

func (c Circle) Contains(point Vector2) bool {
	dx := point.X - c.Center.X
	dy := point.Y - c.Center.Y
	return dx*dx+dy*dy <= c.Radius*c.Radius
}

func (c Circle) Intersects(other Circle) bool {
	dx := other.Center.X - c.Center.X
	dy := other.Center.Y - c.Center.Y
	return dx*dx+dy*dy <= (c.Radius+other.Radius)*(c.Radius+other.Radius)
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Circumference() float64 {
	return 2 * math.Pi * c.Radius
}
