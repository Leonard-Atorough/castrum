package animation

import "github.com/Leonard-Atorough/castrum/core"

// Started is emitted by [Animate] when an [Animator] begins playing a state.
type Started struct {
	// Entity is the entity whose state started.
	Entity core.Entity
	// Tick is the fixed tick of the start.
	Tick uint64
	// State is the name of the state that started.
	State string
}

// Completed is emitted by [Animate] when a state's primary track reaches a
// boundary. It emits one signal per animator per tick, even if a looping state
// wraps multiple times in one tick.
type Completed struct {
	// Entity is the entity whose primary track reached a boundary.
	Entity core.Entity
	// Tick is the fixed tick of the completion.
	Tick uint64
	// State is the name of the state whose primary track reached a boundary.
	State string
	// Loops is the number of forward wraps for [Loop], or 1 for other modes.
	Loops uint64
}
