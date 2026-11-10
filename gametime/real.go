// This file holds the engine's real-time resource: [Real], wall-clock
// elapsed time as world data. Real time is unaffected by pause, time
// scale, and the frame clamp; it is the time of things that stay true
// while the game is paused - menu animations, playtime counters, session
// telemetry. Gameplay logic should read [Simulation] instead: real time
// is frame-rate dependent and therefore not deterministic.

package gametime

import "time"

// Real is the engine's wall-clock time resource: real elapsed time since
// the game started, the current advance's real delta, and the frame clamp
// policy. The engine provisions it when the game is constructed and
// updates it on every advance, before pause and clamp handling.
//
// Real time cannot be paused or scaled; that is its purpose. All fields
// are exported and value-shaped for snapshot and restore.
type Real struct {
	// Elapsed is total real time since the first advance. It accumulates
	// raw elapsed time, unaffected by [Real.MaxFrameTime], pause, or scale.
	Elapsed time.Duration
	// Delta is the real elapsed time of the current advance, before
	// clamping. Frame-phase [core.Context.DeltaTime] is this value clamped
	// by MaxFrameTime.
	Delta time.Duration
	// MaxFrameTime is the maximum elapsed time one advance may contribute
	// to the fixed-loop accumulator. Larger advances are clamped to it.
	// It is wall-clock ingestion policy; real time itself is unclamped.
	MaxFrameTime time.Duration
}

// NewReal returns a real-time resource with the given frame clamp.
func NewReal(maxFrameTime time.Duration) *Real {
	return &Real{MaxFrameTime: maxFrameTime}
}
