package geom

import (
	"fmt"
	"math"
)

type Vector2I struct {
	X, Y int
}

func NewVector2I(x, y int) Vector2I {
	return Vector2I{X: x, Y: y}
}

func (v Vector2I) Add(other Vector2I) Vector2I {
	return Vector2I{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}

func (v Vector2I) Sub(other Vector2I) Vector2I {
	return Vector2I{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

func (v Vector2I) Mul(scalar int) Vector2I {
	return Vector2I{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

func (v Vector2I) Div(scalar int) Vector2I {
	return Vector2I{
		X: v.X / scalar,
		Y: v.Y / scalar,
	}
}

func (v Vector2I) Neg() Vector2I {
	return Vector2I{
		X: -v.X,
		Y: -v.Y,
	}
}

func (v Vector2I) Length() int {
	return int(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}
func (v Vector2I) Normalize() Vector2I {
	length := v.Length()
	if length == 0 {
		return Vector2I{X: 0, Y: 0}
	}
	return Vector2I{
		X: v.X / length,
		Y: v.Y / length,
	}
}

func (v Vector2I) Dot(other Vector2I) int {
	return v.X*other.X + v.Y*other.Y
}

func (v Vector2I) Cross(other Vector2I) int {
	return v.X*other.Y - v.Y*other.X
}

func (v Vector2I) Angle(other Vector2I) float64 {
	dot := v.Dot(other)
	cross := v.Cross(other)
	return math.Atan2(float64(cross), float64(dot))
}

func (v Vector2I) Distance(other Vector2I) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return math.Sqrt(float64(dx*dx + dy*dy))
}

func (v Vector2I) Lerp(other Vector2I, t float64) Vector2I {
	return Vector2I{
		X: v.X + int(float64(other.X-v.X)*t),
		Y: v.Y + int(float64(other.Y-v.Y)*t),
	}
}

func (v Vector2I) Reflect(normal Vector2I) Vector2I {
	dot := v.Dot(normal)
	return v.Sub(normal.Mul(2 * dot))
}

func (v Vector2I) Project(onto Vector2I) Vector2I {
	dot := v.Dot(onto)
	ontoLengthSq := onto.Dot(onto)
	if ontoLengthSq == 0 {
		return Vector2I{X: 0, Y: 0}
	}
	return onto.Mul(dot / ontoLengthSq)
}

func (v Vector2I) Rotate(angle float64) Vector2I {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vector2I{
		X: int(float64(v.X)*cos - float64(v.Y)*sin),
		Y: int(float64(v.X)*sin + float64(v.Y)*cos),
	}
}

func (v Vector2I) String() string {
	return fmt.Sprintf("Vector2I{X: %d, Y: %d}", v.X, v.Y)
}
