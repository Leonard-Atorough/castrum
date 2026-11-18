package gametime

import "time"

// Real tracks the elapsed time supplied to the engine on each advance.
//
// The engine updates it even while paused, without applying time scaling or
// the frame clamp. Use [Simulation] for fixed-step simulation time.
type Real struct {
	// Frame counts advances, including those made while paused.
	Frame uint64 `json:"frame"`
	// Elapsed is the sum of elapsed times supplied to advances.
	Elapsed time.Duration `json:"elapsed"`
	// Delta is the elapsed time supplied to the most recent advance.
	Delta time.Duration `json:"delta"`
	// MaxFrameTime limits elapsed time used for frame updates and, before
	// scaling, the fixed-loop accumulator.
	MaxFrameTime time.Duration `json:"maxFrameTime"`
}

// NewReal returns a real-time resource with the given frame clamp.
func NewReal(maxFrameTime time.Duration) *Real {
	return &Real{MaxFrameTime: maxFrameTime}
}
