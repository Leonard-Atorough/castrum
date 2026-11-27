package gametime

import (
	"container/heap"
	"errors"
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
)

type entry struct {
	name     string
	dueTick  uint64
	seq      uint64
	action   core.SystemFunc
	interval uint64 // zero for one-time entries
}

type entryHeap []*entry

func (h entryHeap) Len() int { return len(h) }

func (h entryHeap) Less(i, j int) bool {
	if h[i].dueTick == h[j].dueTick {
		return h[i].seq < h[j].seq
	}
	return h[i].dueTick < h[j].dueTick
}

func (h entryHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *entryHeap) Push(x any) {
	*h = append(*h, (x.(*entry)))
}

func (h *entryHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Scheduler manages timed execution of systems within a world. It supports
// one-time and recurring entries, and allows querying and cancellation of
// scheduled entries.
//
// Entries are reconciled in the [core.PhasePreFixed] phase of the world's
// update cycle.
//
// Like the world, the Scheduler is not concurrency-safe and should be
// accessed from a single goroutine only.
type Scheduler struct {
	entries *entryHeap
	byName  map[string]*entry
	world   *core.World
	nextSeq uint64
}

// NewScheduler creates a new Scheduler associated with the given world.
// The world's tick will be used as the reference for scheduling entries.
func NewScheduler(world *core.World) *Scheduler {
	return &Scheduler{
		entries: &entryHeap{},
		byName:  make(map[string]*entry),
		world:   world,
	}
}

// Once schedules a one-time execution of the given system after the specified delay.
// The delay is measured in ticks relative to the world's current tick.
//
// Returns an error if the name is empty, the system is nil, or the delay is too large.
func (s *Scheduler) Once(name string, delay uint64, sys core.SystemFunc) error {
	if err := s.validate(name, sys, "scheduler.Once"); err != nil {
		return err
	}
	now := s.world.Tick()
	if delay > math.MaxUint64-now {
		return fmt.Errorf("gametime: scheduler.Once: delay too large")
	}
	s.insert(&entry{name: name, dueTick: now + delay, seq: s.nextSeq, action: sys})
	return nil
}

// Every schedules recurring execution of the given system at the specified interval.
// The interval is measured in ticks relative to the world's current tick.
//
// Returns an error if the name is empty, the system is nil, the interval is zero, or
// the interval is too large.
func (s *Scheduler) Every(name string, interval uint64, sys core.SystemFunc) error {
	if err := s.validate(name, sys, "scheduler.Every"); err != nil {
		return err
	}
	if interval == 0 {
		return fmt.Errorf("gametime: scheduler.Every: interval must be positive")
	}

	now := s.world.Tick()
	if interval > math.MaxUint64-now {
		return fmt.Errorf("gametime: scheduler.Every: interval too large")
	}
	s.insert(&entry{name: name, dueTick: now + interval, seq: s.nextSeq, action: sys, interval: interval})
	return nil
}

// AtTick schedules a one-time execution of the given system at the specified tick.
//
// Returns an error if the name is empty, the system is nil, or the tick is in the past.
func (s *Scheduler) AtTick(name string, tick uint64, sys core.SystemFunc) error {
	if err := s.validate(name, sys, "scheduler.AtTick"); err != nil {
		return err
	}
	now := s.world.Tick()
	if tick < now {
		return fmt.Errorf("gametime: scheduler.AtTick: tick is in the past")
	}
	s.insert(&entry{name: name, dueTick: tick, seq: s.nextSeq, action: sys})
	return nil
}

// Cancel removes the entry with the given name from the scheduler.
func (s *Scheduler) Cancel(name string) error {
	e, exists := s.byName[name]
	if !exists {
		return fmt.Errorf("gametime: scheduler Cancel: unknown entry %q", name)
	}

	for i, entry := range *s.entries {
		if entry == e {
			heap.Remove(s.entries, i)
			break
		}
	}
	delete(s.byName, name)
	return nil
}

// NextDue returns the due tick and existence status of the entry with the given name.
func (s *Scheduler) NextDue(name string) (uint64, bool) {
	e, ok := s.byName[name]
	if !ok {
		return 0, false
	}
	return e.dueTick, true
}

func (s *Scheduler) validate(name string, sys core.SystemFunc, source string) error {
	if name == "" {
		return fmt.Errorf("gametime: %s: name must not be empty", source)
	}
	if sys == nil {
		return fmt.Errorf("gametime: %s: system must not be nil", source)
	}
	if _, exists := s.byName[name]; exists {
		return fmt.Errorf("gametime: %s: duplicate entry name", source)
	}
	return nil
}

func (s *Scheduler) insert(e *entry) {
	s.nextSeq++
	e.seq = s.nextSeq
	heap.Push(s.entries, e)
	s.byName[e.name] = e
}

// SchedulerSystemName is the name used to register the scheduler system in
// the pre-fixed schedule.
const SchedulerSystemName = "engine.scheduler"

// System returns the system that executes due entries from the scheduler.
// Register it in [core.PhasePreFixed] as [SchedulerSystemName] so scheduled
// entries run before the fixed phase's systems.
//
// An entry whose action returns an error stays registered and retries on the
// next run; the rest of the batch still executes. All errors are joined and
// reported after the batch completes.
func (s *Scheduler) System() core.SystemFunc {
	return func(ctx *core.Context) error {
		now := ctx.Tick
		var due []*entry
		for s.entries.Len() > 0 && (*s.entries)[0].dueTick <= now {
			due = append(due, heap.Pop(s.entries).(*entry))
		}
		var errs []error
		for _, e := range due {
			if s.byName[e.name] != e {
				continue
			}
			if err := e.action(ctx); err != nil {

				errs = append(errs, fmt.Errorf("gametime: scheduler system: failed to execute entry %q: %w", e.name, err))
				if s.byName[e.name] == e {
					heap.Push(s.entries, e)
				}
				continue
			}
			if s.byName[e.name] != e {
				continue
			}
			if e.interval > 0 {
				if e.dueTick > math.MaxUint64-e.interval {
					if s.byName[e.name] == e {
						delete(s.byName, e.name)
					}
					continue
				}
				e.dueTick += e.interval
				heap.Push(s.entries, e)
			} else {
				if s.byName[e.name] == e {
					delete(s.byName, e.name)
				}
			}
		}
		return errors.Join(errs...)
	}
}
