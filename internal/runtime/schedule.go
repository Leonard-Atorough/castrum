package runtime

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// systemsInitialCapacity defines the initial capacity for the systems slice in a schedule.
const systemsInitialCapacity = 10

// Entry pairs a system with its registration name. The name identifies
// the registration in error messages and serves as the handle for
// future ordering constraints.
type Entry[T any] struct {
	Name   string
	System T
}

// Schedule represents a collection of systems to be executed in a specific order.
// T is typically a type that implements [core.System].
type Schedule[T any] struct {
	systems []Entry[T]
	invoke  func(T, *core.Context) error
	phase   core.Phase
}

// NewSchedule creates a new schedule for the given phase and invocation function.
func NewSchedule[T any](phase core.Phase, invoke func(T, *core.Context) error) *Schedule[T] {
	return &Schedule[T]{
		systems: make([]Entry[T], 0, systemsInitialCapacity),
		invoke:  invoke,
		phase:   phase,
	}
}

// Add adds entries to the schedule. Entries are appended to the
// existing list of systems in the schedule.
func (s *Schedule[T]) Add(entries ...Entry[T]) {
	s.systems = append(s.systems, entries...)
}

// Run executes all systems in the schedule in the order they were added.
// If any system returns an error, Run stops execution and returns that error.
func (s *Schedule[T]) Run(ctx *core.Context) error {
	for _, e := range s.systems {
		if err := s.invoke(e.System, ctx); err != nil {
			return fmt.Errorf("%s system %q: %w", s.phase, e.Name, err)
		}
	}
	return nil
}
