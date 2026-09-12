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
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestVector2I_DotAndCross(t *testing.T) {
	a := Vector2I{X: 1, Y: 0}
	b := Vector2I{X: 0, Y: 1}

	if got := a.Dot(b); got != 0 {
		t.Errorf("Dot() = %v, want 0", got)
	}
	if got := a.Cross(b); got != 1 {
		t.Errorf("Cross() = %v, want 1", got)
	}
}

func TestVector2I_Angle(t *testing.T) {
	a := Vector2I{X: 1, Y: 0}
	b := Vector2I{X: 0, Y: 1}
	if got := a.Angle(b); math.Abs(got-math.Pi/2) > 1e-9 {
		t.Errorf("Angle() = %v, want pi/2", got)
	}
}

func TestVector2I_Distance(t *testing.T) {
	a := Vector2I{X: 0, Y: 0}
	b := Vector2I{X: 3, Y: 4}
	if got := a.Distance(b); got != 5 {
		t.Errorf("Distance() = %v, want 5", got)
	}
}

func TestVector2I_Lerp(t *testing.T) {
	a := Vector2I{X: 0, Y: 0}
	b := Vector2I{X: 10, Y: 20}
	if got := a.Lerp(b, 0.5); got != (Vector2I{X: 5, Y: 10}) {
		t.Errorf("Lerp(0.5) = %v, want {5 10}", got)
	}
}

func TestVector2I_Reflect(t *testing.T) {
	v := Vector2I{X: 1, Y: -1}
	normal := Vector2I{X: 0, Y: 1}
	if got := v.Reflect(normal); got != (Vector2I{X: 1, Y: 1}) {
		t.Errorf("Reflect() = %v, want {1 1}", got)
	}
}


func TestVector2I_String(t *testing.T) {
	got := Vector2I{X: 1, Y: 2}.String()
	if want := "Vector2I{X: 1, Y: 2}"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
