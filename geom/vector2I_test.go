package geom

import (
	"math"
	"testing"
)

func TestVector2I_Arithmetic(t *testing.T) {
	a := Vector2I{X: 3, Y: 4}
	b := Vector2I{X: 1, Y: 2}

	cases := []struct {
		name string
		got  Vector2I
		want Vector2I
	}{
		{"Add", a.Add(b), Vector2I{X: 4, Y: 6}},
		{"Sub", a.Sub(b), Vector2I{X: 2, Y: 2}},
		{"Mul", a.Mul(2), Vector2I{X: 6, Y: 8}},
		{"Div", a.Div(2), Vector2I{X: 1, Y: 2}},
		{"Neg", a.Neg(), Vector2I{X: -3, Y: -4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestVector2I_Length(t *testing.T) {
	v := Vector2I{X: 3, Y: 4}
	if got := v.Length(); got != 5 {
		t.Fatalf("Length() = %v, want 5", got)
	}
}

func TestVector2I_Normalize(t *testing.T) {
	t.Run("zero vector normalizes to zero", func(t *testing.T) {
		got := Vector2I{}.Normalize()
		if got != (Vector2I{}) {
			t.Fatalf("Normalize() of zero vector = %v, want zero vector", got)
		}
	})

	// Integer division means Normalize only produces a meaningful unit vector
	// for lengths of 1; anything else truncates to zero. Documented here so
	// the behavior doesn't come as a surprise to callers.
	t.Run("non-unit length truncates to zero via integer division", func(t *testing.T) {
		got := Vector2I{X: 3, Y: 4}.Normalize()
		if got != (Vector2I{}) {
			t.Fatalf("Normalize() = %v, want {0 0} (integer truncation)", got)
		}
	})
}

func TestVector2I_DotAndCross(t *testing.T) {
	a := Vector2I{X: 1, Y: 0}
	b := Vector2I{X: 0, Y: 1}

	if got := a.Dot(b); got != 0 {
		t.Fatalf("Dot() = %v, want 0", got)
	}
	if got := a.Cross(b); got != 1 {
		t.Fatalf("Cross() = %v, want 1", got)
	}
}

func TestVector2I_Angle(t *testing.T) {
	a := Vector2I{X: 1, Y: 0}
	b := Vector2I{X: 0, Y: 1}
	if got := a.Angle(b); math.Abs(got-math.Pi/2) > 1e-9 {
		t.Fatalf("Angle() = %v, want pi/2", got)
	}
}

func TestVector2I_Distance(t *testing.T) {
	a := Vector2I{X: 0, Y: 0}
	b := Vector2I{X: 3, Y: 4}
	if got := a.Distance(b); got != 5 {
		t.Fatalf("Distance() = %v, want 5", got)
	}
}

func TestVector2I_Lerp(t *testing.T) {
	a := Vector2I{X: 0, Y: 0}
	b := Vector2I{X: 10, Y: 20}
	if got := a.Lerp(b, 0.5); got != (Vector2I{X: 5, Y: 10}) {
		t.Fatalf("Lerp(0.5) = %v, want {5 10}", got)
	}
}

func TestVector2I_Reflect(t *testing.T) {
	v := Vector2I{X: 1, Y: -1}
	normal := Vector2I{X: 0, Y: 1}
	if got := v.Reflect(normal); got != (Vector2I{X: 1, Y: 1}) {
		t.Fatalf("Reflect() = %v, want {1 1}", got)
	}
}

func TestVector2I_Project(t *testing.T) {
	t.Run("projecting onto a zero vector returns zero instead of dividing by zero", func(t *testing.T) {
		got := Vector2I{X: 1, Y: 1}.Project(Vector2I{})
		if got != (Vector2I{}) {
			t.Fatalf("Project() onto zero vector = %v, want zero vector", got)
		}
	})
}

func TestVector2I_Rotate(t *testing.T) {
	v := Vector2I{X: 1, Y: 0}
	got := v.Rotate(math.Pi / 2)
	if got != (Vector2I{X: 0, Y: 1}) {
		t.Fatalf("Rotate(pi/2) = %v, want {0 1}", got)
	}
}

func TestVector2I_String(t *testing.T) {
	got := Vector2I{X: 1, Y: 2}.String()
	if want := "Vector2I{X: 1, Y: 2}"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
