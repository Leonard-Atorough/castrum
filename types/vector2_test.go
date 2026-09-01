package types

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func vecAlmostEqual(a, b Vector2) bool {
	return almostEqual(a.X, b.X) && almostEqual(a.Y, b.Y)
}

func TestVector2_Arithmetic(t *testing.T) {
	a := Vector2{X: 3, Y: 4}
	b := Vector2{X: 1, Y: 2}

	cases := []struct {
		name string
		got  Vector2
		want Vector2
	}{
		{"Add", a.Add(b), Vector2{X: 4, Y: 6}},
		{"Sub", a.Sub(b), Vector2{X: 2, Y: 2}},
		{"Mul", a.Mul(2), Vector2{X: 6, Y: 8}},
		{"Div", a.Div(2), Vector2{X: 1.5, Y: 2}},
		{"Neg", a.Neg(), Vector2{X: -3, Y: -4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !vecAlmostEqual(tc.got, tc.want) {
				t.Fatalf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestVector2_Length(t *testing.T) {
	v := Vector2{X: 3, Y: 4}
	if got := v.Length(); !almostEqual(got, 5) {
		t.Fatalf("Length() = %v, want 5", got)
	}
}

func TestVector2_Normalize(t *testing.T) {
	t.Run("non-zero vector becomes unit length", func(t *testing.T) {
		v := Vector2{X: 3, Y: 4}
		got := v.Normalize()
		if !almostEqual(got.Length(), 1) {
			t.Fatalf("normalized length = %v, want 1", got.Length())
		}
		if !vecAlmostEqual(got, Vector2{X: 0.6, Y: 0.8}) {
			t.Fatalf("Normalize() = %v, want {0.6 0.8}", got)
		}
	})

	t.Run("zero vector normalizes to zero instead of NaN", func(t *testing.T) {
		got := Vector2{}.Normalize()
		if got != (Vector2{}) {
			t.Fatalf("Normalize() of zero vector = %v, want zero vector", got)
		}
	})
}

func TestVector2_DotAndCross(t *testing.T) {
	a := Vector2{X: 1, Y: 0}
	b := Vector2{X: 0, Y: 1}

	if got := a.Dot(b); got != 0 {
		t.Fatalf("Dot() = %v, want 0 for perpendicular vectors", got)
	}
	if got := a.Cross(b); got != 1 {
		t.Fatalf("Cross() = %v, want 1", got)
	}
}

func TestVector2_Angle(t *testing.T) {
	a := Vector2{X: 1, Y: 0}
	b := Vector2{X: 0, Y: 1}
	if got := a.Angle(b); !almostEqual(got, math.Pi/2) {
		t.Fatalf("Angle() = %v, want pi/2", got)
	}
}

func TestVector2_Distance(t *testing.T) {
	a := Vector2{X: 0, Y: 0}
	b := Vector2{X: 3, Y: 4}
	if got := a.Distance(b); !almostEqual(got, 5) {
		t.Fatalf("Distance() = %v, want 5", got)
	}
}

func TestVector2_Lerp(t *testing.T) {
	a := Vector2{X: 0, Y: 0}
	b := Vector2{X: 10, Y: 20}

	cases := []struct {
		t    float64
		want Vector2
	}{
		{0, a},
		{1, b},
		{0.5, Vector2{X: 5, Y: 10}},
	}
	for _, tc := range cases {
		if got := a.Lerp(b, tc.t); !vecAlmostEqual(got, tc.want) {
			t.Fatalf("Lerp(%v) = %v, want %v", tc.t, got, tc.want)
		}
	}
}

func TestVector2_Reflect(t *testing.T) {
	v := Vector2{X: 1, Y: -1}
	normal := Vector2{X: 0, Y: 1}
	got := v.Reflect(normal)
	if !vecAlmostEqual(got, Vector2{X: 1, Y: 1}) {
		t.Fatalf("Reflect() = %v, want {1 1}", got)
	}
}

func TestVector2_Project(t *testing.T) {
	t.Run("projects onto a non-zero vector", func(t *testing.T) {
		v := Vector2{X: 2, Y: 2}
		onto := Vector2{X: 1, Y: 0}
		got := v.Project(onto)
		if !vecAlmostEqual(got, Vector2{X: 2, Y: 0}) {
			t.Fatalf("Project() = %v, want {2 0}", got)
		}
	})

	t.Run("projecting onto a zero vector returns zero instead of NaN", func(t *testing.T) {
		got := Vector2{X: 1, Y: 1}.Project(Vector2{})
		if got != (Vector2{}) {
			t.Fatalf("Project() onto zero vector = %v, want zero vector", got)
		}
	})
}

func TestVector2_Rotate(t *testing.T) {
	v := Vector2{X: 1, Y: 0}
	got := v.Rotate(math.Pi / 2)
	if !vecAlmostEqual(got, Vector2{X: 0, Y: 1}) {
		t.Fatalf("Rotate(pi/2) = %v, want {0 1}", got)
	}
}

func TestVector2_String(t *testing.T) {
	got := Vector2{X: 1, Y: 2}.String()
	want := "Vector2{X: 1.000000, Y: 2.000000}"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
