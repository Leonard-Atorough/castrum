package gametime

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func newScheduler() (*Scheduler, *core.World) {
	world := core.NewWorld()
	return NewScheduler(world), world
}

func newSchedulerAtTick(tick uint64) (*Scheduler, *core.World) {
	world := core.NewWorld()
	for world.Tick() < tick {
		world.AdvanceTick()
	}
	return NewScheduler(world), world
}

func noop() core.System {
	return core.SystemFunc(func(ctx *core.Context) error { return nil })
}

func record(log *[]string, name string) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		*log = append(*log, name)
		return nil
	})
}

func advance(world *core.World, ticks uint64) {
	for range ticks {
		world.AdvanceTick()
	}
}

func run(s *Scheduler, world *core.World) error {
	return NewSchedulerSystem(s).Update(&core.Context{World: world, Tick: world.Tick()})
}

var errBoom = errors.New("boom")

func TestValidation(t *testing.T) {
	type method struct {
		name string
		fn   func(s *Scheduler, name string, sys core.System) error
	}
	methods := []method{
		{"Once", func(s *Scheduler, name string, sys core.System) error { return s.Once(name, 1, sys) }},
		{"Every", func(s *Scheduler, name string, sys core.System) error { return s.Every(name, 1, sys) }},
		{"AtTick", func(s *Scheduler, name string, sys core.System) error { return s.AtTick(name, 1, sys) }},
	}

	for i, m := range methods {
		other := methods[(i+1)%len(methods)]
		cases := []struct {
			desc  string
			setup func(*Scheduler) error
			call  func(*Scheduler) error
			want  string
		}{
			{
				desc: "empty name",
				call: func(s *Scheduler) error { return m.fn(s, "", noop()) },
				want: "name must not be empty",
			},
			{
				desc: "nil system",
				call: func(s *Scheduler) error { return m.fn(s, "x", nil) },
				want: "system must not be nil",
			},
			{
				desc:  "duplicate name",
				setup: func(s *Scheduler) error { return m.fn(s, "x", noop()) },
				call:  func(s *Scheduler) error { return m.fn(s, "x", noop()) },
				want:  "duplicate entry name",
			},
			{
				desc:  "duplicate name across methods",
				setup: func(s *Scheduler) error { return other.fn(s, "x", noop()) },
				call:  func(s *Scheduler) error { return m.fn(s, "x", noop()) },
				want:  "duplicate entry name",
			},
		}
		for _, tc := range cases {
			t.Run(m.name+" "+tc.desc, func(t *testing.T) {
				s, _ := newScheduler()
				if tc.setup != nil {
					if err := tc.setup(s); err != nil {
						t.Fatalf("setup: %v", err)
					}
				}
				err := tc.call(s)
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.want)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error %q does not contain %q", err, tc.want)
				}
			})
		}
	}

	specific := []struct {
		desc string
		tick uint64
		call func(*Scheduler) error
		want string // empty means the call must succeed
	}{
		{"Once max delay from tick 1", 1,
			func(s *Scheduler) error { return s.Once("x", math.MaxUint64-1, noop()) }, ""},
		{"Once delay larger than remaining ticks", 1,
			func(s *Scheduler) error { return s.Once("x", math.MaxUint64, noop()) }, "delay too large"},
		{"Every zero interval", 0,
			func(s *Scheduler) error { return s.Every("x", 0, noop()) }, "interval must be positive"},
		{"Every max interval from tick 1", 1,
			func(s *Scheduler) error { return s.Every("x", math.MaxUint64-1, noop()) }, ""},
		{"Every interval larger than remaining ticks", 1,
			func(s *Scheduler) error { return s.Every("x", math.MaxUint64, noop()) }, "interval too large"},
		{"AtTick one tick in the past", 1,
			func(s *Scheduler) error { return s.AtTick("x", 0, noop()) }, "tick is in the past"},
		{"AtTick at the current tick", 1,
			func(s *Scheduler) error { return s.AtTick("x", 1, noop()) }, ""},
	}
	for _, tc := range specific {
		t.Run(tc.desc, func(t *testing.T) {
			s, _ := newSchedulerAtTick(tc.tick)
			err := tc.call(s)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				if _, ok := s.NextDue("x"); !ok {
					t.Fatal("entry not registered")
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestNextDue(t *testing.T) {
	const defaultTick = 1

	cases := []struct {
		desc     string
		name     string
		wantTick uint64
		wantOk   bool
	}{
		{"known name", "known name", defaultTick, true},
		{"earlier entry reports its own due", "early name", 0, true},
		{"unknown name", "unknown name", 0, false},
		{"empty name", "", 0, false},
	}
	s, _ := newScheduler()
	err := s.Once("known name", defaultTick, noop())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Earlier due than "known name": proves NextDue reports the named entry's
	// own due tick, not the scheduler's soonest.
	err = s.Once("early name", 0, noop())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			gotTick, gotOk := s.NextDue(tc.name)
			if gotTick != tc.wantTick || gotOk != tc.wantOk {
				t.Fatalf("NextDue(%q) = (%v, %v), want (%v, %v)", tc.name, gotTick, gotOk, tc.wantTick, tc.wantOk)
			}
		})
	}
}

func TestCancel(t *testing.T) {
	s, _ := newScheduler()
	err := s.Once("x", 1, noop())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := s.Once("sibling", 3, noop()); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := s.Cancel("x"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if _, ok := s.NextDue("x"); ok {
		t.Fatal("NextDue after cancel should not report the entry")
	}
	if _, ok := s.NextDue("sibling"); !ok {
		t.Fatal("cancelling one entry removed a sibling")
	}
	// The cancelled name is free for re-registration.
	err = s.Once("x", 5, noop())
	if err != nil {
		t.Fatalf("re-register cancelled name: %v", err)
	}
	if err := s.Cancel("x"); err != nil {
		t.Fatalf("cleanup Cancel: %v", err)
	}
	if err := s.Cancel("x"); err == nil {
		t.Fatal("expected error when cancelling unknown entry")
	}
}

func TestExecuteOnce(t *testing.T) {
	t.Run("waits until due, runs once, unregisters", func(t *testing.T) {
		s, w := newScheduler()
		var log []string
		// A run with no entries scheduled yet is a no-op.
		if err := run(s, w); err != nil {
			t.Fatalf("run on empty scheduler: %v", err)
		}
		if err := s.AtTick("x", 2, record(&log, "x")); err != nil {
			t.Fatalf("setup: %v", err)
		}
		// Ticks 0 and 1: not due yet.
		for tick := uint64(0); tick < 2; tick++ {
			if err := run(s, w); err != nil {
				t.Fatalf("run at tick %d: %v", tick, err)
			}
			if len(log) != 0 {
				t.Fatalf("entry ran at tick %d, not due until 2", tick)
			}
			advance(w, 1)
		}
		// Tick 2: dueTick <= current tick is inclusive, so it runs exactly
		// once and is removed from the registry.
		if err := run(s, w); err != nil {
			t.Fatalf("run at tick 2: %v", err)
		}
		if len(log) != 1 {
			t.Fatalf("entry ran %d times at tick 2, want 1", len(log))
		}
		if _, ok := s.NextDue("x"); ok {
			t.Fatal("entry still registered after execution")
		}
		if err := run(s, w); err != nil {
			t.Fatalf("second run at tick 2: %v", err)
		}
		if len(log) != 1 {
			t.Fatal("one-time entry ran again on the same tick")
		}
	})

	t.Run("due now runs immediately", func(t *testing.T) {
		s, w := newScheduler()
		var log []string
		if err := s.AtTick("x", 0, record(&log, "x")); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := run(s, w); err != nil {
			t.Fatalf("run: %v", err)
		}
		if len(log) != 1 {
			t.Fatalf("entry ran %d times, want 1: the due check is inclusive", len(log))
		}
	})
}

func TestExecuteOrder(t *testing.T) {
	s, w := newScheduler()
	var log []string
	entries := []struct {
		name string
		due  uint64
	}{
		{"a", 5}, {"b", 3}, {"c", 5}, {"d", 4},
	}
	for _, e := range entries {
		if err := s.AtTick(e.name, e.due, record(&log, e.name)); err != nil {
			t.Fatalf("setup %s: %v", e.name, err)
		}
	}
	// Removing a mid-entry also exercises heap.Remove: the surviving entries
	// must still execute in the right order.
	if err := s.Cancel("d"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	advance(w, 5)
	if err := run(s, w); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Ascending due ticks; the tie at tick 5 follows registration order.
	want := []string{"b", "a", "c"}
	if !slices.Equal(log, want) {
		t.Fatalf("execution order = %v, want %v", log, want)
	}
	if _, ok := s.NextDue("d"); ok {
		t.Fatal("cancelled entry executed or remained registered")
	}
}

func TestExecuteEvery(t *testing.T) {
	s, w := newScheduler()
	var log []string
	if err := s.Every("beat", 3, record(&log, "beat")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	advance(w, 3)
	if err := run(s, w); err != nil {
		t.Fatalf("run at tick 3: %v", err)
	}
	if len(log) != 1 {
		t.Fatalf("entry ran %d times at tick 3, want 1", len(log))
	}
	due, ok := s.NextDue("beat")
	if !ok || due != 6 {
		t.Fatalf("NextDue after run = (%d, %v), want (6, true)", due, ok)
	}

	advance(w, 2) // tick 5: before the next due tick
	if err := run(s, w); err != nil {
		t.Fatalf("run at tick 5: %v", err)
	}
	if len(log) != 1 {
		t.Fatal("entry ran before its next due tick")
	}

	advance(w, 1) // tick 6
	if err := run(s, w); err != nil {
		t.Fatalf("run at tick 6: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("entry ran %d times, want 2", len(log))
	}
	if due, ok := s.NextDue("beat"); !ok || due != 9 {
		t.Fatalf("NextDue after second run = (%d, %v), want (9, true)", due, ok)
	}
}

func TestExecuteError(t *testing.T) {
	s, w := newScheduler()
	var log []string
	boom := core.SystemFunc(func(ctx *core.Context) error { return errBoom })
	if err := s.AtTick("boom", 2, boom); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := s.AtTick("after", 2, record(&log, "after")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	advance(w, 2)
	err := run(s, w)
	if !errors.Is(err, errBoom) {
		t.Fatalf("run error = %v, want joined error wrapping errBoom", err)
	}
	// A failure stops only the failing entry: the rest of the batch still
	// runs this tick, and the failed entry stays registered to retry.
	want := []string{"after"}
	if !slices.Equal(log, want) {
		t.Fatalf("execution log = %v, want %v", log, want)
	}
	if _, ok := s.NextDue("boom"); !ok {
		t.Fatal("failing entry dropped from the registry")
	}
	if _, ok := s.NextDue("after"); ok {
		t.Fatal("successful one-time entry stayed registered")
	}

	// The failed entry retries at its original due tick and fails again.
	advance(w, 1)
	if err := run(s, w); err == nil {
		t.Fatal("expected the failing entry to error again on retry")
	}
}

func TestExecuteCancelInterplay(t *testing.T) {
	t.Run("cancelled while collected", func(t *testing.T) {
		s, w := newScheduler()
		var log []string
		first := core.SystemFunc(func(ctx *core.Context) error {
			log = append(log, "first")
			return s.Cancel("second")
		})
		if err := s.AtTick("first", 1, first); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := s.AtTick("second", 1, record(&log, "second")); err != nil {
			t.Fatalf("setup: %v", err)
		}

		advance(w, 1)
		if err := run(s, w); err != nil {
			t.Fatalf("run: %v", err)
		}
		// Both entries were collected for this tick; "second" was cancelled
		// before its turn, so the guard before execution must skip it.
		want := []string{"first"}
		if !slices.Equal(log, want) {
			t.Fatalf("execution log = %v, want %v", log, want)
		}
		if _, ok := s.NextDue("second"); ok {
			t.Fatal("cancelled entry still registered")
		}
	})

	t.Run("cancelled by its own action", func(t *testing.T) {
		s, w := newScheduler()
		var log []string
		self := core.SystemFunc(func(ctx *core.Context) error {
			log = append(log, "self")
			return s.Cancel("self")
		})
		if err := s.Every("self", 1, self); err != nil {
			t.Fatalf("setup: %v", err)
		}

		advance(w, 1)
		if err := run(s, w); err != nil {
			t.Fatalf("run: %v", err)
		}
		if len(log) != 1 {
			t.Fatalf("entry ran %d times, want 1", len(log))
		}
		// Cancelling during its own execution must prevent the reschedule.
		if _, ok := s.NextDue("self"); ok {
			t.Fatal("self-cancelled entry was re-registered")
		}
	})
}

func TestExecuteEveryOverflowDrops(t *testing.T) {
	s, w := newScheduler()
	var log []string
	// Interval MaxUint64 from tick 0 is accepted; the due tick lands on
	// MaxUint64 and the next reschedule would overflow the tick space.
	if err := s.Every("beat", math.MaxUint64, record(&log, "beat")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := NewSchedulerSystem(s).Update(&core.Context{World: w, Tick: math.MaxUint64})
	if err != nil {
		t.Fatalf("run at max tick: %v", err)
	}
	if len(log) != 1 {
		t.Fatalf("entry ran %d times, want 1", len(log))
	}
	if _, ok := s.NextDue("beat"); ok {
		t.Fatal("overflowing reschedule was re-registered")
	}
}
