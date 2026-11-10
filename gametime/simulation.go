// This file holds the engine's time resource: [Simulation], the fixed loop's
// state as world data. The loop reads Paused and Scale on every advance and
// increments Tick on every fixed tick, so simulation time is readable by any
// system, savable with the world, and resettable in tests. There is exactly
// one clock in the engine; games derive game-specific time (day cycles,
// schedules) from Simulation.Tick in their own code.

package gametime

import (
	"fmt"
	"math"
	"time"
)

// Simulation is the engine's single time resource: the fixed loop's pause,
// scale, and tick state. The engine provisions it when the game is
// constructed and the loop advances it; systems read it through
// [core.World.Resource].
//
// All fields are exported and value-shaped: reading them snapshots the
// simulation's time state, and writing them to a fresh Simulation restores
// it. The loop's accumulator is runner state and is deliberately not part of
// the resource: it is an interpolation detail, not time itself.
type Simulation struct {
	// Tick counts fixed ticks since the game started. It stops when Paused
	// and slows or quickens with Scale, mirroring [core.Context.Tick] during
	// fixed updates.
	Tick uint64
	// Delta is the fixed tick interval: the simulation time advanced by one
	// fixed tick. It is constant per tick at every Scale - scale changes how
	// often ticks occur, never how long a tick is. It is set at construction
	// from the fixed rate; changes affect subsequent ticks.
	Delta time.Duration
	// Paused stops fixed ticks. Frame-phase systems, input, and rendering
	// continue; see the game's Pause documentation.
	Paused bool
	// Scale is the rate at which elapsed time accumulates toward fixed
	// ticks: 1 is real time, 0.5 is half speed, 2 is fast forward. It must
	// be a finite positive number; use Pause to stop.
	Scale float64
}

// NewSimulation returns a running simulation with the given fixed tick
// interval, at scale 1, starting at tick 0.
func NewSimulation(delta time.Duration) *Simulation {
	return &Simulation{Delta: delta, Scale: 1}
}

// Elapsed returns total simulation time as a duration: Tick ticks of Delta.
// It is the world-time view of the tick counter; ticks are the
// deterministic unit, this is the human-scale one.
func (s *Simulation) Elapsed() time.Duration {
	return time.Duration(s.Tick) * s.Delta
}

// Pause stops fixed updates. The accumulator keeps its value; Resume
// continues from where the simulation stopped.
func (s *Simulation) Pause() {
	s.Paused = true
}

// Resume clears the paused state. The next advance continues fixed updates
// from the accumulator position preserved by Pause.
func (s *Simulation) Resume() {
	s.Paused = false
}

// SetScale sets the rate at which elapsed time accumulates toward fixed
// ticks. It returns an error for a scale of zero or less, NaN, or infinity;
// use Pause to stop the simulation.
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
