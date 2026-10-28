// Package runtime provides ordered execution of systems for a core phase.
package runtime

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

const systemsInitialCapacity = 10

// Entry associates a system with its registration name.
type Entry[T any] struct {
	// Name identifies the registration in execution errors.
	Name string
	// System is the value passed to the schedule's invocation function.
	System T
	// Group determines the execution group for the system.
	Group string
}

// Schedule runs registered systems in order for one [core.Phase].
type Schedule[T any] struct {
	// systems holds entries in execution order.
	systems []Entry[T]
	// disabled tracks groups that are currently disabled.
	disabled map[string]bool
	// invoke executes one system with the current context.
	invoke func(T, *core.Context) error
	// phase labels execution errors.
	phase core.Phase
}

// NewSchedule returns an empty schedule for phase. invoke executes each system
// with the context passed to [Schedule.Run].
func NewSchedule[T any](phase core.Phase, invoke func(T, *core.Context) error) *Schedule[T] {
	return &Schedule[T]{
		systems:  make([]Entry[T], 0, systemsInitialCapacity),
		disabled: make(map[string]bool),
		invoke:   invoke,
		phase:    phase,
	}
}

// Add appends entries to the schedule in the order provided.
func (s *Schedule[T]) Add(entries ...Entry[T]) {
	s.systems = append(s.systems, entries...)
}

// Run executes each system in registration order. It stops at the first error
// and returns it with the schedule phase and entry name.
func (s *Schedule[T]) Run(ctx *core.Context) error {
	for _, e := range s.systems {
		if e.Group != "" && s.disabled[e.Group] {
			continue
		}
		if err := s.invoke(e.System, ctx); err != nil {
			return fmt.Errorf("%s system %q: %w", s.phase, e.Name, err)
		}
	}
	return nil
}

func (s *Schedule[T]) SetGroupEnabled(group string, enabled bool) {
	if s.disabled == nil {
		s.disabled = make(map[string]bool)
	}
	s.disabled[group] = !enabled
}

func (s *Schedule[T]) Registered(group string) bool {
	for _, e := range s.systems {
		if e.Group == group {
			return true
		}
	}
	return false
}
