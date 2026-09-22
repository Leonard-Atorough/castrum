package runtime

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// systemsInitialCapacity defines the initial capacity for the systems slice in a schedule.
const systemsInitialCapacity = 10

// Schedule represents a collection of systems to be executed in a specific order.
// T is typically a type that implements [core.System].
type Schedule[T any] struct {
	systems []T
	invoke  func(T, *core.Context) error
	phase   core.Phase
}

// NewSchedule creates a new schedule for the given phase and invocation function.
func NewSchedule[T any](phase core.Phase, invoke func(T, *core.Context) error) *Schedule[T] {
	return &Schedule[T]{
		systems: make([]T, 0, systemsInitialCapacity),
		invoke:  invoke,
		phase:   phase,
	}
}

// Add adds one or more systems to the schedule.
// Systems are appended to the existing list of systems in the schedule.
func (s *Schedule[T]) Add(systems ...T) {
	s.systems = append(s.systems, systems...)
}

// Run executes all systems in the schedule in the order they were added.
// If any system returns an error, Run stops execution and returns that error.
func (s *Schedule[T]) Run(ctx *core.Context) error {
	for _, system := range s.systems {
		if err := s.invoke(system, ctx); err != nil {
			return fmt.Errorf("system %T execution failed at phase %v: %w", system, s.phase, err)
		}
	}
	return nil
}
