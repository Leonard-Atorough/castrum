// Package timer provides countdowns as [Timer] components. [NewSystem]
// advances them on fixed ticks and records completions in
// [Timer.CompletedOn]; it emits no events and does not remove timers.
package timer

import (
	"fmt"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

// Timer tracks a countdown on an entity. [NewSystem] advances it on fixed
// ticks.
//
// An entity can hold only one [Timer]. Give each timer its own entity to track
// multiple timers for one actor.
type Timer struct {
	// Duration is the length of each interval and must be positive.
	Duration time.Duration
	// Elapsed is the time accumulated in the current interval.
	Elapsed time.Duration
	// Repeating starts a new interval after each completion and carries over
	// excess elapsed time. Otherwise, the timer stops at Duration.
	Repeating bool
	// Running controls whether elapsed time accumulates. Setting it false
	// pauses an unfinished timer; restart a completed one-shot with
	// [Timer.Restart].
	Running bool
	// CompletedOn is the fixed tick of the most recent completion. Zero means
	// no completion has been recorded because game ticks start at one.
	// [Timer.Restart] clears it.
	CompletedOn uint64
}

// NewTimer returns a running timer with the given duration and repeat setting.
// [Timer.Validate] rejects a non-positive duration when the timer enters
// storage.
func NewTimer(duration time.Duration, repeating bool) Timer {
	return Timer{
		Duration:  duration,
		Repeating: repeating,
		Running:   true,
	}
}

// Validate reports an error if Duration is non-positive or Elapsed is
// negative.
func (t Timer) Validate() error {
	if t.Duration <= 0 {
		return fmt.Errorf("timer duration must be positive, got %v", t.Duration)
	}
	if t.Elapsed < 0 {
		return fmt.Errorf("timer elapsed time must not be negative, got %v", t.Elapsed)
	}
	return nil
}

// Restart clears Elapsed and CompletedOn, then marks the timer as Running.
// To restart a timer stored on an entity, call it inside [core.Entity.Update].
func (t *Timer) Restart() {
	t.Elapsed = 0
	t.Running = true
	t.CompletedOn = 0
}

// JustCompleted reports whether the timer's most recent completion occurred
// on tick. Pass [core.Context.Tick] to detect a completion during the current
// fixed update.
func (t Timer) JustCompleted(tick uint64) bool {
	return t.CompletedOn == tick
}

// HasCompleted reports whether a one-shot timer has completed. It remains
// true until [Timer.Restart]. It always returns false for repeating timers;
// use [Timer.JustCompleted] to observe each completion.
func (t Timer) HasCompleted() bool {
	return !t.Repeating && t.CompletedOn != 0
}

// SystemName is the name used to register the timer system in the fixed
// schedule.
const SystemName = "engine.timer"

// NewSystem returns the fixed-tick system that advances running [Timer]
// components and records completions in [Timer.CompletedOn].
//
// The game constructor registers this system under [SystemName] when
// `castrum.WithTimer` is enabled. Otherwise, register it yourself.
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
