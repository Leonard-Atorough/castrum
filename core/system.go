package core

import (
	"fmt"
	"time"
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
	// PhasePreFrame runs once per display frame before frame systems.
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

// Context gives systems access to the world and timing for the current phase.
// The game populates its fields; systems should not modify them. The world
// itself remains writable.
type Context struct {
	// World is the game world.
	World *World
	// Tick counts fixed simulation steps since startup.
	Tick uint64
	// DeltaTime is capped elapsed frame time in [PhasePreFrame], [PhaseFrame],
	// or [PhasePostFrame], or the fixed tick interval in [PhasePreFixed],
	// [PhaseFixed], or [PhasePostFixed].
	DeltaTime time.Duration
}

// SystemFunc runs game logic during a schedule phase.
//
// A non-nil error stops the schedule and is returned by the game.
type SystemFunc func(ctx *Context) error
