package geom

import (
	"math"
	"testing"
)

func TestEaseCurves(t *testing.T) {
	curves := map[string]Ease{
		"linear":       EaseLinear,
		"in quad":      EaseInQuad,
		"out quad":     EaseOutQuad,
		"in-out quad":  EaseInOutQuad,
		"in cubic":     EaseInCubic,
		"out cubic":    EaseOutCubic,
		"in-out cubic": EaseInOutCubic,
	}
	for name, ease := range curves {
		if got := ease(0); got != 0 {
			t.Errorf("%s: ease(0) = %v, want 0", name, got)
		}
		if got := ease(1); math.Abs(got-1) > 1e-12 {
			t.Errorf("%s: ease(1) = %v, want 1", name, got)
		}
		if got := ease(0.5); got < 0 || got > 1 {
			t.Errorf("%s: ease(0.5) = %v, want progress within [0, 1]", name, got)
		}
		for i := 1; i <= 10; i++ {
			prev := ease(float64(i-1) / 10)
			next := ease(float64(i) / 10)
			if next < prev {
				t.Errorf("%s: not monotonic between %v and %v", name, prev, next)
			}
		}
	}

	// The in-out curves are continuous at the midpoint: both halves
	// produce exactly 0.5 there.
	for name, ease := range map[string]Ease{
		"in-out quad":  EaseInOutQuad,
		"in-out cubic": EaseInOutCubic,
	} {
		if got := ease(0.5); math.Abs(got-0.5) > 1e-12 {
			t.Errorf("%s: ease(0.5) = %v, want 0.5 - the halves must join", name, got)
		}
	}

	// Exact spot values pin the formulas against accidental edits.
	spots := []struct {
		ease Ease
		t, w float64
	}{
		{EaseInQuad, 0.25, 0.0625},
		{EaseOutQuad, 0.25, 0.4375},
		{EaseInCubic, 0.5, 0.125},
		{EaseOutCubic, 0.5, 0.875},
		{EaseInOutQuad, 0.25, 0.125},
		{EaseInOutCubic, 0.75, 0.9375},
	}
	for _, s := range spots {
		if got := s.ease(s.t); math.Abs(got-s.w) > 1e-12 {
			t.Errorf("ease(%v) = %v, want %v", s.t, got, s.w)
		}
	}
}

func TestEaseSymmetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Ease
		out  Ease
	}{
		{"linear", EaseLinear, EaseLinear},
		{"quadratic", EaseInQuad, EaseOutQuad},
		{"cubic", EaseInCubic, EaseOutCubic},
		{"quadratic in-out", EaseInOutQuad, EaseInOutQuad},
		{"cubic in-out", EaseInOutCubic, EaseInOutCubic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for i := 0; i <= 100; i++ {
				progress := float64(i) / 100
				got := tt.out(progress)
				want := 1 - tt.in(1-progress)
				if math.Abs(got-want) > 1e-12 {
					t.Errorf("out(%v) = %v, want 1-in(1-t) = %v", progress, got, want)
				}
			}
		})
	}
}

func TestLerp(t *testing.T) {
	if got := Lerp(0, 10, 0.5); got != 5 {
		t.Errorf("Lerp(0, 10, 0.5) = %v, want 5", got)
	}
	if got := Lerp(2, 4, 0); got != 2 {
		t.Errorf("Lerp(2, 4, 0) = %v, want 2", got)
	}
	if got := Lerp(0, 10, 2); got != 20 {
		t.Errorf("Lerp(0, 10, 2) = %v, want 20 - extrapolated", got)
	}

	// Composition is the eased-lerp pattern: shaping the weight
	// shapes the interpolation, with no intermediate helper.
	if got := Lerp(0, 10, EaseOutQuad(0.5)); got != 7.5 {
		t.Errorf("Lerp with EaseOutQuad(0.5) = %v, want 7.5", got)
	}
	if got := (Vector2{X: 0, Y: 0}).Lerp(Vector2{X: 10, Y: -10}, EaseInQuad(0.5)); got != (Vector2{X: 2.5, Y: -2.5}) {
		t.Errorf("Vector2.Lerp with EaseInQuad(0.5) = %v, want (2.5, -2.5)", got)
	}
}
