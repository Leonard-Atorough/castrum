package runtime

import (
	"errors"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

type testSystem struct{}

func (t *testSystem) Update(ctx *core.Context) error {
	return nil
}

func TestScheduleAdd(t *testing.T) {
	t.Run("adds a system to the schedule", func(t *testing.T) {
		s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		systemAdded := false
		sys := core.SystemFunc(func(ctx *core.Context) error {
			systemAdded = true
			return nil
		})
		s.Add(Entry[core.System]{Name: "test", System: sys})
		if len(s.systems) != 1 {
			t.Errorf("expected 1 system in schedule, got %d", len(s.systems))
		}
		if err := s.Run(&core.Context{}); err != nil {
			t.Errorf("unexpected error running schedule: %v", err)
		}
		if !systemAdded {
			t.Error("the system was not executed")
		}
	})

	t.Run("adds multiple systems to the schedule", func(t *testing.T) {
		s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		system1Added := false
		system2Added := false
		sys1 := core.SystemFunc(func(ctx *core.Context) error {
			system1Added = true
			return nil
		})
		sys2 := core.SystemFunc(func(ctx *core.Context) error {
			system2Added = true
			return nil
		})
		s.Add(
			Entry[core.System]{Name: "first", System: sys1},
			Entry[core.System]{Name: "second", System: sys2},
		)
		if len(s.systems) != 2 {
			t.Errorf("expected 2 systems in schedule, got %d", len(s.systems))
		}
		if err := s.Run(&core.Context{}); err != nil {
			t.Errorf("unexpected error running schedule: %v", err)
		}
		if !system1Added {
			t.Error("the first system was not executed")
		}
		if !system2Added {
			t.Error("the second system was not executed")
		}
	})
}

func TestScheduleRunWithoutSystems(t *testing.T) {
	t.Run("runs schedule with no systems without error", func(t *testing.T) {
		s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		if err := s.Run(&core.Context{}); err != nil {
			t.Errorf("unexpected error running schedule with no systems: %v", err)
		}
	})
}

func TestScheduleRunWithError(t *testing.T) {
	t.Run("runs schedule and returns error from system", func(t *testing.T) {
		s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
			return sys.Update(ctx)
		})
		expectedErr := errors.New("system error")
		sys := core.SystemFunc(func(ctx *core.Context) error {
			return expectedErr
		})
		s.Add(Entry[core.System]{Name: "failing", System: sys})
		err := s.Run(&core.Context{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error %v, got %v", expectedErr, err)
		}
		if !strings.Contains(err.Error(), `"failing"`) {
			t.Errorf("error %q should name the failing system", err)
		}
		if !strings.Contains(err.Error(), "fixed") {
			t.Errorf("error %q should name the phase", err)
		}
	})
}
