package core

import (
	"fmt"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
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

// Context is the state systems run with: the world, the clocks, and
// the input the engine published this frame. The engine and the
// runner write these fields; game systems read them. Treat the
// context as read-only inside systems.
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
	// LogicalWidth is the render target's internal width in pixels.
	// Read along with LogicalHeight to determine the render target's
	// internal resolution.
	LogicalWidth int
	// LogicalHeight is the render target's internal height in pixels.
	// Read along with LogicalWidth to determine the render target's
	// internal resolution.
	LogicalHeight int
	// Input is the current snapshot of all input devices, published
	// by the runner each frame. nil means no runner published input -
	// the headless case - and every read on a nil snapshot returns
	// zero.
	Input *input.Snapshot
	// Actions is the game's action map, published by the engine when
	// WithBindings configures one. It is the same instance the
	// ActionMap resource resolves: mutating through either handle
	// affects the one map. Queries on a nil Actions read zero, so
	// headless games and systems running without bindings need no
	// guards.
	Actions *input.ActionMap
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
