package timingscheduler

import "time"

const (
	TicksPerSecond = 60
	TickDuration   = time.Second / TicksPerSecond
	maxDebt        = TickDuration * 5
)

type Phase int

const (
	Input Phase = iota
	Update
	Render
)

var phaseOrder = []Phase{Input, Update, Render}

type Runnable interface{ Run(dt time.Duration) }

type PauseBehavior int

const (
	PauseWithGame PauseBehavior = iota
	RunWhilePaused
)

type Pausable interface{ PauseBehavior() PauseBehavior }

type Scheduler struct {
	slots  map[Phase][]Runnable
	accum  time.Duration
	paused bool
}

func New() *Scheduler {
	return &Scheduler{
		slots:  make(map[Phase][]Runnable),
		accum:  0,
		paused: false,
	}
}

func (s *Scheduler) AddRunnable(phase Phase, r Runnable) {
	s.slots[phase] = append(s.slots[phase], r)
}

func (s *Scheduler) SetPaused(paused bool) {
	s.paused = paused
}

func (s *Scheduler) Update(realDt time.Duration) {
	s.accum += realDt
	for s.accum >= TickDuration {
		s.runTick()
		s.accum -= TickDuration
	}
	if s.accum > maxDebt {
		s.accum = maxDebt
	}
}

func (s *Scheduler) runTick() {
	for _, phase := range phaseOrder {
		for _, r := range s.slots[phase] {
			if !shouldRun(r, s.paused) {
				continue
			}
			r.Run(TickDuration)
		}
	}
}

func shouldRun(r Runnable, paused bool) bool {
	if p, ok := r.(Pausable); ok {
		switch p.PauseBehavior() {
		case PauseWithGame:
			return !paused
		case RunWhilePaused:
			return true
		}
	}
	return !paused
}
