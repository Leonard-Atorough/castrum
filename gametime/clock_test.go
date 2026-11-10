package gametime

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

func newClockHarness(t *testing.T) (*core.World, core.System) {
	t.Helper()
	world := core.NewWorld()
	return world, NewClockSystem()
}

func tickClock(t *testing.T, world *core.World, sys core.System, number uint64, dt time.Duration) {
	t.Helper()
	if err := sys.Update(&core.Context{World: world, Tick: number, DeltaTime: dt}); err != nil {
		t.Fatalf("clock tick %d: %v", number, err)
	}
}

func clockOf(t *testing.T, world *core.World) *Clock {
	t.Helper()
	c, err := world.Resource[*Clock]()
	if err != nil {
		t.Fatalf("resolve clock: %v", err)
	}
	return c
}

func TestNewSystemProvisionsClock(t *testing.T) {
	world, sys := newClockHarness(t)
	tickClock(t, world, sys, 1, time.Second)
	c := clockOf(t, world)
	if c == nil {
		t.Fatal("provisioned clock is nil")
	}
	if c.Tick != 1 {
		t.Fatalf("Tick = %d, want 1", c.Tick)
	}
	if c.Paused {
		t.Fatal("new clock is paused")
	}
	if c.Scale != 1 {
		t.Fatalf("Scale = %v, want 1", c.Scale)
	}
}

func TestNewSystemUsesProvidedClock(t *testing.T) {
	world := core.NewWorld()
	c := &Clock{Tick: 41, Scale: 1}
	if err := world.Provide(func(*core.World) (*Clock, error) {
		return c, nil
	}); err != nil {
		t.Fatalf("provide clock: %v", err)
	}
	sys := NewClockSystem()
	tickClock(t, world, sys, 1, time.Second)
	if c.Tick != 42 {
		t.Fatalf("Tick = %d, want 42", c.Tick)
	}
	if clockOf(t, world) != c {
		t.Fatal("system did not use the provided clock instance")
	}
}

func TestClockValidate(t *testing.T) {
	valid := []*Clock{
		NewClock(),
		{Scale: 0},
		{Scale: 0.5},
		{Scale: 2},
		{Tick: 100, Paused: true, Scale: 1},
	}
	for _, c := range valid {
		if err := c.Validate(); err != nil {
			t.Fatalf("valid Clock rejected: %v", err)
		}
	}

	for name, c := range map[string]*Clock{
		"negative scale": {Scale: -1},
		"NaN scale":      {Scale: math.NaN()},
		"infinite scale": {Scale: math.Inf(1)},
	} {
		if err := c.Validate(); err == nil {
			t.Fatalf("%s Clock accepted", name)
		}
	}
}

func TestSystemRejectsInvalidScale(t *testing.T) {
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Clock, error) {
		return &Clock{Scale: -1}, nil
	}); err != nil {
		t.Fatalf("provide clock: %v", err)
	}
	sys := NewClockSystem()
	err := sys.Update(&core.Context{World: world, Tick: 1})
	if err == nil {
		t.Fatal("system accepted a negative scale")
	}
	if !strings.Contains(err.Error(), "scale") {
		t.Fatalf("error does not mention scale: %v", err)
	}
}

func TestAdvanceCountsTicks(t *testing.T) {
	world, sys := newClockHarness(t)
	for i := uint64(1); i <= 5; i++ {
		tickClock(t, world, sys, i, time.Second)
	}
	if c := clockOf(t, world); c.Tick != 5 {
		t.Fatalf("Tick = %d, want 5", c.Tick)
	}
}

func TestPausedClockDoesNotAdvance(t *testing.T) {
	world, sys := newClockHarness(t)
	tickClock(t, world, sys, 1, time.Second)
	c := clockOf(t, world)
	c.Paused = true
	tickClock(t, world, sys, 2, time.Second)
	tickClock(t, world, sys, 3, time.Second)
	if c.Tick != 1 {
		t.Fatalf("Tick = %d, want 1", c.Tick)
	}
	c.Paused = false
	tickClock(t, world, sys, 4, time.Second)
	if c.Tick != 2 {
		t.Fatalf("Tick = %d, want 2 after resume", c.Tick)
	}
}

func TestZeroScaleStopsClock(t *testing.T) {
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Clock, error) {
		return &Clock{Scale: 0}, nil
	}); err != nil {
		t.Fatalf("provide clock: %v", err)
	}
	sys := NewClockSystem()
	for i := uint64(1); i <= 3; i++ {
		tickClock(t, world, sys, i, time.Second)
	}
	if c := clockOf(t, world); c.Tick != 0 {
		t.Fatalf("Tick = %d, want 0", c.Tick)
	}
}

func TestFractionalScaleAccumulates(t *testing.T) {
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Clock, error) {
		return &Clock{Scale: 0.5}, nil
	}); err != nil {
		t.Fatalf("provide clock: %v", err)
	}
	sys := NewClockSystem()
	for i := uint64(1); i <= 4; i++ {
		tickClock(t, world, sys, i, time.Second)
	}
	// One game tick every other simulation tick.
	if c := clockOf(t, world); c.Tick != 2 {
		t.Fatalf("Tick = %d, want 2", c.Tick)
	}
}

func TestDoubleScaleAdvancesTwoPerTick(t *testing.T) {
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*Clock, error) {
		return &Clock{Scale: 2}, nil
	}); err != nil {
		t.Fatalf("provide clock: %v", err)
	}
	sys := NewClockSystem()
	for i := uint64(1); i <= 3; i++ {
		tickClock(t, world, sys, i, time.Second)
	}
	if c := clockOf(t, world); c.Tick != 6 {
		t.Fatalf("Tick = %d, want 6", c.Tick)
	}
}

func TestScaleChangeTakesEffectImmediately(t *testing.T) {
	world, sys := newClockHarness(t)
	tickClock(t, world, sys, 1, time.Second)
	c := clockOf(t, world)
	c.Scale = 3
	tickClock(t, world, sys, 2, time.Second)
	if c.Tick != 4 {
		t.Fatalf("Tick = %d, want 4", c.Tick)
	}
}

func TestAdvanceDirect(t *testing.T) {
	c := NewClock()
	if ticks := c.advance(); ticks != 1 {
		t.Fatalf("advance = %d, want 1", ticks)
	}
	c.Paused = true
	if ticks := c.advance(); ticks != 0 {
		t.Fatalf("paused advance = %d, want 0", ticks)
	}
	c.Paused = false
	c.Scale = 0.25
	for i := 0; i < 3; i++ {
		if ticks := c.advance(); ticks != 0 {
			t.Fatalf("advance %d = %d, want 0", i, ticks)
		}
	}
	if ticks := c.advance(); ticks != 1 {
		t.Fatalf("final advance = %d, want 1", ticks)
	}
	if c.Tick != 2 {
		t.Fatalf("Tick = %d, want 2", c.Tick)
	}
}
