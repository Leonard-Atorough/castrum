package gametime

import (
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/engine"
)

// Wiring returns the engine registration for the time subsystem: the
// [Simulation], [Real], [Groups], and [Scheduler] resources, the scheduler
// and timer systems, and the package's blueprint entries. The root package
// applies it with the other registrations when building a game.
func Wiring(fixedDT, maxFrameTime time.Duration) engine.Registration {
	var scheduler *Scheduler
	return engine.Registration{
		Name: "gametime",
		Resources: []func(*core.World) error{
			func(w *core.World) error {
				sim := NewSimulation(fixedDT)
				return w.ProvideEager(func(*core.World) (*Simulation, error) { return sim, nil })
			},
			func(w *core.World) error {
				real := NewReal(maxFrameTime)
				return w.ProvideEager(func(*core.World) (*Real, error) { return real, nil })
			},
			func(w *core.World) error {
				return w.Provide(func(*core.World) (*Groups, error) { return NewGroups(), nil })
			},
			func(w *core.World) error {
				scheduler = NewScheduler(w)
				return w.Provide(func(*core.World) (*Scheduler, error) { return scheduler, nil })
			},
		},
		Systems: []engine.SystemSpec{
			{Name: SchedulerSystemName, Phase: core.PhasePreFixed, Slot: engine.First, New: func() core.System { return NewSchedulerSystem(scheduler) }},
			{Name: TimerSystemName, Phase: core.PhaseFixed, Slot: engine.First, New: func() core.System { return NewTimerSystem() }},
		},
		Blueprints: RegisterBlueprints,
	}
}
