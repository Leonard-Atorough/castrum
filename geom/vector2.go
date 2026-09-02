package geom

import (
	"fmt"
	"math"
)

type Vector2 struct {
	X, Y float64
}

func NewVector2(x, y float64) Vector2 {
	return Vector2{X: x, Y: y}
}

func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}

func (v Vector2) Sub(other Vector2) Vector2 {
	return Vector2{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

func (v Vector2) Mul(scalar float64) Vector2 {
	return Vector2{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

func (v Vector2) Div(scalar float64) Vector2 {
	return Vector2{
		X: v.X / scalar,
		Y: v.Y / scalar,
	}
}

func (v Vector2) Neg() Vector2 {
	return Vector2{
		X: -v.X,
		Y: -v.Y,
	}
}

func (v Vector2) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector2) Normalize() Vector2 {
	length := v.Length()
	if length == 0 {
		return Vector2{X: 0, Y: 0}
	}
	return Vector2{
		X: v.X / length,
		Y: v.Y / length,
	}
}

func (v Vector2) Dot(other Vector2) float64 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vector2) Cross(other Vector2) float64 {
	return v.X*other.Y - v.Y*other.X
}

func (v Vector2) Angle(other Vector2) float64 {
	dot := v.Dot(other)
	cross := v.Cross(other)
	return math.Atan2(cross, dot)
}

func (v Vector2) Distance(other Vector2) float64 {
	return v.Sub(other).Length()
}

func (v Vector2) Lerp(other Vector2, t float64) Vector2 {
	return Vector2{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
	}
}

func (v Vector2) Reflect(normal Vector2) Vector2 {
	dot := v.Dot(normal)
	return v.Sub(normal.Mul(2 * dot))
}

func (v Vector2) Project(onto Vector2) Vector2 {
	dot := v.Dot(onto)
	ontoLengthSq := onto.Dot(onto)
	if ontoLengthSq == 0 {
		return Vector2{X: 0, Y: 0}
	}
	return onto.Mul(dot / ontoLengthSq)
}

func (v Vector2) Rotate(angle float64) Vector2 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vector2{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
	}
}

func (v Vector2) String() string {
	return fmt.Sprintf("Vector2{X: %f, Y: %f}", v.X, v.Y)
}
