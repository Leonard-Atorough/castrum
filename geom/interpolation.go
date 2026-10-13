package geom

import "math"

// Lerp linearly interpolates from from to to by weight.
// Values outside [0, 1] extrapolate beyond the endpoints.
func Lerp(from, to, weight float64) float64 {
	return from + (to-from)*weight
}

// Ease maps normalized time to an interpolation weight.
// Use the result with [Lerp] or [Vector2.Lerp] to shape motion.
//
// The curves provided by this package map [0, 1] to [0, 1], with
// endpoints at 0 and 1. They do not clamp inputs outside this range.
type Ease func(t float64) float64

// EaseLinear returns t unchanged for constant-speed interpolation.
func EaseLinear(t float64) float64 {
	return t
}

// EaseInQuad returns a quadratic easing weight that starts slowly and accelerates.
func EaseInQuad(t float64) float64 {
	return math.Pow(t, 2)
}

// EaseOutQuad returns a quadratic easing weight that starts quickly and decelerates.
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutQuad returns a quadratic easing weight that accelerates in the
// first half and decelerates in the second.
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * math.Pow(t, 2)
	}
	return -1 + (4-2*t)*t
}

// EaseInCubic returns a cubic easing weight that starts slowly and accelerates.
func EaseInCubic(t float64) float64 {
	return math.Pow(t, 3)
}

// EaseOutCubic returns a cubic easing weight that starts quickly and decelerates.
func EaseOutCubic(t float64) float64 {
	t -= 1
	return math.Pow(t, 3) + 1
}

// EaseInOutCubic returns a cubic easing weight that accelerates in the
// first half and decelerates in the second.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * math.Pow(t, 3)
	}
	t -= 1
	return 4*math.Pow(t, 3) + 1
}
