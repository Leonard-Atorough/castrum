package tween

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/geom"
)

func checkUninitialized(t *testing.T, tween *Tween) {
	t.Helper()
	if got := tween.Progress(); got != 0 {
		t.Errorf("Progress() = %v, want 0", got)
	}
	if tween.IsComplete() {
		t.Error("IsComplete() = true, want false")
	}
	if _, err := tween.Update(0); err == nil {
		t.Error("Update() succeeded on an uninitialized tween")
	}
}

func TestNewTween(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		curve    geom.Ease
		want     float64
		wantErr  bool
	}{
		{name: "zero duration", wantErr: true},
		{name: "negative duration", duration: -time.Second, wantErr: true},
		{name: "nil curve is linear", duration: time.Second, want: 0.5},
		{name: "quadratic curve", duration: time.Second, curve: geom.EaseInQuad, want: 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(tt.duration, tt.curve)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewTween() succeeded, want an error")
				}
				checkUninitialized(t, &tween)
				return
			}
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			if got := tween.Progress(); got != 0 {
				t.Errorf("initial Progress() = %v, want 0", got)
			}
			if tween.IsComplete() {
				t.Error("new tween is complete")
			}
			got, err := tween.Update(time.Second / 2)
			if err != nil {
				t.Fatalf("Update() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Update() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTweenUpdate(t *testing.T) {
	t.Parallel()

	tween, err := NewTween(time.Second, geom.EaseInQuad)
	if err != nil {
		t.Fatalf("NewTween() error: %v", err)
	}
	steps := []struct {
		delta    time.Duration
		want     float64
		complete bool
	}{
		{delta: 0, want: 0},
		{delta: 250 * time.Millisecond, want: 0.0625},
		{delta: 250 * time.Millisecond, want: 0.25},
		{delta: 0, want: 0.25},
		{delta: 250 * time.Millisecond, want: 0.5625},
		{delta: 250 * time.Millisecond, want: 1, complete: true},
		{delta: 0, want: 1, complete: true},
		{delta: time.Second, want: 1, complete: true},
	}
	for i, step := range steps {
		got, err := tween.Update(step.delta)
		if err != nil {
			t.Fatalf("step %d: Update() error: %v", i, err)
		}
		if got != step.want {
			t.Errorf("step %d: Update() = %v, want %v", i, got, step.want)
		}
		if progress := tween.Progress(); progress != got {
			t.Errorf("step %d: Progress() = %v, Update() returned %v", i, progress, got)
		}
		if complete := tween.IsComplete(); complete != step.complete {
			t.Errorf("step %d: IsComplete() = %v, want %v", i, complete, step.complete)
		}
	}
}

func TestTweenUpdateSaturates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		first    time.Duration
		delta    time.Duration
	}{
		{"overshoot", time.Second, time.Second / 2, time.Second},
		{"large delta", time.Second, time.Second / 2, time.Duration(math.MaxInt64)},
		{"maximum duration", time.Duration(math.MaxInt64), 1, time.Duration(math.MaxInt64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(tt.duration, nil)
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			if _, err := tween.Update(tt.first); err != nil {
				t.Fatalf("first Update() error: %v", err)
			}
			got, err := tween.Update(tt.delta)
			if err != nil {
				t.Fatalf("Update() error: %v", err)
			}
			if got != 1 {
				t.Errorf("Update() = %v, want 1", got)
			}
			if got := tween.Progress(); got != 1 {
				t.Errorf("Progress() = %v, want 1", got)
			}
			if !tween.IsComplete() {
				t.Error("IsComplete() = false, want true")
			}
		})
	}
}

func TestTweenUpdateCustomCurve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		curve geom.Ease
		want  [3]float64
	}{
		{"shifted endpoints", func(t float64) float64 { return t + 0.5 }, [3]float64{0.5, 1, 1.5}},
		{"overshoot", func(t float64) float64 { return t + 4*t*(1-t) }, [3]float64{0, 1.5, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(time.Second, tt.curve)
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			for i, delta := range []time.Duration{0, time.Second / 2, time.Second / 2, 0, time.Second} {
				got, err := tween.Update(delta)
				if err != nil {
					t.Fatalf("step %d: Update() error: %v", i, err)
				}
				want := tt.want[min(i, 2)]
				if got != want {
					t.Errorf("step %d: Update() = %v, want %v", i, got, want)
				}
				if progress := tween.Progress(); progress != got {
					t.Errorf("step %d: Progress() = %v, Update() returned %v", i, progress, got)
				}
				if complete := tween.IsComplete(); complete != (i >= 2) {
					t.Errorf("step %d: IsComplete() = %v, want %v", i, complete, i >= 2)
				}
			}
		})
	}
}

