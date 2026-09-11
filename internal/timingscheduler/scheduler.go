// Package timingscheduler owns simulation time: fixed ticks, phase order,
// pause. Internal by design — castrum's Ebiten shim plugs into it; game
// code never touches it directly.
package timingscheduler

import "time"

const (
	TicksPerSecond = 60
	TickDuration   = time.Second / TicksPerSecond
	maxDebt        = TickDuration * 5 // spiral-of-death guard
)

type Phase int

const (
	Input Phase = iota
	PreUpdate
	FixedUpdate
	PostUpdate
)

var phaseOrder = []Phase{Input, PreUpdate, FixedUpdate, PostUpdate}

// Runnable is anything a phase executes. Must be deterministic:
// no wall-clock reads, no map-iteration-order dependence.
type Runnable interface{ Run(dt time.Duration) }

type PauseBehavior int

const (
	PauseWithGame PauseBehavior = iota
	RunWhilePaused
)

type Pausable interface{ PauseBehavior() PauseBehavior }

// Scheduler advances real time in fixed ticks, executing phases in order.
type Scheduler struct {
	slots  map[Phase][]Runnable
	accum  time.Duration
	speed  float64
	paused bool
}

func New(speed float64) *Scheduler {
	return &Scheduler{
		slots: make(map[Phase][]Runnable),
		speed: speed,
	}
}

func (s *Scheduler) AddRunnable(phase Phase, r Runnable) {
	s.slots[phase] = append(s.slots[phase], r)
}

// Update consumes one frame of real time, running 0..N fixed ticks.
// realDt is measured by the caller (the only wall-clock read in the engine
// should be at this edge).
func (s *Scheduler) Update(realDt time.Duration) {
	s.accum += time.Duration(float64(realDt) * s.speed)
	if s.accum > maxDebt {
		s.accum = maxDebt
	}
	for s.accum >= TickDuration {
		s.accum -= TickDuration
		s.runTick()
	}
}

// Remainder is the leftover sub-tick time — the interpolation fraction,
// for future interpolated rendering. Not used today.
func (s *Scheduler) Remainder() time.Duration { return s.accum }

func (s *Scheduler) SetTimeScale(timeScale float64) {
	s.speed = timeScale
}

func (s *Scheduler) SetPaused(paused bool) {
	s.paused = paused
}

func (s *Scheduler) runTick() {
	for _, phase := range phaseOrder {
		for _, r := range s.slots[phase] {
			if shouldRun(r, s.paused) {
				r.Run(TickDuration)
			}
		}
	}
}

func shouldRun(r Runnable, paused bool) bool {
	p, ok := r.(Pausable)
	if !ok {
		return true
	}
	switch p.PauseBehavior() {
	case RunWhilePaused:
		return true
	default:
		return !paused
	}

}
