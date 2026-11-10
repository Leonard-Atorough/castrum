package gametime

import "time"

// Real tracks the elapsed time supplied to the engine on each advance.
//
// The engine updates it even while paused, without applying time scaling or
// the frame clamp. Use [Simulation] for fixed-step simulation time.
type Real struct {
	// Frame counts advances, including those made while paused.
	Frame uint64
	// Elapsed is the sum of elapsed times supplied to advances.
	Elapsed time.Duration
	// Delta is the elapsed time supplied to the most recent advance.
	Delta time.Duration
	// MaxFrameTime limits elapsed time used for frame updates and, before
	// scaling, the fixed-loop accumulator.
	MaxFrameTime time.Duration
}

// NewReal returns a real-time resource with the given frame clamp.
func NewReal(maxFrameTime time.Duration) *Real {
	return &Real{MaxFrameTime: maxFrameTime}
}
