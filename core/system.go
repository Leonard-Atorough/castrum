package core

import (
	"fmt"
	"time"

	"github.com/Leonard-Atorough/castrum/input"
)

// Phase identifies when a system runs.
type Phase int8

const (
	// PhasePreStartup runs once after eager resources resolve and before startup systems.
	PhasePreStartup Phase = iota
	// PhaseStartup runs once after pre-startup systems.
	PhaseStartup
	// PhasePostStartup runs once after startup systems.
	PhasePostStartup
	// PhasePreFrame runs once per display frame after input updates and before frame systems.
	PhasePreFrame
	// PhaseFrame runs once per display frame, before fixed ticks.
	// Input consumption belongs here.
	PhaseFrame
	// PhasePostFrame runs once per display frame after frame systems.
	PhasePostFrame
	// PhasePreFixed runs once per fixed tick before fixed systems.
	PhasePreFixed
	// PhaseFixed runs at the fixed simulation rate. Movement,
	// physics, and timers belong here.
	PhaseFixed
	// PhasePostFixed runs once per fixed tick after fixed systems.
	PhasePostFixed
)

// String returns the phase's schedule name, or a formatted value for an unknown phase.
func (s Phase) String() string {
	names := [...]string{
		"pre-startup",
		"startup",
		"post-startup",
		"pre-frame",
		"frame",
		"post-frame",
		"pre-fixed",
		"fixed",
		"post-fixed",
	}
	if s < 0 || int(s) >= len(names) {
		return fmt.Sprintf("schedule(%d)", int8(s))
	}
	return names[s]
}

// Context gives systems access to the world, simulation clocks, and input
// published for the current frame. The engine and runner populate it; systems
// should treat it as read-only.
type Context struct {
	// World is the game world.
	World *World
	// Tick counts fixed simulation steps since startup.
	Tick uint64
	// Frame counts display frames since startup.
	Frame uint64
	// DeltaTime is elapsed frame time in [PhasePreFrame], [PhaseFrame], or
	// [PhasePostFrame], or the fixed tick interval in [PhasePreFixed],
	// [PhaseFixed], or [PhasePostFixed].
	DeltaTime time.Duration
	// Alpha is the fixed-loop remainder for interpolation between ticks.
	// A runner sets it when running draw systems.
	Alpha float64
	// LogicalWidth is the render target's internal width in pixels.
	LogicalWidth int
	// LogicalHeight is the render target's internal height in pixels.
	LogicalHeight int
	// Input is the runner's current input snapshot. A nil snapshot reads as
	// no input.
	Input *input.Snapshot
	// Actions is the game's action map when configured with `WithBindings`.
	// It is the same instance as the ActionMap resource; a nil map reads as
	// no actions.
	Actions *input.ActionMap
}

// System is a unit of game logic that updates state during a schedule phase.
type System interface {
	Update(ctx *Context) error
}

// SystemFunc adapts a function to the [System] interface.
type SystemFunc func(ctx *Context) error

// Update calls f with ctx.
func (f SystemFunc) Update(ctx *Context) error {
	return f(ctx)
}
