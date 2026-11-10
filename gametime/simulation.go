package gametime

import (
	"fmt"
	"math"
	"time"
)

// Simulation holds the engine's fixed-step time state.
//
// The engine updates Tick as fixed updates run. The fixed-loop accumulator
// is not part of this resource, so copying it does not preserve the
// accumulator's position.
type Simulation struct {
	// Tick counts fixed updates since the game started.
	Tick uint64
	// Delta is the fixed time interval per tick, independent of Scale.
	Delta time.Duration
	// Paused stops fixed updates and preserves the accumulator; frame updates
	// continue.
	Paused bool
	// Scale multiplies elapsed time accumulated toward fixed ticks. It must
	// be finite and positive; use [Simulation.Pause] to stop.
	Scale float64
}

// NewSimulation returns a running simulation with the given fixed tick
// interval, at scale 1, starting at tick 0.
func NewSimulation(delta time.Duration) *Simulation {
	return &Simulation{Delta: delta, Scale: 1}
}

// Elapsed returns Tick multiplied by the current Delta.
func (s *Simulation) Elapsed() time.Duration {
	return time.Duration(s.Tick) * s.Delta
}

// Pause stops fixed updates on the next advance.
func (s *Simulation) Pause() {
	s.Paused = true
}

// Resume allows fixed updates to continue on the next advance.
func (s *Simulation) Resume() {
	s.Paused = false
}

// SetScale sets [Simulation.Scale], returning an error without changing it
// if scale is invalid.
func (s *Simulation) SetScale(scale float64) error {
	if math.IsNaN(scale) {
		return fmt.Errorf("simulation scale must not be NaN")
	}
	if math.IsInf(scale, 0) {
		return fmt.Errorf("simulation scale must be finite, got %v", scale)
	}
	if scale <= 0 {
		return fmt.Errorf("simulation scale %v, want > 0; use Pause to stop the simulation", scale)
	}
	s.Scale = scale
	return nil
}
