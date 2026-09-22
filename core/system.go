package core

import (
	"fmt"
	"time"
)

// Phase identifies when a system runs.
type Phase int8

const (
	// PhaseStartup runs once, before the first frame.
	PhaseStartup Phase = iota
	// PhaseFrame runs once per display frame, before fixed ticks.
	// Timers and input consumption belong here.
	PhaseFrame
	// PhaseFixed runs at the fixed simulation rate. Movement and
	// physics belong here.
	PhaseFixed
)

// Note: could try using stringer here
func (s Phase) String() string {
	switch s {
	case PhaseStartup:
		return "startup"
	case PhaseFrame:
		return "frame"
	case PhaseFixed:
		return "fixed"
	default:
		return fmt.Sprintf("schedule(%d)", int8(s))
	}
}

type Context struct {
	// World represents the current world instance.
	World *World
	// Tick counts fixed simulation steps since startup.
	Tick uint64
	// Frame counts display frames since startup.
	Frame uint64
	// DeltaTime is the current schedule's step: elapsed frame time in
	// PhaseFrame, the fixed tick interval in PhaseFixed.
	DeltaTime time.Duration
	// Alpha is the fixed-loop remainder over the tick interval, for
	// interpolating between the last two ticks. Only a runner sets it,
	// when running its draw systems.
	Alpha float64
}

// System is a unit of game logic. It runs during a specific phase
// and updates the game state.
type System interface {
	Update(ctx *Context) error
}

// SystemFunc is an adapter to allow the use of ordinary functions as systems.
type SystemFunc func(ctx *Context) error

// Update calls the system function with the given context.
func (f SystemFunc) Update(ctx *Context) error {
	return f(ctx)
}
