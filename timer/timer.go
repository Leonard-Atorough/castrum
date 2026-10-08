// Package timer provides countdown timers as component state: the
// [Timer] component, and the reconciler from [NewSystem] that
// advances running timers each fixed tick and records completions.
// Nothing is emitted and nothing is removed - a game reads
// [Timer.CompletedOn] to observe completions.
package timer

import (
	"fmt"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

// Timer tracks a countdown as component state. Create one with
// [NewTimer]; the reconciler from [NewSystem] advances it on each
// fixed tick.
//
// A one-shot stops when it completes but remains on its entity.
// [Timer.CompletedOn] records the completion tick until
// [Timer.Restart] clears it; use [Timer.JustCompleted] to
// detect the completion for a single tick and [Timer.HasCompleted]
// to check whether it has finished. A repeating timer starts another
// interval immediately and carries any overshoot forward.
//
// Each entity can hold only one [Timer], because component storage
// has one slot per component type. To track several timers for one
// actor, give each timer its own entity.
type Timer struct {
	// Duration is how long the timer runs before completing. Must
	// be positive; [Timer.Validate] rejects anything else.
	Duration time.Duration
	// Elapsed is the time accumulated since the timer last started.
	// The reconciler advances it; [Timer.Restart] resets it.
	Elapsed time.Duration
	// Repeating reports whether the timer restarts itself after
	// completing, keeping the overshoot, instead of stopping.
	Repeating bool
	// Running reports whether the timer is accumulating time.
	// [NewTimer] starts it running; pause and resume by writing
	// Running inside an update closure.
	Running bool
	// CompletedOn is the fixed tick of the timer's last completion,
	// zero when it has never completed. Nothing clears it on its
	// own: compare against the current tick for the
	// once-per-completion edge, or leave it set to remember that a
	// one-shot fired.
	CompletedOn uint64
}

// NewTimer returns a running Timer for duration. A repeating timer
// restarts itself after each completion; otherwise the timer
// completes once and stops. The duration is validated when the
// component enters storage, not here - spawn rejects a non-positive
// one with [Timer.Validate].
func NewTimer(duration time.Duration, repeating bool) Timer {
	return Timer{
		Duration:  duration,
		Repeating: repeating,
		Running:   true,
	}
}

// Validate checks that the timer is spawnable: a positive duration
// and non-negative elapsed time. It runs whenever the component
// enters storage, so a timer mutated through an update closure
// cannot hold an invalid duration.
func (t Timer) Validate() error {
	if t.Duration <= 0 {
		return fmt.Errorf("timer duration must be positive, got %v", t.Duration)
	}
	if t.Elapsed < 0 {
		return fmt.Errorf("timer elapsed time must not be negative, got %v", t.Elapsed)
	}
	return nil
}

// Restart resets the timer to its beginning: elapsed time resets,
// the completion stamp clears, and the timer runs again. On a
// completed one-shot this also resets the "has it fired" answer.
// Call it inside an update closure - the pointer receiver mutates
// the copy the closure writes back.
func (t *Timer) Restart() {
	t.Elapsed = 0
	t.Running = true
	t.CompletedOn = 0
}

// JustCompleted reports whether the timer completed on the
// given fixed tick - the once-per-completion edge. Pass the context
// tick: the reconciler runs earlier in the same fixed phase, so the
// completion reads true for exactly one tick.
func (t Timer) JustCompleted(tick uint64) bool {
	return t.CompletedOn == tick
}

// HasCompleted reports whether a one-shot timer has completed and
// now holds its finished state - the record that stays until
// Restart clears it. A repeating timer never completes: it fires
// and starts its next interval, so for a repeating timer this read
// is always false and each fire is observed with
// [Timer.JustCompleted] instead. The tick counter starts at one,
// so a stamp of zero always means "never completed".
func (t Timer) HasCompleted() bool {
	return !t.Repeating && t.CompletedOn != 0
}

// NewSystem returns the timer reconciler: every fixed tick it
// advances each running [Timer] by the tick interval and stamps
// [Timer.CompletedOn] when the duration has fully elapsed. A
// one-shot stops and stays on its entity; a repeating timer
// restarts itself, keeping the overshoot so its intervals do not
// drift. Nothing is emitted and no entity is removed - completion
// is component state a game reads.
func NewSystem() core.System {
	return &system{}
}

type system struct {
	timers *core.Query
}

func (s *system) Update(ctx *core.Context) error {
	if s.timers == nil {
		s.timers = core.NewQuery(ctx.World).With(Timer{})
	}

	for e := range s.timers.Execute() {
		e.Update(func(t *Timer) {
			if t.HasCompleted() {
				t.Running = false
				return
			}

			if t.Running {
				t.Elapsed += ctx.DeltaTime
				for t.Elapsed >= t.Duration {
					t.CompletedOn = ctx.Tick

					if t.Repeating {
						t.Elapsed %= t.Duration
					} else {
						t.Elapsed = t.Duration
						t.Running = false
						break
					}
				}
			}
		})
	}
	return nil
}