func TestTweenUpdateNegativeDelta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		elapsed time.Duration
	}{
		{"new", 0},
		{"in progress", time.Second / 2},
		{"complete", time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			if _, err := tween.Update(tt.elapsed); err != nil {
				t.Fatalf("initial Update() error: %v", err)
			}
			progress, complete := tween.Progress(), tween.IsComplete()
			if _, err := tween.Update(-time.Nanosecond); err == nil {
				t.Fatal("Update() succeeded, want an error")
			}
			if got := tween.Progress(); got != progress {
				t.Errorf("Progress() = %v, want unchanged %v", got, progress)
			}
			if got := tween.IsComplete(); got != complete {
				t.Errorf("IsComplete() = %v, want unchanged %v", got, complete)
			}
		})
	}
}

func TestTweenZeroValue(t *testing.T) {
	t.Parallel()

	var tween Tween
	checkUninitialized(t, &tween)
	tween.Reset()
	checkUninitialized(t, &tween)
	if _, err := tween.Update(time.Second); err == nil {
		t.Error("Update(time.Second) succeeded, want an error")
	}
	if got := tween.Progress(); got != 0 {
		t.Errorf("Progress() after Reset and rejected updates = %v, want 0", got)
	}
	if tween.IsComplete() {
		t.Error("zero-value tween became complete")
	}
}

func TestTweenReset(t *testing.T) {
	t.Parallel()

	for _, elapsed := range []time.Duration{time.Second / 2, time.Second, 2 * time.Second} {
		t.Run(elapsed.String(), func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			if _, err := tween.Update(elapsed); err != nil {
				t.Fatalf("Update() error: %v", err)
			}
			tween.Reset()
			if got := tween.Progress(); got != 0 {
				t.Errorf("Progress() after Reset() = %v, want 0", got)
			}
			if tween.IsComplete() {
				t.Error("IsComplete() after Reset() = true, want false")
			}
			got, err := tween.Update(time.Second / 2)
			if err != nil {
				t.Fatalf("Update() after Reset() error: %v", err)
			}
			if got != 0.25 {
				t.Errorf("Update() after Reset() = %v, want 0.25", got)
			}
		})
	}
}

func TestTweenCopy(t *testing.T) {
	t.Parallel()

	first, err := NewTween(time.Second, geom.EaseInQuad)
	if err != nil {
		t.Fatalf("NewTween() error: %v", err)
	}
	if _, err := first.Update(time.Second / 2); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	second := first
	if _, err := second.Update(time.Second / 2); err != nil {
		t.Fatalf("copy Update() error: %v", err)
	}
	if got := first.Progress(); got != 0.25 {
		t.Errorf("original Progress() = %v, want 0.25", got)
	}
	if first.IsComplete() || !second.IsComplete() {
		t.Error("want unfinished original and completed copy")
	}
	second.Reset()
	if got := first.Progress(); got != 0.25 {
		t.Errorf("original Progress() after copy Reset() = %v, want 0.25", got)
	}
}

func TestTweenUpdatePartitioning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		deltas []time.Duration
	}{
		{"single", []time.Duration{750 * time.Millisecond}},
		{"equal steps", []time.Duration{250 * time.Millisecond, 250 * time.Millisecond, 250 * time.Millisecond}},
		{"unequal steps", []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 450 * time.Millisecond}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tween, err := NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("NewTween() error: %v", err)
			}
			for _, delta := range tt.deltas {
				if _, err := tween.Update(delta); err != nil {
					t.Fatalf("Update() error: %v", err)
				}
			}

			if got := tween.Progress(); got != 0.5625 {
				t.Errorf("Progress() = %v, want 0.5625", got)
			}
			if tween.IsComplete() {
				t.Error("IsComplete() = true, want false")
			}
		})
	}
}

func ExampleTween() {
	tween, err := NewTween(time.Second, geom.EaseInQuad)
	if err != nil {
		fmt.Println(err)
		return
	}
	weight, err := tween.Update(500 * time.Millisecond)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(geom.Lerp(0, 100, weight))
	fmt.Println(tween.IsComplete())
	// Output:
	// 25
	// false
}
