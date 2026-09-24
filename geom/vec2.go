package geom

import (
	"fmt"
	"math"
)

type Vector2 struct {
	X, Y float64
}

// Add returns the component-wise sum of v and other.
// It is useful for positions, offsets, and accumulated motion.
func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}

// Sub returns the component-wise difference between v and other.
// It is useful for displacement and direction calculations.
func (v Vector2) Sub(other Vector2) Vector2 {
	return Vector2{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

// Mul returns v scaled by scalar.
// It is useful for changing a vector's magnitude without changing its direction.
func (v Vector2) Mul(scalar float64) Vector2 {
	return Vector2{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

// Div returns v divided by scalar.
// If the scalar is zero, it returns a zero vector to avoid infinities or NaNs.
func (v Vector2) Div(scalar float64) Vector2 {
	if scalar == 0 {
		return Vector2{X: 0, Y: 0}
	}
	return Vector2{
		X: v.X / scalar,
		Y: v.Y / scalar,
	}
}

// Neg returns v with both components negated.
// It is useful for reversing a direction or displacement.
func (v Vector2) Neg() Vector2 {
	return Vector2{
		X: -v.X,
		Y: -v.Y,
	}
}

// Length returns the Euclidean magnitude of v.
// It is useful for measuring the size of a vector.
func (v Vector2) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

// LengthSquared returns the squared Euclidean magnitude of v.
// It is useful for distance comparisons without the cost of a square root.
func (v Vector2) LengthSquared() float64 {
	return v.X*v.X + v.Y*v.Y
}

// Normalize returns a unit vector in v's direction.
// A zero vector returns a zero vector; non-finite input propagates non-finite results.
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

// Dot returns the scalar dot product of v and other.
// It is useful for projections, angles, and checking relative direction.
func (v Vector2) Dot(other Vector2) float64 {
	return v.X*other.X + v.Y*other.Y
}

// Cross returns the scalar two-dimensional cross product of v and other.
// A positive result means other is counter-clockwise from v; a negative result means clockwise.
func (v Vector2) Cross(other Vector2) float64 {
	return v.X*other.Y - v.Y*other.X
}

// Angle returns the signed angle in radians from v to other.
// Positive values indicate counter-clockwise rotation; zero vectors have no meaningful angle.
func (v Vector2) Angle(other Vector2) float64 {
	dot := v.Dot(other)
	cross := v.Cross(other)
	return math.Atan2(cross, dot)
}

// AngleDeg returns the signed angle in degrees from v to other.
// It is useful at APIs that expose angles to designers or other human-facing tools.
func (v Vector2) AngleDeg(other Vector2) float64 {
	return v.Angle(other) * (180 / math.Pi)
}

// Distance returns the Euclidean distance between v and other.
// Use DistanceSquared when only comparing distances.
func (v Vector2) Distance(other Vector2) float64 {
	return v.Sub(other).Length()
}

// DistanceSquared returns the squared Euclidean distance between v and other.
// It is useful for proximity checks without a square root.
func (v Vector2) DistanceSquared(other Vector2) float64 {
	return v.Sub(other).LengthSquared()
}

// Lerp linearly interpolates from v to other by t.
// Values outside [0, 1] extrapolate beyond the endpoints.
func (v Vector2) Lerp(other Vector2, t float64) Vector2 {
	return Vector2{
		X: v.X + (other.X-v.X)*t,
		Y: v.Y + (other.Y-v.Y)*t,
	}
}

// Reflect returns v reflected across a surface with the given normal.
// The normal must be unit length; a zero normal returns v unchanged.
func (v Vector2) Reflect(normal Vector2) Vector2 {
	if normal.IsZero() {
		return v
	}
	dot := v.Dot(normal)
	return v.Sub(normal.Mul(2 * dot))
}

// Project returns the component of v parallel to onto.
// Projection onto a zero vector returns a zero vector.
func (v Vector2) Project(onto Vector2) Vector2 {
	if onto.IsZero() {
		return Vector2{X: 0, Y: 0}
	}
	dot := v.Dot(onto)
	ontoLengthSq := onto.Dot(onto)
	return onto.Mul(dot / ontoLengthSq)
}

// Rotate returns v rotated counter-clockwise by angle radians.
// Rotation preserves the vector's magnitude apart from floating-point error.
func (v Vector2) Rotate(angle float64) Vector2 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vector2{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
	}
}

// Clamp clamps each component of v to the corresponding range in min and max.
// Reversed bounds are normalized per component before clamping.
func (v Vector2) Clamp(min, max Vector2) Vector2 {
	return Vector2{
		X: math.Max(math.Min(min.X, max.X), math.Min(math.Max(min.X, max.X), v.X)),
		Y: math.Max(math.Min(min.Y, max.Y), math.Min(math.Max(min.Y, max.Y), v.Y)),
	}
}

// ClampMagnitude limits v's magnitude to maxLength without changing its direction.
// A non-positive maxLength returns a zero vector.
func (v Vector2) ClampMagnitude(maxLength float64) Vector2 {
	if maxLength <= 0 {
		return Vector2{X: 0, Y: 0}
	}
	if v.LengthSquared() > maxLength*maxLength {
		return v.Normalize().Mul(maxLength)
	}
	return v
}

// IsZero reports whether both components of v are exactly zero.
// It is not an epsilon comparison and returns false for non-finite values.
func (v Vector2) IsZero() bool {
	return v.X == 0 && v.Y == 0
}

// AlmostEqual reports whether the components of v and other differ by no more than epsilon.
// Negative epsilon always returns false.
func (v Vector2) AlmostEqual(other Vector2, epsilon float64) bool {
	return math.Abs(v.X-other.X) <= epsilon && math.Abs(v.Y-other.Y) <= epsilon
}

// Min returns a vector containing the minimum components of this vector and another vector.
func (v Vector2) Min(other Vector2) Vector2 {
	return Vector2{
		X: math.Min(v.X, other.X),
		Y: math.Min(v.Y, other.Y),
	}
}

// Max returns a vector containing the maximum components of this vector and another vector.
func (v Vector2) Max(other Vector2) Vector2 {
	return Vector2{
		X: math.Max(v.X, other.X),
		Y: math.Max(v.Y, other.Y),
	}
}

// String returns the string representation of the vector.
func (v Vector2) String() string {
	return fmt.Sprintf("Vector2{X: %f, Y: %f}", v.X, v.Y)
}
