// Package tween provides caller-advanced [Tween] playheads for duration-based
// transitions.
package tween

import (
	"fmt"
	"time"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Tween tracks elapsed time and applies an easing curve to normalized progress.
// Pass its progress to [geom.Lerp] or [geom.Vector2.Lerp] to interpolate values.
// The caller chooses the clock by supplying elapsed time to [Tween.Update].
//
// Copying a Tween copies its playback state. Copies share the curve and any
// state it captures; the curve should be pure so repeated reads agree.
//
// The zero value is uninitialized: Update returns an error, Progress returns 0,
// IsComplete returns false, and Reset leaves it uninitialized.
type Tween struct {
	duration time.Duration
	elapsed  time.Duration
	curve    geom.Ease
}

// NewTween returns a Tween with the given duration and easing curve.
// It returns an error if duration is non-positive. A nil curve means [geom.EaseLinear].
func NewTween(duration time.Duration, ease geom.Ease) (Tween, error) {
	if ease == nil {
		ease = geom.EaseLinear
	}
	if duration <= 0 {
		return Tween{}, fmt.Errorf("tween: duration must be greater than zero")
	}
	return Tween{
		duration: duration,
		curve:    ease,
	}, nil
}

// Update advances the tween by delta and returns [Tween.Progress].
// Advancement stops at the duration and discards excess time. A negative delta
// returns an error without changing the tween.
func (t *Tween) Update(delta time.Duration) (float64, error) {
	if !t.isInitialized() {
		return 0, fmt.Errorf("tween: not initialized")
	}
	if delta < 0 {
		return 0, fmt.Errorf("tween: delta must be non-negative")
	}

	if delta >= t.duration-t.elapsed {
		t.elapsed = t.duration
	} else {
		t.elapsed += delta
	}
	return t.Progress(), nil
}

// Progress returns the current eased progress of the tween without advancing it.
// It evaluates the curve at elapsed time divided by duration. The result is
// not clamped, including at completion.
func (t *Tween) Progress() float64 {
	if !t.isInitialized() {
		return 0
	}
	progress := float64(t.elapsed) / float64(t.duration)
	return t.curve(progress)
}

// IsComplete reports whether the tween has reached its duration.
// Completion remains true until [Tween.Reset], regardless of the curve's output.
func (t *Tween) IsComplete() bool {
	if !t.isInitialized() {
		return false
	}
	return t.elapsed >= t.duration
}

// Reset restarts the tween without changing its duration or curve.
func (t *Tween) Reset() {
	t.elapsed = 0
}

func (t *Tween) isInitialized() bool {
	return t.duration > 0 && t.curve != nil
}
