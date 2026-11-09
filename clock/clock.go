// Package clock provides the engine's virtual game time as a [Clock]
// resource. [NewSystem] advances it on fixed ticks, independently of
// [core.Context.Tick], so game time can be paused, scaled, and restored
// without touching the fixed-timestep contract. The clock carries no
// calendar or cycle structure; games derive such meaning from its ticks.
package clock

import (
	"errors"
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
)

// Clock is the engine's virtual game time. It advances only via
// [NewSystem], one fixed tick at a time, and all of its state that matters
// across a save and restore is exported and value-shaped.
type Clock struct {
	// Tick counts elapsed game ticks. It is independent of
	// [core.Context.Tick] so pausing or restoring game time never touches
	// the fixed-timestep contract.
	Tick uint64
	// Paused stops the clock from accumulating time. Unlike the game's
	// fixed-loop pause, a paused clock keeps fixed systems running.
	Paused bool
	// Scale is the clock's relative speed: 1 advances one game tick per
	// fixed tick, 2 advances two, 0.5 accumulates one game tick every other
	// fixed tick, and 0 stops the clock. It must be a finite
	// non-negative number.
	Scale float64

	// carry holds the fractional remainder of the last scaled advance. It
	// is not snapshot-relevant: a partially accumulated fraction does not
	// matter after a restore.
	carry float64
}

// New returns a running clock at scale 1, starting at tick 0.
func New() *Clock {
	return &Clock{Scale: 1}
}

// Validate reports an error if Scale is negative, NaN, or infinite.
func (c *Clock) Validate() error {
	if math.IsNaN(c.Scale) {
		return fmt.Errorf("clock scale must not be NaN")
	}
	if math.IsInf(c.Scale, 0) {
		return fmt.Errorf("clock scale must be finite, got %v", c.Scale)
	}
	if c.Scale < 0 {
		return fmt.Errorf("clock scale must not be negative, got %v", c.Scale)
	}
	return nil
}

// advance moves the clock forward by Scale once and reports the number of
// game ticks accumulated. It assumes Validate has passed.
func (c *Clock) advance() uint64 {
	if c.Paused {
		return 0
	}
	c.carry += c.Scale
	whole := math.Floor(c.carry)
	c.carry -= whole
	if whole < 0 {
		return 0
	}
	ticks := uint64(whole)
	c.Tick += ticks
	return ticks
}

// SystemName is the name used to register the clock system in the fixed
// schedule.
const SystemName = "engine.clock"

// NewSystem returns the fixed-tick system that provisions the *Clock
// resource and advances it by Scale each simulation tick. A paused or
// zero-scaled clock accumulates nothing.
func NewSystem() core.System {
	return &system{}
}

type system struct {
	clock *Clock
}

func (s *system) Update(ctx *core.Context) error {
	if s.clock == nil {
		clock, err := ctx.World.Resource[*Clock]()
		if err != nil {
			if !errors.Is(err, core.ErrResourceNotRegistered) {
				return fmt.Errorf("clock system: resolve clock: %w", err)
			}
			if err := ctx.World.Provide(func(*core.World) (*Clock, error) {
				return New(), nil
			}); err != nil {
				return fmt.Errorf("clock system: provide clock: %w", err)
			}
			if clock, err = ctx.World.Resource[*Clock](); err != nil {
				return fmt.Errorf("clock system: resolve clock: %w", err)
			}
		}
		s.clock = clock
	}
	if err := s.clock.Validate(); err != nil {
		return fmt.Errorf("clock system: %w", err)
	}
	s.clock.advance()
	return nil
}
